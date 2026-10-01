package processor

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/info"
	"github.com/image-server/image-server/logger"
	adapter "github.com/image-server/image-server/processor/cli"
	vipsadapter "github.com/image-server/image-server/processor/vips"
)

type ProcessorResult struct {
	ResizedPath string
	Error       error
}

var ImageProcessings map[string][]chan ProcessorResult
var processingMutex sync.RWMutex // To protect ImageProcessings

func init() {
	ImageProcessings = make(map[string][]chan ProcessorResult)
}

type Processor struct {
	Source             string
	Destination        string
	ImageConfiguration *core.ImageConfiguration
	ImageDetails       *info.ImageProperties
	Channels           *ProcessorChannels
}

type ProcessorChannels struct {
	ImageProcessed chan *core.ImageConfiguration
	Skipped        chan string
}

func (p *Processor) CreateImage() error {
	if p.ImageDetails == nil {
		log.Panic("ImageDetails is required")
	}

	c := make(chan ProcessorResult)
	go p.uniqueCreateImage(c)
	ipr := <-c

	return ipr.Error
}

// uniqueCreateImage generates the image once when several requests ask for
// the same destination; the others wait for that result. Notifications are
// only sent on success: callers stop reading the channels after an error.
func (p *Processor) uniqueCreateImage(c chan ProcessorResult) {
	key := p.Destination

	processingMutex.Lock()
	if waiters, present := ImageProcessings[key]; present {
		// Buffered so the generating request never blocks on this waiter
		wc := make(chan ProcessorResult, 1)
		ImageProcessings[key] = append(waiters, wc)
		processingMutex.Unlock()

		result := <-wc
		c <- result
		if result.Error == nil {
			p.notifySkipped()
		}
		return
	}
	ImageProcessings[key] = []chan ProcessorResult{}
	processingMutex.Unlock()

	processed, err := p.createIfNotAvailable()
	result := ProcessorResult{p.Destination, err}

	// Removing the key under the same lock that waiters join under means no
	// waiter can be added after the results are sent
	processingMutex.Lock()
	waiters := ImageProcessings[key]
	delete(ImageProcessings, key)
	processingMutex.Unlock()

	for _, wc := range waiters {
		wc <- result
	}
	c <- result

	if err != nil {
		return
	}
	if processed {
		p.notifyProcessed()
	} else {
		p.notifySkipped()
	}
}

func (p *Processor) createIfNotAvailable() (bool, error) {
	if _, err := os.Stat(p.Destination); os.IsNotExist(err) {
		start := time.Now()

		dir := filepath.Dir(p.Destination)
		os.MkdirAll(dir, 0700)

		// Use vips for supported formats, fall back to ImageMagick for unsupported formats
		format := strings.ToLower(p.ImageConfiguration.Format)
		inputContentType := ""
		if p.ImageDetails != nil {
			inputContentType = p.ImageDetails.ContentType
		}
		var processor core.Processor

		// Use vips if available and both input and output formats are supported
		useVips := vipsadapter.Available &&
			vipsadapter.SupportedFormat(format) &&
			(inputContentType == "" || vipsadapter.SupportedInputFormat(inputContentType))

		if useVips {
			processor = &vipsadapter.Processor{
				Source:             p.Source,
				Destination:        p.Destination,
				ImageConfiguration: p.ImageConfiguration,
				ImageDetails:       p.ImageDetails,
			}
		} else {
			processor = &adapter.Processor{
				Source:             p.Source,
				Destination:        p.Destination,
				ImageConfiguration: p.ImageConfiguration,
				ImageDetails:       p.ImageDetails,
			}
		}

		err = processor.CreateImage()

		if err != nil {
			logger.ImageProcessedWithErrors(p.ImageConfiguration)
			log.Println(err)
			return false, err
		}

		elapsed := time.Since(start)
		log.Printf("Took %s to generate image: %s", elapsed, p.Destination)
		return true, nil
	} else {
		return false, nil
	}
}

func (p *Processor) notifyProcessed() {
	logger.ImageProcessed(p.ImageConfiguration)

	p.Channels.ImageProcessed <- p.ImageConfiguration
	close(p.Channels.ImageProcessed)
	close(p.Channels.Skipped)
}

func (p *Processor) notifySkipped() {
	logger.ImageAlreadyProcessed(p.ImageConfiguration)

	p.Channels.Skipped <- p.Destination
	close(p.Channels.ImageProcessed)
	close(p.Channels.Skipped)
}
