package cmd

import (
	"os"
	"strings"

	cliprocessor "github.com/image-server/image-server/cli"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var cmdCli = &cobra.Command{
	Use:   "cli [path]",
	Short: "CLI for the image server",
	Long:  `CLI for the image server. The images can be processed using this command.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.Errorf("CLI requires the path")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()

		path := args[0]
		outputsStr, err := getOutputsFlag(flags)
		if err != nil {
			return err
		}
		namespace, err := getNamespaceFlag(flags)
		if err != nil {
			return err
		}

		sc, err := serverConfiguration()
		if err != nil {
			return err
		}

		outputs := strings.Split(outputsStr, ",")

		if path != "" {
			err = cliprocessor.Process(sc, namespace, outputs, path)
		} else {
			err = cliprocessor.ProcessStream(sc, namespace, outputs, os.Stdin)
		}
		if err != nil {
			return err
		}

		return nil
	},
}

func getOutputsFlag(flags *flag.FlagSet) (string, error) {
	var outputs string
	var err error

	if flags == nil {
		return outputs, errors.New("No flags passed to getOutputsFlag")
	}

	outputs, err = flags.GetString("outputs")
	if err != nil {
		return outputs, err
	}

	return outputs, nil
}

func getNamespaceFlag(flags *flag.FlagSet) (string, error) {
	var namespace string
	var err error

	if flags == nil {
		return namespace, errors.New("No flags passed to getNamespaceFlag")
	}

	namespace, err = flags.GetString("namespace")
	if err != nil {
		return namespace, err
	}

	return namespace, nil
}

func init() {
	RootCmd.AddCommand(cmdCli)

	// CLI flags
	cmdCli.Flags().StringVar(&config.namespace, "namespace", "", "Namespace")
	cmdCli.Flags().StringVar(&config.outputs, "outputs", "", "Output files with dimension and compression: 'x300.jpg,x300.webp'")
	cmdCli.Flags().StringVar(&config.listen, "listen", "127.0.0.1", "IP address the server listens to")

	// HTTP Server settings
	cmdCli.Flags().StringVar(&config.port, "port", "7000", "Specifies the server port.")
	cmdCli.Flags().StringVar(&config.extensions, "extensions", "jpg,gif,webp", "Whitelisted extensions (separated by commas)")
	cmdCli.Flags().StringVar(&config.localBasePath, "local_base_path", "public", "Directory where the images will be saved")

	// Uploader paths
	cmdCli.Flags().StringVar(&config.remoteBaseURL, "remote_base_url", "", "Source domain for images")
	cmdCli.Flags().StringVar(&config.remoteBasePath, "remote_base_path", "", "base path for cloud storage")

	// Uploader
	cmdCli.Flags().StringVar(&config.uploaderType, "uploader", "", "Uploader ['s3', 'manta']")
	cmdCli.Flags().IntVar(&config.maxFileAge, "max_file_age", 30, "Max file age in minutes")

	// S3 uploader
	cmdCli.Flags().StringVar(&config.awsAccessKeyID, "aws_access_key_id", "", "S3 Access Key")
	cmdCli.Flags().StringVar(&config.awsSecretKey, "aws_secret_key", "", "S3 Secret")
	cmdCli.Flags().StringVar(&config.awsBucket, "aws_bucket", "", "S3 Bucket")
	cmdCli.Flags().StringVar(&config.awsRegion, "aws_region", "", "S3 Region")

	// Manta uploader
	cmdCli.Flags().StringVar(&config.mantaURL, "manta_url", "", "URL of Manta endpoint. https://us-east.manta.joyent.com")
	cmdCli.Flags().StringVar(&config.mantaUser, "manta_user", "", "The account name")
	cmdCli.Flags().StringVar(&config.mantaKeyID, "manta_key_id", "", "The fingerprint of the account or user SSH public key. Example: $(ssh-keygen -l -f $HOME/.ssh/id_rsa.pub | awk '{print $2}')")
	cmdCli.Flags().StringVar(&config.sdcIdentity, "sdc_identity", "", "Example: $HOME/.ssh/id_rsa")

	// Default image settings
	cmdCli.Flags().IntVar(&config.maximumWidth, "maximum_width", 1000, "Maximum image width")
	cmdCli.Flags().IntVar(&config.defaultQuality, "default_quality", 75, "Default image compression quality")

	// Settings
	cmdCli.Flags().IntVar(&config.uploaderConcurrency, "uploader_concurrency", 10, "Uploader concurrency")
	cmdCli.Flags().IntVar(&config.processorConcurrency, "processor_concurrency", 4, "Processor concurrency")
	cmdCli.Flags().IntVar(&config.httpTimeout, "http_timeout", 5, "HTTP request timeout in seconds")
	cmdCli.Flags().IntVar(&config.gomaxprocs, "gomaxprocs", 0, "It will use the default when set to 0")

	// Required flags
	cmdCli.MarkFlagRequired("outputs")
}
