package server

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/core/signature"
	"github.com/image-server/image-server/logger"
	"github.com/image-server/image-server/namespaces"
	"github.com/unrolled/render"
)

// Admin routes are signed with the admin secrets, so the signature
// middleware leaves them to their handlers
var (
	deleteNamespacePath = regexp.MustCompile(`^/[a-z0-9_-]+/?$`)
	renameNamespacePath = regexp.MustCompile(`^/[a-z0-9_-]+/rename/[a-z0-9_-]+/?$`)
)

func isNamespaceAdminRequest(r *http.Request) bool {
	switch r.Method {
	case http.MethodDelete:
		return deleteNamespacePath.MatchString(r.URL.Path)
	case http.MethodPost:
		return renameNamespacePath.MatchString(r.URL.Path)
	}
	return false
}

type renameResponse struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Status string `json:"status"`
}

type deleteNamespaceResponse struct {
	Namespace     string `json:"namespace"`
	Status        string `json:"status"`
	ImagesDeleted int    `json:"images_deleted"`
	FilesDeleted  int    `json:"files_deleted"`
}

// Reasons a namespace admin request is rejected, for the counter and the
// warning log line
const (
	rejectBadSignature    = "bad_signature"
	rejectWrongSecret     = "wrong_secret"
	rejectPatternMismatch = "pattern_mismatch"
	rejectInvalidName     = "invalid_name"
	rejectDisabled        = "disabled"
)

// RenameNamespaceHandler moves a namespace to a new name. A missing source
// returns status "missing", so retries are safe; an existing target is 409.
func RenameNamespaceHandler(w http.ResponseWriter, req *http.Request, sc *core.ServerConfiguration) {
	defer logger.RequestLatency("rename_namespace", time.Now())
	start := time.Now()
	vars := mux.Vars(req)
	from, to := vars["namespace"], vars["to"]

	if !authorizeNamespaceAdmin(w, req, sc, "rename") {
		return
	}

	status, err := namespaces.Rename(sc.LocalBasePath, from, to)
	if errors.Is(err, namespaces.ErrInvalidName) {
		rejectNamespaceAdmin(w, req, "rename", rejectInvalidName, http.StatusBadRequest, err)
		return
	}
	requestID := requestID(req)
	if err != nil {
		go logger.NamespaceAdmin("rename", "error", "")
		log.Printf("Rename failed from=%s to=%s request_id=%q: %v", from, to, requestID, err)
		errorHandlerJSON(err, w, namespaceAdminErrorStatus(err))
		return
	}

	go logger.NamespaceAdmin("rename", status, "")
	log.Printf("Renamed namespace from=%s to=%s status=%s took=%s request_id=%q", from, to, status, time.Since(start), requestID)
	render.New(render.Options{IndentJSON: true}).JSON(w, http.StatusOK, renameResponse{From: from, To: to, Status: status})
}

// DeleteNamespaceHandler deletes a namespace with every image in it. Only
// namespaces matching the configured pattern can be deleted; a missing one
// returns status "missing".
func DeleteNamespaceHandler(w http.ResponseWriter, req *http.Request, sc *core.ServerConfiguration) {
	defer logger.RequestLatency("delete_namespace", time.Now())
	start := time.Now()
	namespace := mux.Vars(req)["namespace"]

	if sc.NamespaceAdmin != nil && sc.NamespaceAdmin.DeletePattern == nil {
		rejectNamespaceAdmin(w, req, "delete", rejectDisabled, http.StatusNotFound, errors.New("deleting namespaces is not enabled"))
		return
	}
	if !authorizeNamespaceAdmin(w, req, sc, "delete") {
		return
	}
	if !sc.NamespaceAdmin.DeletePattern.MatchString(namespace) {
		rejectNamespaceAdmin(w, req, "delete", rejectPatternMismatch, http.StatusForbidden, errors.New("namespace does not match the delete pattern"))
		return
	}

	result, err := namespaces.Delete(sc.LocalBasePath, namespace)
	if errors.Is(err, namespaces.ErrInvalidName) {
		rejectNamespaceAdmin(w, req, "delete", rejectInvalidName, http.StatusBadRequest, err)
		return
	}
	requestID := requestID(req)
	if err != nil {
		go logger.NamespaceAdmin("delete", "error", "")
		log.Printf("Delete failed namespace=%s request_id=%q: %v", namespace, requestID, err)
		errorHandlerJSON(err, w, namespaceAdminErrorStatus(err))
		return
	}

	go logger.NamespaceAdmin("delete", result.Status, "")
	log.Printf("Deleted namespace=%s status=%s images=%d files=%d took=%s request_id=%q",
		namespace, result.Status, result.Images, result.Files, time.Since(start), requestID)
	render.New(render.Options{IndentJSON: true}).JSON(w, http.StatusOK, deleteNamespaceResponse{
		Namespace:     namespace,
		Status:        result.Status,
		ImagesDeleted: result.Images,
		FilesDeleted:  result.Files,
	})
}

// authorizeNamespaceAdmin rejects the request and returns false when admin
// routes are disabled, the signature is not an admin one, or storage is remote
func authorizeNamespaceAdmin(w http.ResponseWriter, req *http.Request, sc *core.ServerConfiguration, op string) bool {
	ac := sc.NamespaceAdmin
	if ac == nil {
		rejectNamespaceAdmin(w, req, op, rejectDisabled, http.StatusNotFound, errors.New("Not Found"))
		return false
	}

	validator := signature.NewValidator(&signature.Config{Enabled: true, Secrets: ac.Secrets, MaxTTL: ac.MaxTTL, ExactPath: true})
	if err := validator.ValidateRequest(req); err != nil {
		reason := rejectBadSignature
		if signedWithUploadSecret(req, sc) {
			reason = rejectWrongSecret
		}
		recordRejection(req, op, reason)
		signature.WriteError(w, err)
		return false
	}

	// S3 has no rename, and deleting only the local copy would leave the
	// remote one
	if sc.UploaderIsAws() {
		rejectNamespaceAdmin(w, req, op, rejectDisabled, http.StatusNotImplemented, errors.New("namespace admin requires local storage (--uploader noop)"))
		return false
	}
	return true
}

// signedWithUploadSecret is true when the request carries a valid upload
// signature: a caller using the wrong key, rather than a forged one
func signedWithUploadSecret(req *http.Request, sc *core.ServerConfiguration) bool {
	if sc.SignatureConfig == nil || len(sc.SignatureConfig.Secrets) == 0 {
		return false
	}
	upload := signature.NewValidator(&signature.Config{Enabled: true, Secrets: sc.SignatureConfig.Secrets, MaxTTL: sc.SignatureConfig.MaxTTL})
	return upload.ValidateRequest(req) == nil
}

func rejectNamespaceAdmin(w http.ResponseWriter, req *http.Request, op, reason string, status int, err error) {
	recordRejection(req, op, reason)
	errorHandlerJSON(err, w, status)
}

// Rejections are always counted, but at most rejectionLogLimit warnings are
// logged per minute, so probing cannot flood the log
const rejectionLogLimit = 20

var rejectionLog struct {
	sync.Mutex
	window     time.Time
	logged     int
	suppressed int
}

// recordRejection counts a rejected admin request and logs a warning. It
// logs the path, never the query: that holds the signature.
func recordRejection(req *http.Request, op, reason string) {
	go logger.NamespaceAdmin(op, "rejected", reason)

	now := time.Now()
	rejectionLog.Lock()
	if now.Sub(rejectionLog.window) >= time.Minute {
		if rejectionLog.suppressed > 0 {
			log.Printf("WARN namespace admin: %d rejected request warnings suppressed", rejectionLog.suppressed)
		}
		rejectionLog.window, rejectionLog.logged, rejectionLog.suppressed = now, 0, 0
	}
	if rejectionLog.logged >= rejectionLogLimit {
		rejectionLog.suppressed++
		rejectionLog.Unlock()
		return
	}
	rejectionLog.logged++
	rejectionLog.Unlock()

	log.Printf("WARN namespace admin rejected op=%s reason=%s method=%s path=%q namespace=%s request_id=%q remote=%s forwarded_for=%q",
		op, reason, req.Method, req.URL.Path, mux.Vars(req)["namespace"], requestID(req), req.RemoteAddr, req.Header.Get("X-Forwarded-For"))
}

func namespaceAdminErrorStatus(err error) int {
	switch {
	case errors.Is(err, namespaces.ErrInvalidName):
		return http.StatusBadRequest
	case errors.Is(err, namespaces.ErrExists):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// requestID is the caller's X-Request-Id, for the audit log line
func requestID(req *http.Request) string {
	id := req.Header.Get("X-Request-Id")
	if len(id) > 128 {
		id = id[:128]
	}
	return id
}
