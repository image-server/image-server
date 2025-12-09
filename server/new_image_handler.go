package server

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/info"
	"github.com/image-server/image-server/logger"
	"github.com/image-server/image-server/request"
	"github.com/image-server/image-server/uploader"
	"github.com/unrolled/render"
)

// NewImageHandler handles posting new images
func NewImageHandler(w http.ResponseWriter, req *http.Request, sc *core.ServerConfiguration) {
	defer logger.RequestLatency("new_image", time.Now())
	go logger.ImagePosted()

	qs := req.URL.Query()
	vars := mux.Vars(req)
	sourceURL := qs.Get("source")
	namespace := vars["namespace"]
	outputs := []string{}
	contentType := req.Header.Get("Content-Type")

	if qs.Get("outputs") != "" {
		outputs = strings.Split(qs.Get("outputs"), ",")
	}

	request := &request.Request{
		ServerConfiguration: sc,
		Namespace:           namespace,
		Outputs:             outputs,
		Uploader:            uploader.DefaultUploader(sc),
		Paths:               sc.Adapters.Paths,
		SourceURL:           sourceURL,
		SourceData:          req.Body,
		ContentType:         contentType,
	}

	imageDetails, err := request.Create()
	if err != nil {
		go logger.ImagePostingFailed()
		log.Println("Failed to create image from", sourceURL, "-", err)
		errorHandlerJSON(err, w, http.StatusNotFound)
		return
	}

	renderImageDetails(w, imageDetails)
}

func renderImageDetails(w http.ResponseWriter, imageDetails *info.ImageProperties) {
	r := render.New(render.Options{
		IndentJSON: true,
	})

	r.JSON(w, http.StatusOK, imageDetails)
}
