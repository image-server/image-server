package server

import (
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/processor/cli"
	"github.com/tylerb/graceful"
	"github.com/unrolled/render"
)

// Stats store atomic values
type Stats struct {
	Current     uint64 `json:"current"`
	TotalCount  uint64 `json:"total_count"`
	FailedCount uint64 `json:"failed_count"`
}

// Status keeps the current state of the server
type Status struct {
	Version    string `json:"version"`
	GitHash    string `json:"githash"`
	BuildStamp string `json:"buildstamp"`
	Message    string `json:"message"`
	Posting    *Stats `json:"posting"`
}

// ShuttingDown variable is used to note that the server is about to shut down.
// It is false by default, and set to true when a shutdown signal is received.
var ShuttingDown bool

func init() {
	ShuttingDown = false
}

// InitializeStatusServer starts a web server that can be used to monitor the health of the application.
// It returns a response with status code 200 if the system is healthy.
func InitializeStatusServer(listen string, port string) {
	log.Printf("starting status check server on http://%s:%s", listen, port)

	srv := &graceful.Server{
		Timeout: 30 * time.Second,
		Server: &http.Server{
			Addr:    listen + ":" + port,
			Handler: &ServerStatus{},
		},
	}

	srv.ListenAndServe()
}

// IncrCounter increments a given counter by 1
func IncrCounter(addr *uint64) {
	atomic.AddUint64(addr, 1)
}

// DecrCounter decrements a given counter by 1
func DecrCounter(addr *uint64) {
	atomic.AddUint64(addr, ^uint64(0))
}

var status = &Status{
	Version:    core.VERSION,
	GitHash:    core.GitHash,
	BuildStamp: core.BuildStamp,
	Posting:    &Stats{Current: 0, TotalCount: 0, FailedCount: 0},
}

// ServerStatus implements the http.Handler interface
type ServerStatus struct{}

// ServeHTTP serves the http response for the status page.
// It returns a response code 200 when the image server is available to process images.

// It returns a status code 501 when the server is shutting down, or when a processor is not detected.
// Details are provided in the body of the request.
//
func (f *ServerStatus) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	processorAvailable := cli.Available

	r := render.New(render.Options{
		IndentJSON: true,
	})

	var code int

	if ShuttingDown {
		status.Message = "Shutting down"
		code = 501
	} else if processorAvailable {
		status.Message = "OK"
		code = 200
	} else {
		status.Message = "There is no processor available. Make sure you have image magick installed."
		code = 501
	}

	r.JSON(w, code, status)
}
