// /home/krylon/go/src/ticker/prefetch/00_srv_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 06. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-02 14:22:27 krylon>

package prefetch

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
)

const imgPath = "testdata/sleepy_bear.jpg"

var srv *httptest.Server
var img []byte

func readFile() error {
	var (
		err error
		fh  *os.File
		buf bytes.Buffer
	)

	if fh, err = os.Open(imgPath); err != nil {
		return err
	}

	defer fh.Close() // nolint: errcheck

	if _, err = io.Copy(&buf, fh); err != nil {
		return err
	}

	img = buf.Bytes()

	return nil
} // func readFile() error

func startServer() error {
	var (
		err error
	)

	if err = readFile(); err != nil {
		return err
	}

	srv = httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(200)
			w.Write(img) // nolint: errcheck
		}))
	srv.EnableHTTP2 = false
	srv.Start()
	return nil
} // func startServer() error
