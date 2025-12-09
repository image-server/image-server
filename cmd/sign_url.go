package cmd

import (
	"fmt"
	"time"

	"github.com/image-server/image-server/core/signature"
	"github.com/spf13/cobra"
)

var signURLCmd = &cobra.Command{
	Use:   "sign-url",
	Short: "Generate a signed URL for testing",
	Long: `Generate a signed URL for testing or debugging purposes.

This command is useful for:
- Testing your signature validation setup
- Debugging signature issues
- Generating one-off signed URLs for manual uploads

Example:
  image-server sign-url --secret "your-secret" --method POST --path /photos --ttl 15m
  image-server sign-url --secret "your-secret" --base-url https://images.example.com --method GET --path /photos/abc/def/ghi/jkl/x300.jpg`,
	RunE: func(cmd *cobra.Command, args []string) error {
		secret, _ := cmd.Flags().GetString("secret")
		baseURL, _ := cmd.Flags().GetString("base-url")
		method, _ := cmd.Flags().GetString("method")
		path, _ := cmd.Flags().GetString("path")
		ttlStr, _ := cmd.Flags().GetString("ttl")

		if secret == "" {
			return fmt.Errorf("--secret is required")
		}
		if path == "" {
			return fmt.Errorf("--path is required")
		}

		ttl, err := time.ParseDuration(ttlStr)
		if err != nil {
			return fmt.Errorf("invalid TTL duration: %w", err)
		}

		signer := signature.NewSigner(secret, baseURL)
		url := signer.SignURL(method, path, ttl)

		fmt.Println(url)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(signURLCmd)

	signURLCmd.Flags().String("secret", "", "Signing secret (required)")
	signURLCmd.Flags().String("base-url", "", "Base URL for the image server (optional)")
	signURLCmd.Flags().String("method", "POST", "HTTP method (GET, POST)")
	signURLCmd.Flags().String("path", "", "Path to sign (required)")
	signURLCmd.Flags().String("ttl", "15m", "Time-to-live for the signature (e.g., 15m, 1h)")
}
