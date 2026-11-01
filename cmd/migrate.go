package cmd

import (
	"fmt"
	"os"

	"github.com/image-server/image-server/migrate"
	"github.com/image-server/image-server/uploader/s3"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var migrateOpts struct {
	namespace string
	variants  bool
	dryRun    bool
	workers   int
}

var cmdMigrate = &cobra.Command{
	Use:   "migrate",
	Short: "Copy the local image tree to the S3 bucket",
	Long: `Copy originals and info.json (and, with --variants, rendered variants) from
--local_base_path to the bucket, under --remote_base_path, with the same keys
and content types the server uses. Files already in the bucket are skipped, so
it can be run again. tmp/ is not copied.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if config.awsBucket == "" {
			return errors.New("--aws_bucket is required")
		}
		s3.Initialize(config.awsBucket, config.awsRegion, s3.Options{UsePathStyle: config.awsPathStyle, SkipACL: config.awsSkipACL})

		summary, err := migrate.Run(migrate.Options{
			Root:       config.localBasePath,
			RemoteBase: config.remoteBasePath,
			Namespace:  migrateOpts.namespace,
			Variants:   migrateOpts.variants,
			DryRun:     migrateOpts.dryRun,
			Workers:    migrateOpts.workers,
			Log:        os.Stdout,
		}, &s3.Uploader{})
		if err != nil {
			return err
		}

		verb := "uploaded"
		if migrateOpts.dryRun {
			verb = "would upload"
		}
		fmt.Printf("images=%d %s=%d already_in_bucket=%d skipped=%d failed=%d\n",
			summary.Images, verb, summary.Uploaded, summary.Existing, summary.Skipped, summary.Failed)
		if summary.Failed > 0 {
			return errors.Errorf("%d files failed", summary.Failed)
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(cmdMigrate)

	f := cmdMigrate.Flags()
	f.StringVar(&config.localBasePath, "local_base_path", "public", "Directory with the images to copy")
	f.StringVar(&config.remoteBasePath, "remote_base_path", "", "Base path in the bucket")
	f.StringVar(&config.awsBucket, "aws_bucket", "", "S3 bucket")
	f.StringVar(&config.awsRegion, "aws_region", "", "S3 region")
	f.BoolVar(&config.awsPathStyle, "aws_use_path_style", false, "Address the bucket in the URL path (endpoint/bucket/key), as most S3-compatible stores expect")
	f.BoolVar(&config.awsSkipACL, "aws_skip_acl", false, "Upload without the public-read ACL, for private buckets and stores that reject ACLs")
	f.StringVar(&migrateOpts.namespace, "namespace", "", "Only copy this namespace")
	f.BoolVar(&migrateOpts.variants, "variants", false, "Also copy rendered variants; they are otherwise generated again when requested")
	f.BoolVar(&migrateOpts.dryRun, "dry_run", false, "List what would be uploaded, without uploading")
	f.IntVar(&migrateOpts.workers, "workers", 8, "Images copied in parallel")
}
