package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/gorilla/mux"
)

// Router is local alias for router type.
type Router = mux.Router

// NewRouter is local alias for router creation function.
var NewRouter = mux.NewRouter

// ErrAjax is error object on AJAX API handlers calls.
type ErrAjax struct {
	What error
	Code int
}

func (e *ErrAjax) Error() string {
	return fmt.Sprintf("error with code %d: %s", e.Code, e.What.Error())
}

func (e *ErrAjax) Unwrap() error {
	return e.What
}

// MarshalJSON is standard JSON interface implementation for errors on Ajax.
func (e *ErrAjax) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		What string `json:"what"`
		When int64  `json:"when"`
		Code int    `json:"code,omitempty"`
	}{
		e.What.Error(),
		UnixJSNow(),
		e.Code,
	})
}

// UnixJS converts time to UNIX-time in milliseconds, compatible with javascript time format.
func UnixJS(u time.Time) int64 {
	return u.UnixNano() / 1000000
}

// UnixJSNow returns same result as Date.now() in javascript.
func UnixJSNow() int64 {
	return time.Now().UnixNano() / 1000000
}

const (
	jsoncontent = "application/json;charset=utf-8"
	htmlcontent = "text/html;charset=utf-8"
	csscontent  = "text/css;charset=utf-8"
	jscontent   = "text/javascript;charset=utf-8"
)

var serverlabel string

func makeServerLabel(label, version string) {
	serverlabel = fmt.Sprintf("%s/%s (%s)", label, version, runtime.GOOS)
}

// AjaxGetArg fetch and unmarshal request argument.
func AjaxGetArg(r *http.Request, arg interface{}) error {
	if jb, _ := io.ReadAll(r.Body); len(jb) > 0 {
		if err := json.Unmarshal(jb, arg); err != nil {
			return &ErrAjax{err, AECbadjson}
		}
	} else {
		return &ErrAjax{ErrNoJSON, AECnoreq}
	}
	return nil
}

// WriteStdHeader setup common response headers.
func WriteStdHeader(w http.ResponseWriter) {
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Server", serverlabel)
	w.Header().Set("X-Frame-Options", "sameorigin")
}

// WriteHTMLHeader setup standard response headers for message with HTML content.
func WriteHTMLHeader(w http.ResponseWriter) {
	WriteStdHeader(w)
	w.Header().Set("Content-Type", htmlcontent)
}

// WriteJSONHeader setup standard response headers for message with JSON content.
func WriteJSONHeader(w http.ResponseWriter) {
	WriteStdHeader(w)
	w.Header().Set("Content-Type", jsoncontent)
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

// WriteJSON writes to response given status code and marshaled body.
func WriteJSON(w http.ResponseWriter, status int, body interface{}) {
	if body == nil {
		w.WriteHeader(status)
		WriteJSONHeader(w)
		return
	}
	/*if b, ok := body.([]byte); ok {
		w.WriteHeader(status)
		WriteJSONHeader(w)
		w.Write(b)
		return
	}*/
	var b, err = json.Marshal(body)
	if err == nil {
		w.WriteHeader(status)
		WriteJSONHeader(w)
		w.Write(b)
	} else {
		b, _ = json.Marshal(&ErrAjax{err, AECbadbody})
		w.WriteHeader(http.StatusInternalServerError)
		WriteJSONHeader(w)
		w.Write(b)
	}
}

// WriteOK puts 200 status code and some data to response.
func WriteOK(w http.ResponseWriter, body interface{}) {
	WriteJSON(w, http.StatusOK, body)
}

// WriteError puts to response given error status code and ErrAjax formed by given error object.
func WriteError(w http.ResponseWriter, status int, err error, code int) {
	WriteJSON(w, status, &ErrAjax{err, code})
}

// WriteError400 puts to response 400 status code and ErrAjax formed by given error object.
func WriteError400(w http.ResponseWriter, err error, code int) {
	WriteJSON(w, http.StatusBadRequest, &ErrAjax{err, code})
}

// WriteError500 puts to response 500 status code and ErrAjax formed by given error object.
func WriteError500(w http.ResponseWriter, err error, code int) {
	WriteJSON(w, http.StatusInternalServerError, &ErrAjax{err, code})
}

// RegisterRoutes puts application routes to given router.
func RegisterRoutes(gmux *Router) {
	// API routes
	var v0 = gmux.PathPrefix("/api/v0").Subrouter()
	v0.Path("/ping").HandlerFunc(apiPing)
	v0.Path("/upload").Methods("POST", "PUT").HandlerFunc(apiUpload)
}
