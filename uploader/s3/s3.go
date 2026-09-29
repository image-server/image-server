package s3

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Uploader for S3
type Uploader struct {
}

var svc *s3.Client
var bucket string

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
	_, err = svc.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(destination),
		ContentType: aws.String(contType),
		Body:        reader,
		ACL:         types.ObjectCannedACLPublicRead,
	})
	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		log.Printf("AWS S3 upload canceled due to timeout destination: %s, Error: %v\n", destination, err)
	}

	return err
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
func Initialize(bucketName string, regionName string) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(regionName),
		config.WithRetryer(retryer),
	)
	if err != nil {
		log.Fatalf("AWS S3 configuration failed: %v", err)
	}
	initialize(bucketName, cfg)
}

func retryer() aws.Retryer {
	return retry.AddWithMaxAttempts(retry.NewStandard(), maxAttempts)
}

func initialize(bucketName string, cfg aws.Config, optFns ...func(*s3.Options)) {
	svc = s3.NewFromConfig(cfg, optFns...)
	bucket = bucketName
}
