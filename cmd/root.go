package cmd

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/core/signature"
	"github.com/image-server/image-server/fetcher"
	"github.com/image-server/image-server/fetcher/http"
	"github.com/image-server/image-server/logger/prometheus"
	"github.com/image-server/image-server/logger/statsd"
	"github.com/image-server/image-server/logger/webhook"
	"github.com/image-server/image-server/namespaces"
	"github.com/image-server/image-server/paths"
	"github.com/image-server/image-server/uploader"
	"github.com/image-server/image-server/uploader/s3"
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
	awsPathStyle   bool
	awsSkipACL     bool
	s3Reads        bool

	cacheControl   string
	cacheTagHeader string

	renameCopyWorkers int
	deleteWorkers     int

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

	// Namespace rename and delete
	allowNamespaceAdmin     bool
	adminSigningSecretsFile string
	deleteNamespacePattern  string

	// Webhooks
	webhookURL     string
	webhookSecret  string
	webhookTimeout int
	webhookEvents  string
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

	if sc.S3Reads {
		if !sc.UploaderIsAws() {
			log.Fatal("--s3_reads requires the S3 uploader (--uploader s3)")
		}
		fetcher.Store = &s3.Uploader{}
	}
	// 0 keeps the defaults, e.g. for commands without these flags
	if sc.RenameCopyWorkers < 0 || sc.DeleteWorkers < 0 {
		log.Fatal("--rename_copy_workers and --delete_workers must be at least 1 (0 keeps the default)")
	}
	if sc.RenameCopyWorkers > 0 {
		namespaces.CopyWorkers = sc.RenameCopyWorkers
	}
	if sc.UploaderIsAws() && config.allowNamespaceAdmin {
		log.Printf("Namespace admin in S3: %d copies, %d delete requests at once", namespaces.CopyWorkers, sc.DeleteWorkers)
	}
	if sc.CacheTagHeader != "" && !headerName.MatchString(sc.CacheTagHeader) {
		log.Fatalf("Invalid --cache_tag_header %q", sc.CacheTagHeader)
	}

	// Enable webhooks if configured
	if config.webhookURL != "" {
		webhookCfg := &webhook.Config{
			URL:     config.webhookURL,
			Secret:  config.webhookSecret,
			Timeout: time.Duration(config.webhookTimeout) * time.Second,
			Events:  parseWebhookEvents(config.webhookEvents),
		}
		webhook.Enable(webhookCfg, adapters.Paths)
	}

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

	namespaceAdmin := namespaceAdminConfiguration()

	return &core.ServerConfiguration{
		AllowedExtensions: allowedExtensions,
		LocalBasePath:     config.localBasePath,

		MaximumWidth:   config.maximumWidth,
		RemoteBasePath: config.remoteBasePath,
		RemoteBaseURL:  config.remoteBaseURL,

		UploaderType: uploader,
		MaxFileAge:   maxFileAge,

		// AWS specific
		AWSAccessKeyID:  config.awsAccessKeyID,
		AWSSecretKey:    config.awsSecretKey,
		AWSBucket:       config.awsBucket,
		AWSRegion:       config.awsRegion,
		AWSUsePathStyle: config.awsPathStyle,
		AWSSkipACL:      config.awsSkipACL,
		S3Reads:         config.s3Reads,

		CacheControl:   config.cacheControl,
		CacheTagHeader: config.cacheTagHeader,

		RenameCopyWorkers: config.renameCopyWorkers,
		DeleteWorkers:     config.deleteWorkers,

		Outputs:              config.outputs,
		DefaultQuality:       uint(config.defaultQuality),
		UploaderConcurrency:  uint(config.uploaderConcurrency),
		ProcessorConcurrency: uint(config.processorConcurrency),
		HTTPTimeout:          httpTimeout,

		// Signature validation
		SignatureConfig: sigConfig,

		NamespaceAdmin: namespaceAdmin,
	}
}

// namespaceAdminConfiguration returns nil unless --allow-namespace-admin is
// set. It refuses to start without admin secrets.
func namespaceAdminConfiguration() *core.NamespaceAdminConfiguration {
	if !config.allowNamespaceAdmin {
		return nil
	}

	var secrets []string
	if config.adminSigningSecretsFile != "" {
		var err error
		secrets, err = signature.LoadSecretsFromFile(config.adminSigningSecretsFile)
		if err != nil {
			log.Fatalf("Failed to load admin signing secrets: %v", err)
		}
	}
	if len(secrets) == 0 {
		log.Fatal("--allow-namespace-admin requires secrets. Use --admin-signing-secrets-file")
	}

	maxTTL := time.Duration(config.signatureMaxTTL) * time.Minute
	if maxTTL == 0 {
		maxTTL = time.Hour
	}

	ac := &core.NamespaceAdminConfiguration{Secrets: secrets, MaxTTL: maxTTL}
	if config.deleteNamespacePattern != "" {
		pattern, err := namespaces.CompileDeletePattern(config.deleteNamespacePattern)
		if err != nil {
			log.Fatalf("Invalid --delete-namespace-pattern: %v", err)
		}
		ac.DeletePattern = pattern
	}
	log.Printf("Namespace admin enabled (delete pattern: %q, secrets count: %d)", config.deleteNamespacePattern, len(secrets))
	return ac
}

// headerName is an HTTP header field name (a token)
var headerName = regexp.MustCompile(`^[A-Za-z0-9!#$%&'*+.^_|~-]+$`)

// initializeUploader creates base path on destination server
func initializeUploader(sc *core.ServerConfiguration) {
	err := uploader.Initialize(sc)
	if err != nil {
		log.Println("EXITING: Unable to initialize uploader: ", err)
		os.Exit(2)
	}
}

// parseWebhookEvents parses a comma-separated list of webhook events
func parseWebhookEvents(events string) map[webhook.EventType]bool {
	if events == "" {
		return nil // nil means all events
	}

	result := make(map[webhook.EventType]bool)
	for _, e := range strings.Split(events, ",") {
		switch strings.TrimSpace(e) {
		case "uploaded":
			result[webhook.EventImageUploaded] = true
		case "processed":
			result[webhook.EventImageProcessed] = true
		case "failed":
			result[webhook.EventImageProcessingFailed] = true
		case "batch_complete":
			result[webhook.EventBatchComplete] = true
		default:
			log.Printf("Warning: unknown webhook event type: %s", e)
		}
	}
	return result
}
