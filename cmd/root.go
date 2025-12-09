package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/core/signature"
	"github.com/image-server/image-server/fetcher/http"
	"github.com/image-server/image-server/logger/prometheus"
	"github.com/image-server/image-server/logger/statsd"
	"github.com/image-server/image-server/paths"
	"github.com/image-server/image-server/uploader"
	"github.com/spf13/cobra"
)

// configT collects all the global state of the logging setup.
type configT struct {
	port          string
	extensions    string
	localBasePath string

	remoteBaseURL  string
	remoteBasePath string

	namespace string
	outputs   string
	listen    string

	uploaderType string
	maxFileAge   int

	awsAccessKeyID string
	awsSecretKey   string
	awsBucket      string
	awsRegion      string

	maximumWidth   int
	defaultQuality int

	uploaderConcurrency  int
	processorConcurrency int
	httpTimeout          int
	gomaxprocs           int

	enableStatsd bool
	statsdHost   string
	statsdPort   int
	statsdPrefix string
	profile      bool

	version bool

	// Signature validation
	requireSignature        bool
	requireSignatureForRead bool
	signingSecretsFile      string
	signatureMaxTTL         int
}

var config configT

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "image-server",
	Short: "image-server is an image processing server",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func serverConfiguration() (*core.ServerConfiguration, error) {
	sc := serverConfigurationFromConfig()
	if config.enableStatsd {
		statsd.Enable(config.statsdHost, config.statsdPort, config.statsdPrefix)
	}
	prometheus.Enable()

	adapters := &core.Adapters{
		Fetcher: &http.Fetcher{},
		Paths:   &paths.Paths{LocalBasePath: sc.LocalBasePath, RemoteBasePath: sc.RemoteBasePath, RemoteBaseURL: sc.RemoteBaseURL},
	}
	sc.Adapters = adapters
	sc.CleanUpTicker = time.NewTicker(2 * time.Minute)

	return sc, nil
}

// serverConfigurationFromContext returns a core.ServerConfiguration initialized
// from command line flags or defaults.
// Command line flags preceding the Command (server, process, etc) are registered
// as globals. Flags succeeding the Command are not globals.
func serverConfigurationFromConfig() *core.ServerConfiguration {
	httpTimeout := time.Duration(config.httpTimeout) * time.Second
	var maxFileAge time.Duration
	if config.maxFileAge > 0 {
		maxFileAge = time.Duration(config.maxFileAge) * time.Minute
	} else {
		maxFileAge = time.Minute
	}

	var uploader string
	if config.uploaderType != "" {
		uploader = config.uploaderType
	} else {
		if config.awsAccessKeyID != "" {
			uploader = "s3"
		} else {
			uploader = "noop"
		}
	}

	var allowedExtensions = []string{}

	// Need to check if the string is empty because strings.Split("") returns a slice with one element
	if len(config.extensions) > 0 {
		allowedExtensions = strings.Split(config.extensions, ",")
	}

	// Build signature configuration
	var sigConfig *signature.Config
	if config.requireSignature {
		var secrets []string
		var err error

		if config.signingSecretsFile != "" {
			secrets, err = signature.LoadSecretsFromFile(config.signingSecretsFile)
			if err != nil {
				log.Fatalf("Failed to load signing secrets: %v", err)
			}
		}

		if len(secrets) == 0 {
			log.Fatal("Signature validation enabled but no secrets provided. Use --signing-secrets-file")
		}

		maxTTL := time.Duration(config.signatureMaxTTL) * time.Minute
		if maxTTL == 0 {
			maxTTL = time.Hour // Default to 1 hour
		}

		sigConfig = &signature.Config{
			Enabled:         true,
			RequireForReads: config.requireSignatureForRead,
			Secrets:         secrets,
			MaxTTL:          maxTTL,
		}
		log.Printf("Signature validation enabled (require for reads: %v, max TTL: %v, secrets count: %d)",
			sigConfig.RequireForReads, sigConfig.MaxTTL, len(sigConfig.Secrets))
	}

	return &core.ServerConfiguration{
		AllowedExtensions: allowedExtensions,
		LocalBasePath:     config.localBasePath,

		MaximumWidth:   config.maximumWidth,
		RemoteBasePath: config.remoteBasePath,
		RemoteBaseURL:  config.remoteBaseURL,

		UploaderType: uploader,
		MaxFileAge:   maxFileAge,

		// AWS specific
		AWSAccessKeyID: config.awsAccessKeyID,
		AWSSecretKey:   config.awsSecretKey,
		AWSBucket:      config.awsBucket,
		AWSRegion:      config.awsRegion,

		Outputs:             config.outputs,
		DefaultQuality:      uint(config.defaultQuality),
		UploaderConcurrency: uint(config.uploaderConcurrency),
		HTTPTimeout:         httpTimeout,

		// Signature validation
		SignatureConfig: sigConfig,
	}
}

// initializeUploader creates base path on destination server
func initializeUploader(sc *core.ServerConfiguration) {
	err := uploader.Initialize(sc)
	if err != nil {
		log.Println("EXITING: Unable to initialize uploader: ", err)
		os.Exit(2)
	}
}
