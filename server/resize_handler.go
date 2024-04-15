package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/logger"
	"github.com/image-server/image-server/parser"
	"github.com/image-server/image-server/request"
	"github.com/image-server/image-server/uploader"
)

// ResizeHandler asumes the original image is either stores locally or on the remote server
// it returns the processed image in given dimension and format.
// When an image is requested more than once, only one will do the processing,
// and both requests will return the same output
func ResizeHandler(w http.ResponseWriter, req *http.Request, sc *core.ServerConfiguration) {
	defer logger.RequestLatency("resize_image", time.Now())

	vars := mux.Vars(req)
	filename := vars["filename"]

	ic, err := parser.NameToConfiguration(sc, filename)
	if err != nil {
		errorHandler(err, w, req, http.StatusNotFound)
		return
	}

	if isFormatForbidden(ic.Format, sc) {
		errorHandler(errors.New("Not Found"), w, req, http.StatusNotFound)
		return
	}

	ic.ID = varsToHash(vars)
	ic.Namespace = vars["namespace"]

	qs := req.URL.Query()

	ir := request.Request{
		ServerConfiguration: sc,
		Namespace:           vars["namespace"],
		Outputs:             strings.Split(qs.Get("outputs"), ","),
		Uploader:            uploader.DefaultUploader(sc),
		Paths:               sc.Adapters.Paths,
		Hash:                ic.ID,
	}

	err = ir.Process(ic)
	if err != nil {
		errorHandlerJSON(err, w, http.StatusNotFound)
		return
	}

	localResizedPath := sc.Adapters.Paths.LocalImagePath(ic.Namespace, ic.ID, ic.Filename)
	http.ServeFile(w, req, localResizedPath)
}

func varsToHash(vars map[string]string) string {
	return fmt.Sprintf("%s%s%s%s", vars["id1"], vars["id2"], vars["id3"], vars["id4"])
}

func isFormatForbidden(format string, sc *core.ServerConfiguration) bool {
	if format == "" || len(sc.AllowedExtensions) == 0 {
		return false
	}

	for _, ext := range sc.AllowedExtensions {
		if ext == format {
			return false
		}
	}

	return true
}
