package request

import (
	"fmt"
	"io"
	"sync"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/logger"
	"github.com/image-server/image-server/parser"
)

type Request struct {
	ServerConfiguration *core.ServerConfiguration
	Namespace           string
	Outputs             []string
	Uploader            core.Uploader
	Paths               core.Paths
	Hash                string
	SourceURL           string
	SourceData          io.ReadCloser
	ContentType         string
	directoryListing    map[string]string
}

func (r *Request) ProcessMultiple() error {
	missing, err := r.CalculateMissingOutputs()
	if err != nil {
		return err
	}

	if missing == nil {
		// All the files are already uploaded. Nothing do do!
		logger.AllImagesAlreadyProcessed(r.Namespace, r.Hash, r.SourceURL)
		return nil
	}

	// Outputs share the decoded original, so processing them side by side is
	// cheap; the limit keeps a long list from taking every core
	limit := 4
	if r.ServerConfiguration.ProcessorConcurrency > 0 {
		limit = int(r.ServerConfiguration.ProcessorConcurrency)
	}
	slots := make(chan struct{}, limit)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	failed := func() bool {
		mu.Lock()
		defer mu.Unlock()
		return firstErr != nil
	}

	for _, filename := range missing {
		wg.Add(1)
		go func(filename string) {
			defer wg.Done()
			err := r.processAndUpload(filename, slots, failed)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(filename)
	}
	wg.Wait()

	return firstErr
}

// processAndUpload generates one output, holding a slot while processing,
// then uploads it. It skips the work once another output has failed.
func (r *Request) processAndUpload(filename string, slots chan struct{}, failed func() bool) error {
	ic, err := parser.NameToConfiguration(r.ServerConfiguration, filename)
	if err != nil {
		return err
	}
	ic.Namespace = r.Namespace
	ic.ID = r.Hash

	slots <- struct{}{}
	if failed() {
		<-slots
		return nil
	}
	err = r.Process(ic)
	<-slots
	if err != nil {
		return err
	}

	localResizedPath := r.Paths.LocalImagePath(r.Namespace, r.Hash, ic.Filename)
	remoteResizedPath := r.Paths.RemoteImagePath(ic.Namespace, ic.ID, ic.Filename)
	return r.Uploader.Upload(localResizedPath, remoteResizedPath, ic.ToContentType())
}

// CalculateMissingOutputs determine what versions need to be generated
func (r *Request) CalculateMissingOutputs() (itemOutputs []string, err error) {
	if r.Outputs == nil {
		return nil, nil
	}

	err = r.FetchRemoteFileListing()

	if err == nil {
		for _, output := range r.Outputs {
			if r.RemoteMissesFile(output) {
				itemOutputs = append(itemOutputs, output)
			}
		}

	} else {
		return nil, err
	}

	return itemOutputs, nil
}

func (r *Request) RemoteMissesFile(filename string) bool {
	_, ok := r.directoryListing[filename]
	return !ok
}

func (r *Request) FetchRemoteFileListing() error {
	if r.directoryListing == nil {
		r.directoryListing = make(map[string]string)
	} else {
		// Already fetched the listing
		return nil
	}
	if r.Uploader == nil {
		return fmt.Errorf("missing uploader")
	}

	remoteDirectory := r.Paths.RemoteImageDirectory(r.Namespace, r.Hash)
	entries, err := r.Uploader.ListDirectory(remoteDirectory)

	if err != nil {
		return err
	}

	for _, entry := range entries {
		r.directoryListing[entry] = entry
	}
	return nil
}
