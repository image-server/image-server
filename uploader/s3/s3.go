package s3

import (
	"context"
	"os"
	"path/filepath"
	"time"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/golang/glog"
)

// Uploader for S3
type Uploader struct {
}

var manager *s3manager.Uploader
var svc *s3.S3
var bucket string

// Upload copies a file int a bucket in S3
func (u *Uploader) Upload(source string, destination string, contType string) error {
	reader, err := os.Open(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	timeout := 4 * time.Minute

	ctx := context.Background()
	var cancelFn func()
	if timeout > 0 {
		ctx, cancelFn = context.WithTimeout(ctx, timeout)
	}
	// Ensure the context is canceled to prevent leaking.
	// See context package for more information, https://golang.org/pkg/context/
	defer cancelFn()

	// Uploads the object to S3. The Context will interrupt the request if the
	// timeout expires.
	_, err = manager.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(destination),
		ContentType: aws.String(contType),
		Body:        reader,
		ACL:         aws.String("public-read"),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == request.CanceledErrorCode {
			// If the SDK can determine the request or retry delay was canceled
			// by a context the CanceledErrorCode error code will be returned.
			glog.Infof("AWS S3 upload canceled due to timeout destination: %s, Error: %v\n", destination, err)
		}
	}

	return err
}

func (u *Uploader) ListDirectory(directory string) ([]string, error) {
	var names []string
	prefix := directory
	delim := ""
	marker := ""
	var max int64 = 1000
	resp, err := svc.ListObjects(&s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(delim),
		Marker:    aws.String(marker),
		MaxKeys:   aws.Int64(max),
	})
	if err == nil {
		entries := resp.Contents
		for _, entry := range entries {
			name := filepath.Base(aws.StringValue(entry.Key))
			names = append(names, name)
		}
	}
	return names, err
}

// CreateDirectory does nothing since a directory does not need to be created on S3
// Directories are virtual, and defined by the path of the object
func (u *Uploader) CreateDirectory(path string) error {
	return nil
}

func Initialize(bucketName string, regionName string) {
	// Initial credentials loaded from SDK's default credential chain. Such as
	// the environment, shared credentials (~/.aws/credentials), or EC2 Instance
	// Role. These credentials will be used to to make the STS Assume Role API.
	sess := session.Must(session.NewSession(&aws.Config{
		Region:     aws.String(regionName),
		MaxRetries: aws.Int(4),
	}))
	manager = s3manager.NewUploader(sess)
	svc = s3.New(sess)

	bucket = bucketName
}
