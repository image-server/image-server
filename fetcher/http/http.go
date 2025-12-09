package http

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Fetcher struct{}

var transport *http.Transport

func init() {
	transport = &http.Transport{
		ResponseHeaderTimeout: 10 * time.Second, // Timeout waiting for header
		Dial: (&net.Dialer{
			Timeout: 10 * time.Second, // Connection timeout
			// KeepAlive: 0, // Disable connection keepalive for fetcher transport
		}).Dial,
	}
}

func (f *Fetcher) Fetch(url string, destination string) error {
	if _, err := os.Stat(destination); os.IsNotExist(err) {
		start := time.Now()

		client := Client{http.Client{Transport: transport}}
		resp, err := client.Get(url)

		if err != nil {
			return err
		}
		defer resp.Body.Close()
		defer transport.CloseIdleConnections()

		if resp.StatusCode != 200 {
			return fmt.Errorf("Unable to download image: %s, status code: %d", url, resp.StatusCode)
		}

		log.Printf("Downloaded from %s with code %d", url, resp.StatusCode)

		dir := filepath.Dir(destination)
		os.MkdirAll(dir, 0700)

		out, err := os.Create(destination)
		if err != nil {
			log.Printf("Unable to create file: %s - %v", destination, err)
			return fmt.Errorf("Unable to create file: %s", destination)
		}
		defer out.Close()

		io.Copy(out, resp.Body)

		fileInfo, err := out.Stat()
		if err != nil {
			return err
		}

		if fileInfo.Size() < 10 {
			defer os.Remove(destination)
			return errors.New("File is empty")
		}

		log.Printf("Took %s to download image: %s", time.Since(start), destination)
	} else {
		log.Printf("Fetcher: image is already present on destination: %s", destination)
	}
	return nil
}
