package core

import (
	"regexp"
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
	AWSUsePathStyle      bool
	AWSSkipACL           bool
	// S3Reads reads files missing locally from the bucket with the S3
	// credentials, instead of over HTTP from RemoteBaseURL
	S3Reads bool
	// CacheControl and CacheTagHeader are set on successful image reads
	// when not empty; the cache tag value is the namespace
	CacheControl   string
	CacheTagHeader string
	UploaderType   string
	CleanUpTicker  *time.Ticker
	MaxFileAge     time.Duration

	// Signature validation
	SignatureConfig *signature.Config

	// NamespaceAdmin is nil unless --allow-namespace-admin is set
	NamespaceAdmin *NamespaceAdminConfiguration
}

// NamespaceAdminConfiguration holds the settings for renaming and deleting
// namespaces
type NamespaceAdminConfiguration struct {
	// Secrets sign admin requests only; upload secrets are not accepted
	Secrets []string
	MaxTTL  time.Duration
	// DeletePattern must match the whole namespace name for a delete; nil
	// disables deleting
	DeletePattern *regexp.Regexp
}

func (sc *ServerConfiguration) UploaderIsAws() bool {
	uploader := strings.ToLower(sc.UploaderType)
	if uploader == "aws" || uploader == "s3" {
		return true
	}
	return false
}
