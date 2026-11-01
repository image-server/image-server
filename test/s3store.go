package test

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Store is a bucket on a real S3-compatible store, for integration tests
type S3Store struct {
	Bucket string
	Region string
	Client *s3.Client
}

// NewS3Store creates an empty bucket for the test and removes it afterwards.
// It skips the test unless S3_INTEGRATION is set; the store is configured
// with the standard AWS environment variables (AWS_ENDPOINT_URL_S3, keys,
// region), and is addressed path-style.
func NewS3Store(t *testing.T) *S3Store {
	t.Helper()
	if os.Getenv("S3_INTEGRATION") == "" {
		t.Skip("set S3_INTEGRATION=1 and the AWS_* variables to run against an S3-compatible store")
	}
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		t.Fatal(err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) { o.UsePathStyle = true })

	b := make([]byte, 6)
	rand.Read(b)
	store := &S3Store{Bucket: fmt.Sprintf("it-%x", b), Region: region, Client: client}
	if _, err := client.CreateBucket(context.Background(), &s3.CreateBucketInput{Bucket: aws.String(store.Bucket)}); err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	t.Cleanup(func() {
		for _, key := range store.Keys(t, "") {
			client.DeleteObject(context.Background(), &s3.DeleteObjectInput{Bucket: aws.String(store.Bucket), Key: aws.String(key)})
		}
		client.DeleteBucket(context.Background(), &s3.DeleteBucketInput{Bucket: aws.String(store.Bucket)})
	})
	return store
}

// Keys lists the keys under prefix
func (s *S3Store) Keys(t *testing.T, prefix string) []string {
	t.Helper()
	var keys []string
	p := s3.NewListObjectsV2Paginator(s.Client, &s3.ListObjectsV2Input{Bucket: aws.String(s.Bucket), Prefix: aws.String(prefix)})
	for p.HasMorePages() {
		page, err := p.NextPage(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, o := range page.Contents {
			keys = append(keys, aws.ToString(o.Key))
		}
	}
	return keys
}

// Head returns the object's metadata, or nil when it does not exist
func (s *S3Store) Head(t *testing.T, key string) *s3.HeadObjectOutput {
	t.Helper()
	out, err := s.Client.HeadObject(context.Background(), &s3.HeadObjectInput{Bucket: aws.String(s.Bucket), Key: aws.String(key)})
	if err != nil {
		var nf *types.NotFound
		if errors.As(err, &nf) {
			return nil
		}
		t.Fatal(err)
	}
	return out
}

// Put stores data at key
func (s *S3Store) Put(t *testing.T, key string, data []byte, contentType string) {
	t.Helper()
	_, err := s.Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(s.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		t.Fatal(err)
	}
}
