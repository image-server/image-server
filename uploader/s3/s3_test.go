package s3_test

import (
	"log"
	"os"
	"testing"

	. "github.com/image-server/image-server/test"
	"github.com/image-server/image-server/uploader/s3"
)

func TestItemToHash(t *testing.T) {
	if !hasAwsAuthentication() {
		return
	}

	bucketName := os.Getenv("AWS_BUCKET")
	regionName := os.Getenv("AWS_REGION")

	s3.Initialize(bucketName, regionName)

	uploader := s3.Uploader{}

	existing, err := uploader.ListDirectory("p/543/47c/442/1c41f9467a3f5afed64943b")
	Ok(t, err)
	log.Println(existing)

	// Equals(t, "6ad5544baa6f5e852e1af26f8c2e45db", image.ToHash())
}

func hasAwsAuthentication() bool {
	hasRegion := len(os.Getenv("AWS_REGION")) > 0
	hasBucket := len(os.Getenv("AWS_BUCKET")) > 0
	return hasRegion && hasBucket
}
