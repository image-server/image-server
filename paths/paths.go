package paths

import (
	"crypto/rand"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/image-server/image-server/cache"
)

// Paths
type Paths struct {
	LocalBasePath  string
	RemoteBasePath string
	RemoteBaseURL  string
}

// LocalOriginalPath returns local path for original image
func (p *Paths) LocalOriginalPath(namespace string, md5 string) string {
	return filepath.Join(p.LocalBasePath, p.originalPath(namespace, md5))
}

// LocalImageDirectory returns location for locally cached images
func (p *Paths) LocalImageDirectory(namespace string, md5 string) string {
	return filepath.Join(p.LocalBasePath, p.imageDirectory(namespace, md5))
}

// RemoteImageDirectory returns location for directory for images and info
func (p *Paths) RemoteImageDirectory(namespace string, md5 string) string {
	return filepath.Join(p.RemoteBasePath, p.imageDirectory(namespace, md5))
}

//  LocalImagePath returns local path for resized image
func (p *Paths) LocalImagePath(namespace string, md5 string, imageName string) string {
	return filepath.Join(p.LocalBasePath, p.imagePath(namespace, md5, imageName))
}

func (p *Paths) RemoteImagePath(namespace string, md5 string, imageName string) string {
	return filepath.Join(p.RemoteBasePath, p.imagePath(namespace, md5, imageName))
}

// RemoteImageURL returns "" when no remote store is configured (RemoteBaseURL empty).
func (p *Paths) RemoteImageURL(namespace string, md5 string, imageName string) string {
	if p.RemoteBaseURL == "" {
		return ""
	}
	u, _ := url.Parse(p.RemoteBaseURL)
	u.Path = filepath.Join(u.Path, p.RemoteImagePath(namespace, md5, imageName))
	return u.String()
}

// RemoteOriginalPath returns local path for original image
func (p *Paths) RemoteOriginalPath(namespace string, md5 string) string {
	return filepath.Join(p.RemoteBasePath, p.originalPath(namespace, md5))
}

// RemoteOriginalURL returns "" when no remote store is configured (RemoteBaseURL empty).
func (p *Paths) RemoteOriginalURL(namespace string, md5 string) string {
	if p.RemoteBaseURL == "" {
		return ""
	}
	u, _ := url.Parse(p.RemoteBaseURL)
	u.Path = filepath.Join(u.Path, p.RemoteOriginalPath(namespace, md5))
	return u.String()
}

func (p *Paths) LocalInfoPath(namespace string, md5 string) string {
	return filepath.Join(p.LocalBasePath, p.infoPath(namespace, md5))
}

func (p *Paths) RemoteInfoPath(namespace string, md5 string) string {
	return filepath.Join(p.RemoteBasePath, p.infoPath(namespace, md5))
}

// TempImagePath returns a temporary path to download files.
// This path is only used before the file is downloaded (we don't know the HASH of the contents just yet)
func (p *Paths) TempImagePath(urlstr string) string {
	u, _ := url.Parse(urlstr)
	name := cache.URLHash(u)
	return filepath.Join(p.LocalBasePath, "tmp", name)
}

func (p *Paths) RandomTempPath() string {
	b := make([]byte, 16)
	rand.Read(b)
	name := fmt.Sprintf("%x", b)
	return filepath.Join(p.LocalBasePath, "tmp", name)
}

// originalPath
func (p *Paths) originalPath(namespace string, md5 string) string {
	return filepath.Join(p.imageDirectory(namespace, md5), "original")
}

// imageDirectory returns relative directory starting at image root
func (p *Paths) imageDirectory(namespace string, md5 string) string {
	partitions := []string{md5[0:3], md5[3:6], md5[6:9], md5[9:32]}
	return filepath.Join(namespace, strings.Join(partitions, "/"))
}

func (p *Paths) infoPath(namespace string, md5 string) string {
	return filepath.Join(p.imageDirectory(namespace, md5), "info.json")
}

//  imagePath returns relative path to resized image
func (p *Paths) imagePath(namespace string, md5 string, imageName string) string {
	return filepath.Join(p.imageDirectory(namespace, md5), imageName)
}
