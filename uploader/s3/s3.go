package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// Uploader for S3
type Uploader struct {
}

// Options configures the client for S3-compatible stores
type Options struct {
	// UsePathStyle puts the bucket in the URL path (endpoint/bucket/key)
	// instead of the host name, as most S3-compatible stores expect
	UsePathStyle bool
	// SkipACL leaves out the public-read ACL on uploads, for private buckets
	// and stores that reject ACLs
	SkipACL bool
	// DeleteWorkers is how many delete requests run at once; 0 keeps the
	// default
	DeleteWorkers int
}

// ErrNotFound is returned by Download when the key does not exist
var ErrNotFound = errors.New("not found in the bucket")

var svc *s3.Client
var bucket string
var skipACL bool

// Same budget as the SDK v1 setup: MaxRetries 4 means 5 attempts in total.
const maxAttempts = 5

const uploadTimeout = 4 * time.Minute

// Upload copies a file int a bucket in S3
func (u *Uploader) Upload(source string, destination string, contType string) error {
	reader, err := os.Open(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	// The context interrupts the request (and its retries) when the timeout expires.
	ctx, cancel := context.WithTimeout(context.Background(), uploadTimeout)
	defer cancel()

	// A single PUT: images are far below the 5 GB limit, and the file is
	// seekable, so the SDK can rewind it to sign and retry.
	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(destination),
		ContentType: aws.String(contType),
		Body:        reader,
	}
	if !skipACL {
		input.ACL = types.ObjectCannedACLPublicRead
	}
	_, err = svc.PutObject(ctx, input)
	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		log.Printf("AWS S3 upload canceled due to timeout destination: %s, Error: %v\n", destination, err)
	}

	return err
}

// Download writes the object at key to destination, using the client's
// credentials. It returns ErrNotFound when the key does not exist. The file
// appears complete or not at all.
func (u *Uploader) Download(key string, destination string) error {
	ctx, cancel := context.WithTimeout(context.Background(), uploadTimeout)
	defer cancel()

	out, err := svc.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return fmt.Errorf("%w: %s", ErrNotFound, key)
		}
		return err
	}
	defer out.Body.Close()

	if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(destination), ".download-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, out.Body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), destination)
}

func (u *Uploader) ListDirectory(directory string) ([]string, error) {
	var names []string
	paginator := s3.NewListObjectsV2Paginator(svc, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(directory),
		MaxKeys: aws.Int32(1000),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return names, err
		}
		for _, entry := range page.Contents {
			names = append(names, filepath.Base(aws.ToString(entry.Key)))
		}
	}
	return names, nil
}

// CreateDirectory does nothing since a directory does not need to be created on S3
// Directories are virtual, and defined by the path of the object
func (u *Uploader) CreateDirectory(path string) error {
	return nil
}

// Initialize loads credentials from the SDK's default chain: environment,
// shared config/credentials (~/.aws), web identity, ECS/EC2 instance role.
// AWS_ENDPOINT_URL / AWS_ENDPOINT_URL_S3 point it at an S3-compatible store.
func Initialize(bucketName string, regionName string, opts Options) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(regionName),
		config.WithRetryer(retryer),
	)
	if err != nil {
		log.Fatalf("AWS S3 configuration failed: %v", err)
	}
	skipACL = opts.SkipACL
	if opts.DeleteWorkers > 0 {
		deleteWorkers = opts.DeleteWorkers
	}
	initialize(bucketName, cfg, func(o *s3.Options) {
		o.UsePathStyle = opts.UsePathStyle
	})
}

func retryer() aws.Retryer {
	return retry.AddWithMaxAttempts(retry.NewStandard(), maxAttempts)
}

func initialize(bucketName string, cfg aws.Config, optFns ...func(*s3.Options)) {
	svc = s3.NewFromConfig(cfg, optFns...)
	bucket = bucketName
}

// deleteBatch is the most keys one DeleteObjects request may delete, and
// deleteWorkers how many requests run at once
var (
	deleteBatch   = 1000
	deleteWorkers = 8
)

// ListKeys returns every key under prefix
func (u *Uploader) ListKeys(prefix string) ([]string, error) {
	var keys []string
	paginator := s3.NewListObjectsV2Paginator(svc, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(1000),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return keys, err
		}
		for _, entry := range page.Contents {
			keys = append(keys, aws.ToString(entry.Key))
		}
	}
	return keys, nil
}

// Copy copies the object at src to dst in the same bucket, keeping its
// content type
func (u *Uploader) Copy(src string, dst string) error {
	ctx, cancel := context.WithTimeout(context.Background(), uploadTimeout)
	defer cancel()

	segments := strings.Split(bucket+"/"+src, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(dst),
		CopySource: aws.String(strings.Join(segments, "/")),
	}
	// A copy does not keep the source's ACL
	if !skipACL {
		input.ACL = types.ObjectCannedACLPublicRead
	}
	_, err := svc.CopyObject(ctx, input)
	return err
}

// DeleteKeys deletes keys in batches of at most deleteBatch, sent
// deleteWorkers at a time. Keys are split into at least deleteWorkers
// batches, since stores may delete the keys of one batch one by one. Keys
// that do not exist are not an error.
func (u *Uploader) DeleteKeys(keys []string) error {
	size := (len(keys) + deleteWorkers - 1) / deleteWorkers
	if size > deleteBatch {
		size = deleteBatch
	}
	if size < 1 {
		size = 1
	}
	var batches [][]string
	for start := 0; start < len(keys); start += size {
		end := start + size
		if end > len(keys) {
			end = len(keys)
		}
		batches = append(batches, keys[start:end])
	}
	return parallel(len(batches), deleteWorkers, func(i int) error { return deleteBatchOf(batches[i]) })
}

func deleteBatchOf(keys []string) error {
	objects := make([]types.ObjectIdentifier, 0, len(keys))
	for _, key := range keys {
		objects = append(objects, types.ObjectIdentifier{Key: aws.String(key)})
	}

	ctx, cancel := context.WithTimeout(context.Background(), uploadTimeout)
	defer cancel()
	out, err := svc.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &types.Delete{Objects: objects, Quiet: aws.Bool(true)},
	})
	if err != nil {
		return err
	}
	for _, e := range out.Errors {
		if aws.ToString(e.Code) == "NoSuchKey" {
			continue
		}
		return fmt.Errorf("delete %s: %s: %s", aws.ToString(e.Key), aws.ToString(e.Code), aws.ToString(e.Message))
	}
	return nil
}

// parallel calls fn(0..n-1) with at most workers running at once. It stops
// starting new calls after the first error, and returns it.
func parallel(n, workers int, fn func(i int) error) error {
	var mu sync.Mutex
	var firstErr error
	next := 0
	var wg sync.WaitGroup
	for w := 0; w < workers && w < n; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				mu.Lock()
				if firstErr != nil || next >= n {
					mu.Unlock()
					return
				}
				i := next
				next++
				mu.Unlock()
				if err := fn(i); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	return firstErr
}

// PutBytes stores a small object
func (u *Uploader) PutBytes(key string, data []byte, contentType string) error {
	ctx, cancel := context.WithTimeout(context.Background(), uploadTimeout)
	defer cancel()
	_, err := svc.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		Body:        bytes.NewReader(data),
	})
	return err
}

// GetBytes reads a small object; found is false when the key does not exist
func (u *Uploader) GetBytes(key string) (data []byte, found bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), uploadTimeout)
	defer cancel()
	out, err := svc.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		if isNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer out.Body.Close()
	data, err = io.ReadAll(out.Body)
	return data, err == nil, err
}

func isNotFound(err error) bool {
	var noKey *types.NoSuchKey
	var apiErr smithy.APIError
	return errors.As(err, &noKey) || (errors.As(err, &apiErr) && apiErr.ErrorCode() == "NotFound")
}
