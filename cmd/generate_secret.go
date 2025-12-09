package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/spf13/cobra"
)

var generateSecretCmd = &cobra.Command{
	Use:   "generate-secret",
	Short: "Generate a cryptographically secure signing secret",
	Long: `Generate a cryptographically secure random secret for use with signed URLs.

The generated secret can be added to your secrets file (one secret per line).
For secret rotation, add the new secret to the top of the file while keeping
the old secret(s) below it until all existing signed URLs have expired.

Example:
  image-server generate-secret >> /etc/image-server/secrets.txt`,
	RunE: func(cmd *cobra.Command, args []string) error {
		length, _ := cmd.Flags().GetInt("length")
		count, _ := cmd.Flags().GetInt("count")

		for i := 0; i < count; i++ {
			secret, err := generateSecret(length)
			if err != nil {
				return fmt.Errorf("failed to generate secret: %w", err)
			}
			fmt.Println(secret)
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(generateSecretCmd)

	generateSecretCmd.Flags().Int("length", 32, "Length of the secret in bytes (before base64 encoding)")
	generateSecretCmd.Flags().Int("count", 1, "Number of secrets to generate")
}

func generateSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
