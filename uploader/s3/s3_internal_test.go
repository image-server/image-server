package s3

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	. "github.com/image-server/image-server/test"
)

// useFakeS3 points the package client at a local HTTP handler (path-style,
// static credentials) with the retry policy Initialize uses.
func useFakeS3(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	cfg := aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider("key", "secret", ""),
		// Production policy, minus the backoff wait.
		Retryer: func() aws.Retryer { return retry.AddWithMaxBackoffDelay(retryer(), time.Millisecond) },
	}
	initialize("test-bucket", cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(srv.URL)
		o.UsePathStyle = true
	})
}

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "image.jpg")
	Ok(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestUploadPutsPublicObject(t *testing.T) {
	var got *http.Request
	var body []byte
	useFakeS3(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		body, _ = io.ReadAll(r.Body)
	})

	err := (&Uploader{}).Upload(writeFile(t, "jpeg bytes"), "ns/6da/b5f/6d8/d4bd/w300.jpg", "image/jpeg")
	Ok(t, err)

	Equals(t, http.MethodPut, got.Method)
	Equals(t, "/test-bucket/ns/6da/b5f/6d8/d4bd/w300.jpg", got.URL.Path)
	Equals(t, "image/jpeg", got.Header.Get("Content-Type"))
	Equals(t, "public-read", got.Header.Get("X-Amz-Acl"))
	Equals(t, "jpeg bytes", string(body))
}

func TestUploadRetriesServerErrors(t *testing.T) {
	var attempts int32
	useFakeS3(t, func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		if atomic.AddInt32(&attempts, 1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	Ok(t, (&Uploader{}).Upload(writeFile(t, "x"), "k.jpg", "image/jpeg"))
	Equals(t, int32(3), atomic.LoadInt32(&attempts))
}

func TestUploadGivesUpAfterMaxAttempts(t *testing.T) {
	var attempts int32
	useFakeS3(t, func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	err := (&Uploader{}).Upload(writeFile(t, "x"), "k.jpg", "image/jpeg")
	Assert(t, err != nil, "expected an error after retries")
	Equals(t, int32(maxAttempts), atomic.LoadInt32(&attempts))
}

func TestUploadMissingSourceFile(t *testing.T) {
	useFakeS3(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request expected")
	})
	err := (&Uploader{}).Upload(filepath.Join(t.TempDir(), "missing.jpg"), "k.jpg", "image/jpeg")
	Assert(t, os.IsNotExist(err), "expected not-exist error, got %v", err)
}

// ListDirectory follows continuation tokens and returns base names.
func TestListDirectoryPaginates(t *testing.T) {
	var prefixes []string
	useFakeS3(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		prefixes = append(prefixes, q.Get("prefix"))
		w.Header().Set("Content-Type", "application/xml")
		if q.Get("continuation-token") == "" {
			fmt.Fprint(w, `<ListBucketResult><IsTruncated>true</IsTruncated><NextContinuationToken>page2</NextContinuationToken>`+
				`<Contents><Key>ns/abc/original</Key></Contents><Contents><Key>ns/abc/w300.jpg</Key></Contents></ListBucketResult>`)
			return
		}
		fmt.Fprint(w, `<ListBucketResult><IsTruncated>false</IsTruncated>`+
			`<Contents><Key>ns/abc/w600.webp</Key></Contents></ListBucketResult>`)
	})

	names, err := (&Uploader{}).ListDirectory("ns/abc")
	Ok(t, err)
	Equals(t, []string{"original", "w300.jpg", "w600.webp"}, names)
	Equals(t, []string{"ns/abc", "ns/abc"}, prefixes)
}

func TestListDirectoryError(t *testing.T) {
	useFakeS3(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `<Error><Code>AccessDenied</Code><Message>denied</Message></Error>`)
	})
	_, err := (&Uploader{}).ListDirectory("ns/abc")
	Assert(t, err != nil, "expected AccessDenied error")
}
