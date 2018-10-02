package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/fetcher/http"
	"github.com/image-server/image-server/logger/logfile"
	"github.com/image-server/image-server/logger/statsd"
	"github.com/image-server/image-server/paths"
	"github.com/image-server/image-server/uploader"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	mantaURL    string
	mantaUser   string
	mantaKeyID  string
	sdcIdentity string

	maximumWidth   int
	defaultQuality int

	uploaderConcurrency  int
	processorConcurrency int
	httpTimeout          int
	gomaxprocs           int

	statsdHost   string
	statsdPort   int
	statsdPrefix string
	profile      bool

	version bool
}

var cfgFile string
var config configT

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "",
	Short: "Root command (empty) to avoid to have multiple main commands",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobra-example.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".cobra-example" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".cobra-example")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func serverConfiguration() (*core.ServerConfiguration, error) {
	sc := serverConfigurationFromConfig()
	statsd.Enable(config.statsdHost, config.statsdPort, config.statsdPrefix)
	logfile.Enable()

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
		} else if config.mantaKeyID != "" {
			uploader = "manta"
		} else {
			uploader = "noop"
		}
	}

	return &core.ServerConfiguration{
		WhitelistedExtensions: strings.Split(config.extensions, ","),
		LocalBasePath:         config.localBasePath,

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

		// Manta specific
		MantaURL:    config.mantaURL,
		MantaUser:   config.mantaUser,
		MantaKeyID:  config.mantaKeyID,
		SDCIdentity: config.sdcIdentity,

		Outputs:             config.outputs,
		DefaultQuality:      uint(config.defaultQuality),
		UploaderConcurrency: uint(config.uploaderConcurrency),
		HTTPTimeout:         httpTimeout,
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
