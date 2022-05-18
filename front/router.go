package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"google.golang.org/grpc/grpclog"
	"gopkg.in/yaml.v3"
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

// AjaxErr is error object on AJAX API handlers calls.
type AjaxErr struct {
	// message with problem description
	What jerr `json:"what" yaml:"what" xml:"what"`
	// time of error rising, in milliseconds of UNIX format
	When unix_t `json:"when" yaml:"when" xml:"when"`
	// unique API error code
	Code int `json:"code,omitempty" yaml:"code,omitempty" xml:"code,omitempty"`
	// URL with problem detailed description
	Info string `json:"info,omitempty" yaml:"info,omitempty" xml:"info,omitempty"`
}

// MakeAjaxErr is AjaxErr simple constructor.
func MakeAjaxErr(what error, code int) *AjaxErr {
	return &AjaxErr{
		What: jerr{what},
		When: UnixJSNow(),
		Code: code,
	}
}

// MakeAjaxInfo is AjaxErr constructor with info URL.
func MakeAjaxInfo(what error, code int, info string) *AjaxErr {
	return &AjaxErr{
		What: jerr{what},
		When: UnixJSNow(),
		Code: code,
		Info: info,
	}
}

func (e *AjaxErr) Error() string {
	return fmt.Sprintf("error with code %d: %s", e.Code, e.What.Error())
}

// Unwrap returns inherited error object.
func (e *AjaxErr) Unwrap() error {
	return e.What
}

// ErrPanic is error object that helps to get stack trace of goroutine within panic rises.
type ErrPanic struct {
	AjaxErr
	Stack string `json:"stack,omitempty"`
}

// MakeErrPanic is ErrPanic constructor.
func MakeErrPanic(what error, code int, stack string) *ErrPanic {
	return &ErrPanic{
		AjaxErr: AjaxErr{
			What: jerr{what},
			When: UnixJSNow(),
			Code: code,
		},
		Stack: stack,
	}
}

type unix_t uint64

func (ut unix_t) Time() time.Time {
	return time.Unix(int64(ut/1000), int64(ut%1000)*1000000)
}

const ExifDate = "2006:01:02 15:04:05.999"

// MarshalYAML is YAML marshaler interface implementation.
func (ut unix_t) MarshalYAML() (interface{}, error) {
	return ut.Time().Format(ExifDate), nil
}

// UnmarshalYAML is YAML unmarshaler interface implementation.
func (ut *unix_t) UnmarshalYAML(value *yaml.Node) (err error) {
	var t time.Time
	if t, err = time.Parse(ExifDate, value.Value); err != nil {
		return
	}
	*ut = UnixJS(t)
	return
}

// UnixJS converts time to UNIX-time in milliseconds, compatible with javascript time format.
func UnixJS(u time.Time) unix_t {
	return unix_t(u.UnixNano() / 1000000)
}

// UnixJSNow returns same result as Date.now() in javascript.
func UnixJSNow() unix_t {
	return unix_t(time.Now().UnixNano() / 1000000)
}

// TimeJS is backward conversion from javascript compatible Unix time
// in milliseconds to golang structure.
func TimeJS(ujs unix_t) time.Time {
	return time.Unix(int64(ujs/1000), int64(ujs%1000)*1000000)
}

////////////////
// Routes API //
////////////////

// Router is local alias for router type.
type Router = mux.Router

// NewRouter is local alias for router creation function.
var NewRouter = mux.NewRouter

const (
	jsoncontent = "application/json; charset=utf-8"
	htmlcontent = "text/html; charset=utf-8"
	csscontent  = "text/css; charset=utf-8"
	jscontent   = "text/javascript; charset=utf-8"
)

// "Server" field for HTTP headers.
var serverlabel = fmt.Sprintf("dfs/%s (%s)", buildvers, runtime.GOOS)

// ParseBody fetch and unmarshal request argument.
func ParseBody(w http.ResponseWriter, r *http.Request, arg interface{}) (err error) {
	if jb, _ := io.ReadAll(r.Body); len(jb) > 0 {
		var ctype = r.Header.Get("Content-Type")
		if pos := strings.IndexByte(ctype, ';'); pos != -1 {
			ctype = ctype[:pos]
		}
		if ctype == "application/json" {
			if err = json.Unmarshal(jb, arg); err != nil {
				WriteError400(w, r, err, AECbadjson)
				return
			}
		} else if ctype == "application/x-yaml" || ctype == "application/yaml" {
			if err = yaml.Unmarshal(jb, arg); err != nil {
				WriteError400(w, r, err, AECbadyaml)
				return
			}
		} else if ctype == "application/xml" {
			if err = xml.Unmarshal(jb, arg); err != nil {
				WriteError400(w, r, err, AECbadxml)
				return
			}
		} else {
			WriteError400(w, r, ErrArgUndef, AECargundef)
			return
		}
	} else {
		err = ErrNoJSON
		WriteError400(w, r, err, AECnoreq)
		return
	}
	return
}

// WriteStdHeader setup common response headers.
func WriteStdHeader(w http.ResponseWriter) {
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Server", serverlabel)
}

// WriteHTMLHeader setup standard response headers for message with HTML content.
func WriteHTMLHeader(w http.ResponseWriter) {
	WriteStdHeader(w)
	w.Header().Set("X-Frame-Options", "sameorigin")
	w.Header().Set("Content-Type", htmlcontent)
}

// WriteRet writes to response given status code and marshaled body.
func WriteRet(w http.ResponseWriter, r *http.Request, status int, body interface{}) {
	if body == nil {
		w.WriteHeader(status)
		WriteStdHeader(w)
		return
	}
	var list []string
	if val := r.Header.Get("Accept"); val != "" {
		if pos := strings.IndexByte(val, ';'); pos != -1 {
			val = val[:pos]
		}
		list = strings.Split(val, ", ")
	} else {
		var ctype = r.Header.Get("Content-Type")
		if pos := strings.IndexByte(ctype, ';'); pos != -1 {
			ctype = ctype[:pos]
		}
		if ctype == "" {
			ctype = "application/json"
		}
		list = []string{ctype}
	}
	var b []byte
	var err error
	for _, ctype := range list {
		if ctype == "*/*" || ctype == "application/json" {
			WriteStdHeader(w)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			if b, err = json.Marshal(body); err != nil {
				break
			}
			w.Write(b)
			return
		} else if ctype == "application/x-yaml" || ctype == "application/yaml" {
			WriteStdHeader(w)
			w.Header().Set("Content-Type", ctype)
			w.WriteHeader(status)
			if b, err = yaml.Marshal(body); err != nil {
				break
			}
			w.Write(b)
			return
		} else if ctype == "application/xml" {
			WriteStdHeader(w)
			w.Header().Set("Content-Type", ctype)
			w.WriteHeader(status)
			if b, err = xml.Marshal(body); err != nil {
				break
			}
			w.Write(b)
			return
		}
	}
	WriteStdHeader(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	if err == nil {
		err = ErrBadEnc // no released encoding was found
	}
	b, _ = json.Marshal(MakeAjaxErr(err, AECbadenc))
	w.Write(b)
	return
}

// WriteOK puts 200 status code and some data to response.
func WriteOK(w http.ResponseWriter, r *http.Request, body interface{}) {
	WriteRet(w, r, http.StatusOK, body)
}

// WriteError puts to response given error status code and AjaxErr formed by given error object.
func WriteError(w http.ResponseWriter, r *http.Request, status int, err error, code int) {
	WriteRet(w, r, status, MakeAjaxErr(err, code))
}

// WriteError400 puts to response 400 status code and AjaxErr formed by given error object.
func WriteError400(w http.ResponseWriter, r *http.Request, err error, code int) {
	WriteRet(w, r, http.StatusBadRequest, MakeAjaxErr(err, code))
}

// WriteError500 puts to response 500 status code and AjaxErr formed by given error object.
func WriteError500(w http.ResponseWriter, r *http.Request, err error, code int) {
	WriteRet(w, r, http.StatusInternalServerError, MakeAjaxErr(err, code))
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
				grpclog.Errorln(str)
				WriteRet(w, r, http.StatusInternalServerError, MakeErrPanic(err, AECpanic, str))
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
	var api = gmux.PathPrefix("/api").Subrouter()
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
