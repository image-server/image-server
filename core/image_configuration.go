package core

import (
	"log"

	"github.com/image-server/image-server/mime"
)

// ImageConfiguration struct
// Properties used to generate new image
type ImageConfiguration struct {
	// ServerConfiguration *ServerConfiguration
	ID        string
	Width     int
	Height    int
	Filename  string
	Format    string
	Source    string
	Quality   uint
	Namespace string
}

// ToContentType returns the content type based on the image format
func (ic *ImageConfiguration) ToContentType() string {
	contentType := mime.ExtToContentType(ic.Format)

	if contentType == "" {
		log.Printf("ToContentType: Can't extract content type from format. format=%s, contentType=%s", ic.Format, contentType)
	}

	return contentType
}
