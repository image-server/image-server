package server

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/image-server/image-server/processor/cli"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tylerb/graceful"
	"github.com/unrolled/render"
	"github.com/urfave/negroni"
)

// AdminData keeps the current state of the server
type AdminData struct {
	Message string `json:"message"`
}

// ShuttingDown variable is used to note that the server is about to shut down.
// It is false by default, and set to true when a shutdown signal is received.
var ShuttingDown bool

func init() {
	ShuttingDown = false
}

// InitializeAdminServer starts a web server that can be used to monitor the health of the application.
// It returns a response with data code 200 if the system is healthy.
func InitializeAdminServer(listen string, port string) {
	log.Printf("starting data check server on http://%s:%s", listen, port)

	srv := &graceful.Server{
		Timeout: 30 * time.Second,
		Server: &http.Server{
			Addr:    listen + ":" + port,
			Handler: NewAdminServerHandler(),
		},
	}

	srv.ListenAndServe()
}

// NewAdminServerHandler serves the probes and metrics only. It must not
// serve stored images: this port has no signature check.
func NewAdminServerHandler() http.Handler {
	router := mux.NewRouter()
	admin := &AdminHandler{}
	router.HandleFunc("/probe/ready", admin.ServeHTTP)
	router.HandleFunc("/probe/live", admin.ServeHTTP)
	router.Handle("/metrics", promhttp.Handler())

	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())
	n.UseHandler(router)
	return n
}

var data = &AdminData{}

// AdminHandler implements the http.Handler interface
type AdminHandler struct{}

// ServeHTTP serves the http response for the health page.
// It returns a response code 200 when the image server is available to process images.

// It returns a data code 501 when the server is shutting down, or when a processor is not detected.
// Details are provided in the body of the request.
//
func (f *AdminHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	processorAvailable := cli.Available

	r := render.New(render.Options{
		IndentJSON: true,
	})

	var code int

	if ShuttingDown {
		data.Message = "Shutting down"
		code = 501
	} else if processorAvailable {
		data.Message = "OK"
		code = 200
	} else {
		data.Message = "There is no processor available. Make sure you have image magick installed."
		code = 501
	}

	r.JSON(w, code, data)
}
