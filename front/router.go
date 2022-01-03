package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type void = struct{}

type jerr struct {
	error
}

// Unwrap returns inherited error object.
func (e *jerr) Unwrap() error {
	return e.error
}

// MarshalJSON is standard JSON interface implementation to stream errors on Ajax.
func (e *jerr) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Error())
}

// ErrAjax is error object on AJAX API handlers calls.
type ErrAjax struct {
	What jerr  `json:"what"`
	When int64 `json:"when"`
	Code int   `json:"code,omitempty"`
}

// MakeAjaxErr is ErrAjax constructor.
func MakeAjaxErr(what error, code int) *ErrAjax {
	return &ErrAjax{
		What: jerr{what},
		When: UnixJSNow(),
		Code: code,
	}
}

func (e *ErrAjax) Error() string {
	return fmt.Sprintf("error with code %d: %s", e.Code, e.What.Error())
}

// Unwrap returns inherited error object.
func (e *ErrAjax) Unwrap() error {
	return e.What
}

// ErrPanic is error object that helps to get stack trace of goroutine within panic rises.
type ErrPanic struct {
	ErrAjax
	Stack string `json:"stack,omitempty"`
}

// MakeErrPanic is ErrPanic constructor.
func MakeErrPanic(what error, code int, stack string) *ErrPanic {
	return &ErrPanic{
		ErrAjax: ErrAjax{
			What: jerr{what},
			When: UnixJSNow(),
			Code: code,
		},
		Stack: stack,
	}
}

// UnixJS converts time to UNIX-time in milliseconds, compatible with javascript time format.
func UnixJS(u time.Time) int64 {
	return u.UnixNano() / 1000000
}

// UnixJSNow returns same result as Date.now() in javascript.
func UnixJSNow() int64 {
	return time.Now().UnixNano() / 1000000
}

////////////////
// Routes API //
////////////////

// Router is local alias for router type.
type Router = mux.Router

// NewRouter is local alias for router creation function.
var NewRouter = mux.NewRouter

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
func AjaxGetArg(w http.ResponseWriter, r *http.Request, arg interface{}) (err error) {
	if jb, _ := io.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, arg); err != nil {
			WriteError400(w, err, AECbadjson)
			return
		}
	} else {
		err = ErrNoJSON
		WriteError400(w, err, AECnoreq)
		return
	}
	return
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
		b, _ = json.Marshal(MakeAjaxErr(err, AECbadbody))
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
	WriteJSON(w, status, MakeAjaxErr(err, code))
}

// WriteError400 puts to response 400 status code and ErrAjax formed by given error object.
func WriteError400(w http.ResponseWriter, err error, code int) {
	WriteJSON(w, http.StatusBadRequest, MakeAjaxErr(err, code))
}

// WriteError500 puts to response 500 status code and ErrAjax formed by given error object.
func WriteError500(w http.ResponseWriter, err error, code int) {
	WriteJSON(w, http.StatusInternalServerError, MakeAjaxErr(err, code))
}

//////////////////
// Routes table //
//////////////////

// Transaction locker, locks until handler will be done.
var handwg sync.WaitGroup

// AjaxMiddleware is base handler middleware for AJAX API calls.
func AjaxMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if what := recover(); what != nil {
				var err error
				switch v := what.(type) {
				case error:
					err = v
				case string:
					err = errors.New(v)
				case fmt.Stringer:
					err = errors.New(v.String())
				default:
					err = errors.New("panic was thrown at handler")
				}
				var buf [2048]byte
				var stacklen = runtime.Stack(buf[:], false)
				var str = string(buf[:stacklen])
				log.Println(str)
				WriteJSON(w, http.StatusInternalServerError, MakeErrPanic(err, AECpanic, str))
			}
		}()

		// lock before exit check
		handwg.Add(1)
		defer handwg.Done()

		// check on exit during handler is called
		select {
		case <-exitctx.Done():
			return
		default:
		}

		// call the next handler, which can be another middleware in the chain, or the final handler
		next.ServeHTTP(w, r)
	})
}

// RegisterRoutes puts application routes to given router.
func RegisterRoutes(gmux *Router) {
	// API routes
	var api = gmux.PathPrefix("/api/v0").Subrouter()
	api.Use(AjaxMiddleware)
	api.Path("/ping").HandlerFunc(pingAPI)
	api.Path("/nodesize").HandlerFunc(nodesizeAPI)
	api.Path("/upload").Methods("POST", "PUT").HandlerFunc(uploadAPI)
	api.Path("/download").HandlerFunc(downloadAPI)
	api.Path("/fileinfo").HandlerFunc(fileinfoAPI)
	api.Path("/remove").HandlerFunc(removeAPI)
	api.Path("/clear").HandlerFunc(clearAPI)
	api.Path("/addnode").HandlerFunc(addnodeAPI)
}
