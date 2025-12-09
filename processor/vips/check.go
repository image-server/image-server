package vips

import (
	"github.com/davidbyttow/govips/v2/vips"
)

var Available bool

func init() {
	Available = true

	// Try to initialize vips - if libvips is not installed, this will fail
	vips.Startup(nil)

	// Check if vips is actually working by getting the version
	if vips.Version == "" {
		Available = false
	}
}

// Shutdown should be called when the application exits to clean up vips resources
func Shutdown() {
	vips.Shutdown()
}

// SupportedFormat returns true if the output format can be produced by vips
func SupportedFormat(format string) bool {
	switch format {
	case "jpg", "jpeg", "png", "webp", "gif":
		return true
	default:
		return false
	}
}

// SupportedInputFormat returns true if vips can read this input format
func SupportedInputFormat(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/webp", "image/gif", "application/pdf":
		return true
	default:
		return false
	}
}
