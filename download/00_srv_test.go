// /home/krylon/go/src/ticker/download/00_srv_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 28. 06. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-28 23:26:11 krylon>

package download

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

var srv *httptest.Server
var urlRoot string
var pat = regexp.MustCompile("[.]([^.]+)$")

const assetFolder = "testdata"

func handleRequest(w http.ResponseWriter, r *http.Request) {
	var (
		err                        error
		localpath, rpath, mimetype string
		match                      []string
		fh                         *os.File
	)

	rpath = r.URL.EscapedPath()

	if match = pat.FindStringSubmatch(rpath); match == nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(404)
		w.Write([]byte(fmt.Sprintf("Could not determine MIME type for %s", rpath))) // nolint: errcheck
		return
	}

	switch match[1] {
	case "jpg":
		mimetype = "image/jpeg"
	case "js":
		mimetype = "text/javascript"
	case "html":
		mimetype = "text/html"
	default:
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(404)
		w.Write([]byte(fmt.Sprintf("Could not determine MIME type for %s", match[1]))) // nolint: errcheck
		return
	}

	localpath = filepath.Join(assetFolder, path.Base(rpath))

	if fh, err = os.Open(localpath); err != nil {
		var msg = fmt.Sprintf(
			"Cannot open %s: %s",
			localpath,
			err.Error())
		fmt.Fprintf(os.Stderr, "%s\n", msg)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(404)
		w.Write([]byte(msg)) // nolint: errcheck
		return
	}

	defer fh.Close()

	w.Header().Set("Content-Type", mimetype)
	w.WriteHeader(200)
	io.Copy(w, fh) // nolint: errcheck
} // func handleRequest(w http.ResponseWriter, r *http.Request)

func startServer() error {
	srv = httptest.NewUnstartedServer(http.HandlerFunc(handleRequest))
	srv.EnableHTTP2 = false
	srv.Start()
	urlRoot = srv.URL
	return nil
} // func startServer() error
