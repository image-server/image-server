package core

import (
	"strings"
	"time"

	"github.com/image-server/image-server/core/signature"
)

// ServerConfiguration struct
type ServerConfiguration struct {
	AllowedExtensions    []string
	MaximumWidth         int
	LocalBasePath        string
	RemoteBasePath       string
	RemoteBaseURL        string
	DefaultQuality       uint
	UploaderConcurrency  uint
	ProcessorConcurrency uint
	HTTPTimeout          time.Duration
	Adapters             *Adapters
	Outputs              string
	AWSAccessKeyID       string
	AWSSecretKey         string
	AWSBucket            string
	AWSRegion            string
	UploaderType         string
	CleanUpTicker        *time.Ticker
	MaxFileAge           time.Duration

	// Signature validation
	SignatureConfig *signature.Config
}

func (sc *ServerConfiguration) UploaderIsAws() bool {
	uploader := strings.ToLower(sc.UploaderType)
	if uploader == "aws" || uploader == "s3" {
		return true
	}
	return false
}
