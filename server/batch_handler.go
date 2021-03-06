package server

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/image-server/image-server/core"
	mantajob "github.com/image-server/image-server/job/manta"
	"github.com/image-server/image-server/logger"
	"github.com/image-server/image-server/uploader/manta/client"
	"github.com/unrolled/render"
)

func CreateBatchHandler(w http.ResponseWriter, req *http.Request, sc *core.ServerConfiguration) {
	defer logger.RequestLatency("create_batch", time.Now())

	vars := mux.Vars(req)
	namespace := vars["namespace"]

	r := render.New(render.Options{
		IndentJSON: true,
	})

	job, err := mantajob.CreateJob(sc.Outputs, sc.RemoteBasePath, namespace, req.Body)
	if err != nil {
		errorHandlerJSON(err, w, http.StatusInternalServerError)
		return
	}

	json := map[string]string{
		"job_id": job.JobID,
	}

	r.JSON(w, http.StatusOK, json)

	go job.AddInputs()
}

func BatchHandler(w http.ResponseWriter, req *http.Request, sc *core.ServerConfiguration) {
	defer logger.RequestLatency("batch", time.Now())

	vars := mux.Vars(req)
	uuid := vars["uuid"]

	mantaClient := client.DefaultClient()
	job, err := mantaClient.GetJob(uuid)

	if err != nil {
		log.Println(err)
		errorHandler(err, w, req, 500)
		return
	}

	if job.State == "done" {
		result, err := getJobOutput(uuid, mantaClient)
		if err != nil {
			log.Println(err)
			errorHandler(err, w, req, 500)
			return
		}

		w.WriteHeader(200)
		io.Copy(w, result)
	} else {
		// If not complete, return job details and 202
		r := render.New(render.Options{
			IndentJSON: true,
		})

		r.JSON(w, 202, job)
	}
}

func getJobOutput(uuid string, mantaClient *client.Client) (io.Reader, error) {
	output, err := mantaClient.GetJobOutput(uuid)
	if err != nil {
		return nil, err
	}

	result, err := mantaClient.GetObject(output)
	if err != nil {
		return nil, err
	}

	return result, nil
}
