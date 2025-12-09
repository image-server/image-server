package processor

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/info"
	"github.com/image-server/image-server/logger"
	adapter "github.com/image-server/image-server/processor/cli"
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

func (p *Processor) uniqueCreateImage(c chan ProcessorResult) {
	key := p.Destination
	_, present := ImageProcessings[key]

	processingMutex.Lock()

	if present {
		ImageProcessings[key] = append(ImageProcessings[key], c)
		processingMutex.Unlock()
		p.notifySkipped()
	} else {
		ImageProcessings[key] = []chan ProcessorResult{c}
		processingMutex.Unlock()

		processed, err := p.createIfNotAvailable()

		for _, cc := range ImageProcessings[key] {
			cc <- ProcessorResult{p.Destination, err}
			close(cc)
		}
		processingMutex.Lock()
		delete(ImageProcessings, key)
		processingMutex.Unlock()

		if processed {
			p.notifyProcessed()
		} else {
			p.notifySkipped()
		}
	}
}

func (p *Processor) createIfNotAvailable() (bool, error) {
	if _, err := os.Stat(p.Destination); os.IsNotExist(err) {
		start := time.Now()

		dir := filepath.Dir(p.Destination)
		os.MkdirAll(dir, 0700)

		processor := &adapter.Processor{
			Source:             p.Source,
			Destination:        p.Destination,
			ImageConfiguration: p.ImageConfiguration,
			ImageDetails:       p.ImageDetails,
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
