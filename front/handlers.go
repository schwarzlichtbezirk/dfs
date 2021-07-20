package main

import (
	"errors"
	"io"
	"log"
	"net/http"
)

// API error codes.
// Each error code have unique source code point,
// so this error code at service reply exactly points to error place.
const (
	AECnull    = 0
	AECbadbody = 1
	AECnoreq   = 2
	AECbadjson = 3

	// upload
	AECuploadform = 4
)

// HTTP error messages
var (
	ErrNoJSON = errors.New("data not given")
	ErrNoData = errors.New("data is empty")
)

// APIHANDLER
func apiPing(w http.ResponseWriter, r *http.Request) {
	var body, _ = io.ReadAll(r.Body)
	w.WriteHeader(http.StatusOK)
	WriteJSONHeader(w)
	w.Write(body)
}

func apiUpload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	var file, handler, err = r.FormFile("datafile")
	if err != nil {
		WriteError400(w, err, AECuploadform)
		return
	}
	defer file.Close()
	var mime = "N/A"
	if ct, ok := handler.Header["Content-Type"]; ok && len(ct) > 0 {
		mime = ct[0]
	}
	log.Printf("upload file: %s, size: %d, mime: %s\n", handler.Filename, handler.Size, mime)

	WriteOK(w, nil)
}
