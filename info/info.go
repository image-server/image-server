package info

import (
	"crypto/md5"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/image-server/image-server/mime"
	_ "golang.org/x/image/webp"
)

type Info struct {
	Path        string
	ContentType string
}

func (i Info) FileHash() (hash string, err error) {
	infile, err := os.Open(i.Path)
	if err != nil {
		return "", err
	}
	defer infile.Close()

	h := md5.New()
	io.Copy(h, infile)

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// ImageDetails extracts file hash, height, and width when providing a image path
// it returns an ImageDetails object
func (i Info) ImageDetails() (*ImageProperties, error) {
	if reader, err := os.Open(i.Path); err == nil {
		defer reader.Close()
		var contentType string
		var details *ImageProperties

		im, format, err := image.DecodeConfig(reader)
		if err == nil && format != "" {
			contentType, err = getContentTypeFromExtension(format)
			if err != nil {
				return nil, err
			}

			details = &ImageProperties{
				Height:      im.Height,
				Width:       im.Width,
				ContentType: contentType,
			}
		} else if i.ContentType == "image/svg+xml" {
			// Imagemagick is unable to determine svg filetype without file format
			details = &ImageProperties{
				ContentType: i.ContentType,
			}
		} else {
			// can't calculate content type, so will use ImageMagick as fallback
			// use fallback
			details, err = i.DetailsFromImageMagick()
			if err != nil {
				return nil, err
			}
		}

		hash, err := i.FileHash()
		details.Hash = hash
		return details, nil

	} else {
		return nil, err
	}
}

func (i Info) DetailsFromImageMagick() (*ImageProperties, error) {
	tmpDir, err := ioutil.TempDir("", "magick")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	args := []string{"-format", "%[fx:w]:%[fx:h]:%m", i.Path}
	cmd := exec.Command("identify", args...)
	cmd.Env = []string{"TMPDIR=" + tmpDir, "MAGICK_DISK_LIMIT=100000000"}
	out, err := cmd.Output()

	if err != nil {
		return nil, fmt.Errorf("ImageMagick failed to identify properties")
	}

	dimensions := fmt.Sprintf("%s", out)
	dimensions = strings.TrimSpace(dimensions)

	log.Println("Info.DetailsFromImageMagick - Using ImageMagick as fallback:", i.Path)

	d := strings.Split(dimensions, ":")
	w, err := strconv.Atoi(d[0])
	if err != nil {
		log.Printf("Can't convert width to integer: %s\n", d[0])
		return nil, err
	}

	h, err := strconv.Atoi(d[1])
	if err != nil {
		log.Printf("Can't convert height to integer: %s\n", d[1])
		return nil, err
	}

	contentType, err := getContentTypeFromExtension(d[2])
	if err != nil {
		return nil, err
	}

	return &ImageProperties{
		Height:      h,
		Width:       w,
		ContentType: contentType,
	}, nil
}

func getContentTypeFromExtension(format string) (string, error) {
	if format == "" {
		return "", errors.New("Can't extract format")
	}

	contentType := mime.ExtToContentType(format)
	if contentType == "" {
		return "", fmt.Errorf("Can't extract content type from format. format=%s, contentType=%s", format, contentType)
	}

	return contentType, nil
}
