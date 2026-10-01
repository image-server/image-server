package cli

import (
	"fmt"
	"log"
	"regexp"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/info"
	"github.com/image-server/image-server/parser"
	"github.com/image-server/image-server/processor"
	"github.com/image-server/image-server/uploader"
)

var pathHashRegex *regexp.Regexp

func init() {
	pathHashRegex = regexp.MustCompile(`\/([0-9a-f]{3})\/([0-9a-f]{3})\/([0-9a-f]{3})\/([0-9a-f]{23})\/`)
}

type ImageUpload struct {
	ServerConfiguration *core.ServerConfiguration
	Namespace           string
	Hash                string
	Filename            string
	LocalPath           string
	ContentType         string
}

func (iu *ImageUpload) Upload() error {
	uploader := uploader.DefaultUploader(iu.ServerConfiguration)
	remoteResizedPath := iu.ServerConfiguration.Adapters.Paths.RemoteImagePath(iu.Namespace, iu.Hash, iu.Filename)
	log.Printf("uploading %s to remote: %s", iu.LocalPath, remoteResizedPath)
	return uploader.Upload(iu.LocalPath, remoteResizedPath, iu.ContentType)
}

type ImageProcessor struct {
	Image     *Image
	Outputs   []string
	Namespace string
	channel   chan (string)
}

func NewImageProcessor(namespace string, path string, outputs []string) *ImageProcessor {
	processingChannel := make(chan string)
	return &ImageProcessor{
		Image:     NewImage(path, outputs, processingChannel),
		Outputs:   outputs,
		Namespace: namespace,
		channel:   processingChannel,
	}
}

// ProcessMissing processes all requested outputs
func (ip *ImageProcessor) ProcessMissing(sc *core.ServerConfiguration) error {
	for _, filename := range ip.Outputs {
		err := ip.ProcessOutput(sc, filename)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ip *ImageProcessor) ProcessOutput(sc *core.ServerConfiguration, filename string) error {
	ic, err := parser.NameToConfiguration(sc, filename)
	if err != nil {
		return fmt.Errorf("Error parsing name %s: %w", filename, err)
	}

	// Buffered so the goroutine can finish after the path has been received
	errc := make(chan error, 1)
	go func() {
		errc <- ip.Image.ProcessOutput(sc, ip.Namespace, filename)
	}()

	// when Image.ProcessOutput puts something on the channel, take that info
	// and run it through ImageUpload to upload it. It only returns before
	// sending a path when it fails.
	select {
	case err := <-errc:
		return err
	case localImagePath := <-ip.channel:
		// Hash                string

		upload := ImageUpload{
			ServerConfiguration: sc,
			LocalPath:           localImagePath,
			Filename:            filename,
			Namespace:           ip.Namespace,
			Hash:                ip.Image.Hash,
			ContentType:         ic.ToContentType(),
		}
		if err := upload.Upload(); err != nil {
			return fmt.Errorf("uploading %s: %w", filename, err)
		}
	}

	return nil
}

type Image struct {
	LocalOriginalPath string
	Outputs           []string
	Hash              string
	processingChannel chan string
}

func NewImage(path string, outputs []string, c chan string) *Image {
	img := &Image{
		LocalOriginalPath: path,
		Outputs:           outputs,
		processingChannel: c,
	}
	img.Hash = img.ToHash()
	return img
}

func (i *Image) ToHash() string {
	m := pathHashRegex.FindStringSubmatch(i.LocalOriginalPath)
	return fmt.Sprintf("%s%s%s%s", m[1], m[2], m[3], m[4])
}

// ProcessOutput
// Takes a filename, sends Image metadata through a processor to generate that new file
// Once complete, pushes a LocalImage onto channel c
func (i *Image) ProcessOutput(sc *core.ServerConfiguration, namespace string, filename string) error {
	ic, err := parser.NameToConfiguration(sc, filename)
	if err != nil {
		return fmt.Errorf("Error parsing name %s: %w", filename, err)
	}

	details, err := (&info.Info{Path: i.LocalOriginalPath}).ImageDetails()
	if err != nil {
		return err
	}

	ic.Namespace = namespace
	ic.ID = i.Hash

	pchan := &processor.ProcessorChannels{
		ImageProcessed: make(chan *core.ImageConfiguration),
		Skipped:        make(chan string),
	}

	localPath := sc.Adapters.Paths.LocalImagePath(namespace, i.Hash, filename)

	p := processor.Processor{
		Source:             i.LocalOriginalPath,
		Destination:        localPath,
		ImageConfiguration: ic,
		ImageDetails:       details,
		Channels:           pchan,
	}

	err = p.CreateImage()

	if err != nil {
		return err
	}

	select {
	case <-pchan.ImageProcessed:
		i.processingChannel <- localPath
	case path := <-pchan.Skipped:
		log.Println("Skipped processing (image)", path)
		i.processingChannel <- localPath
	}

	return nil
}
