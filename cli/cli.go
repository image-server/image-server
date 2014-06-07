package main

import (
	"log"
	"sync"
	"bitbucket.org/tebeka/base62"

	"github.com/wanelo/image-server/core"
	"github.com/wanelo/image-server/fetcher/http"
	"github.com/wanelo/image-server/parser"
)

// A result is the product of reading and summing a file using MD5.
type result struct {
	ID  int
	Err error
}

// digester reads path names from paths and sends digests of the corresponding
// files on c until either paths or done is closed.
func digester(conf *CliConfiguration, done <-chan struct{}, ids <-chan int, c chan<- result) {
	for id := range ids { // HLpaths
		encodedID := base62.Encode(uint64(id))
		sc := conf.ServerConfiguration
		ic := &core.ImageConfiguration{
			ServerConfiguration: sc,
			Namespace:           "p",
			ID:                  encodedID,
		}
		err := http.FetchOriginal(ic)

		for _, filename := range conf.Outputs {
			ic, err := parser.NameToConfiguration(sc, filename)
			if err != nil {
				log.Printf("Error parsing name: %v\n", err)
				continue
			}

			ic.ServerConfiguration = sc
			ic.Namespace = conf.Namespace
			ic.ID = encodedID

			sc.Adapters.Processor.CreateImage(ic)
		}

		select {
		case c <- result{id, err}:
		case <-done:
			return
		}

	}
}

func enqueueIds(done <-chan struct{}, ids []int) <-chan int {
	idsc := make(chan int)
	go func() { // HL
		// Close the ids channel after Walk returns.
		defer close(idsc) // HL

		for _, id := range ids {
			idsc <- id
		}
	}()
	return idsc
}

func createAll(conf *CliConfiguration) error {
	done := make(chan struct{})
	defer close(done)

	ids, _ := conf.ProductIds()
	idsc := enqueueIds(done, ids)

	// Start a fixed number of goroutines to read and digest files.
	c := make(chan result) // HLc
	var wg sync.WaitGroup

	numDigesters := conf.Concurrency
	wg.Add(numDigesters)

	for i := 0; i < numDigesters; i++ {
		go func() {
			digester(conf, done, idsc, c) // HLc
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(c) // HLc
	}()
	// End of pipeline. OMIT

	for r := range c {
		log.Printf("Completed processing image %v", r.ID)
	}

	return nil
}

func main() {
	// Creates images in a specific range
	// Returns urls of generated images

	cliConfiguration := extractCliConfiguration()
	err := createAll(cliConfiguration)

	// m, err := MD5All()
	if err != nil {
		log.Println(err)
		return
	}
}