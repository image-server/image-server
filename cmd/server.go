package cmd

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/image-server/image-server/file_garbage_collector"
	"github.com/image-server/image-server/server"
)

var serverCmd = &cobra.Command{
	Use:   "server [flags]",
	Short: "image server",
	Long:  `Up the image server with the specified configuration`,
	RunE: func(cmd *cobra.Command, args []string) error {
		go handleShutdownSignals()

		if config.profile {
			go initializePprofServer()
		}

		sc, err := serverConfiguration()
		if err != nil {
			return err
		}

		go initializeUploader(sc)

		if config.uploaderType != "noop" {
		  go file_garbage_collector.Start(sc)
		}

		port := config.port
		server.InitializeServer(sc, config.listen, port)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(serverCmd)

	// CLI flags
	serverCmd.Flags().StringVar(&config.namespace, "namespace", "", "Namespace")
	serverCmd.Flags().StringVar(&config.outputs, "outputs", "", "Output files with dimension and compression: 'x300.jpg,x300.webp'")
	serverCmd.Flags().StringVar(&config.listen, "listen", "127.0.0.1", "IP address the server listens to")

	// HTTP Server settings
	serverCmd.Flags().StringVar(&config.port, "port", "7000", "Specifies the server port.")
	serverCmd.Flags().StringVar(&config.extensions, "extensions", "jpg,gif,webp", "Whitelisted extensions (separated by commas)")
	serverCmd.Flags().StringVar(&config.localBasePath, "local_base_path", "public", "Directory where the images will be saved")

	// Uploader paths
	serverCmd.Flags().StringVar(&config.remoteBaseURL, "remote_base_url", "", "Source domain for images")
	serverCmd.Flags().StringVar(&config.remoteBasePath, "remote_base_path", "", "base path for cloud storage")

	// Uploader
	serverCmd.Flags().StringVar(&config.uploaderType, "uploader", "", "Uploader ['s3', 'noop']")
	serverCmd.Flags().IntVar(&config.maxFileAge, "max_file_age", 30, "Max file age in minutes")

	// S3 uploader
	serverCmd.Flags().StringVar(&config.awsAccessKeyID, "aws_access_key_id", "", "S3 Access Key")
	serverCmd.Flags().StringVar(&config.awsSecretKey, "aws_secret_key", "", "S3 Secret")
	serverCmd.Flags().StringVar(&config.awsBucket, "aws_bucket", "", "S3 Bucket")
	serverCmd.Flags().StringVar(&config.awsRegion, "aws_region", "", "S3 Region")

	// Default image settings
	serverCmd.Flags().IntVar(&config.maximumWidth, "maximum_width", 1000, "Maximum image width")
	serverCmd.Flags().IntVar(&config.defaultQuality, "default_quality", 75, "Default image compression quality")

	// Settings
	serverCmd.Flags().IntVar(&config.uploaderConcurrency, "uploader_concurrency", 10, "Uploader concurrency")
	serverCmd.Flags().IntVar(&config.processorConcurrency, "processor_concurrency", 4, "Processor concurrency")
	serverCmd.Flags().IntVar(&config.httpTimeout, "http_timeout", 5, "HTTP request timeout in seconds")
	serverCmd.Flags().IntVar(&config.gomaxprocs, "gomaxprocs", 0, "It will use the default when set to 0")

	// Monitoring and Profiling
	serverCmd.Flags().StringVar(&config.statsdHost, "statsd_host", "127.0.0.1", "Statsd host")
	serverCmd.Flags().IntVar(&config.statsdPort, "statsd_port", 8125, "Statsd port")
	serverCmd.Flags().StringVar(&config.statsdPrefix, "statsd_prefix", "image_server.", "Statsd prefix")
	serverCmd.Flags().BoolVar(&config.profile, "profile", false, "Enable pprof")
	serverCmd.Flags().BoolVar(&config.enableStatsd, "enable_statsd", false, "Enable statsd metrics")

	// Signature validation
	serverCmd.Flags().BoolVar(&config.requireSignature, "require-signature", false, "Require signed URLs for uploads")
	serverCmd.Flags().BoolVar(&config.requireSignatureForRead, "require-signature-for-reads", false, "Also require signed URLs for GET requests")
	serverCmd.Flags().StringVar(&config.signingSecretsFile, "signing-secrets-file", "", "Path to file containing signing secrets (one per line)")
	serverCmd.Flags().IntVar(&config.signatureMaxTTL, "signature-max-ttl", 60, "Maximum allowed signature TTL in minutes")
}

func handleShutdownSignals() {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT)

	<-shutdown
	server.ShuttingDown = true
	log.Println("Shutting down. Allowing requests to finish within 30 seconds. Interrupt again to quit immediately.")

	go func() {
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT)

		<-shutdown
		log.Println("Forced to shutdown.")
		os.Exit(0)
	}()
}

func initializePprofServer() {
	log.Println(http.ListenAndServe("localhost:6060", nil))
}

func setGoMaxProcs(maxprocs int) {
	if maxprocs != 0 {
		runtime.GOMAXPROCS(maxprocs)
	}
}
