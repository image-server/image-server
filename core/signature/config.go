package signature

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds signature validation configuration
type Config struct {
	// Enabled controls whether signature validation is active
	Enabled bool

	// RequireForReads controls whether GET requests also require signatures
	RequireForReads bool

	// Secrets is the list of valid signing secrets (supports rotation)
	Secrets []string

	// MaxTTL is the maximum allowed expiration window
	MaxTTL time.Duration
}

// LoadSecretsFromFile reads signing secrets from a file (one per line)
// Empty lines and lines starting with # are ignored
func LoadSecretsFromFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open secrets file: %w", err)
	}
	defer file.Close()

	var secrets []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		secrets = append(secrets, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read secrets file: %w", err)
	}

	if len(secrets) == 0 {
		return nil, fmt.Errorf("no secrets found in file: %s", path)
	}

	return secrets, nil
}
