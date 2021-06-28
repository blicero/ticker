// /home/krylon/go/src/ticker/download/01_download_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 28. 06. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-28 23:24:52 krylon>

package download

import (
	"testing"
	"ticker/feed"
)

var dl *Agent

func TestCreateDownloader(t *testing.T) {
	var err error

	if dl, err = NewAgent(1); err != nil {
		dl = nil
		t.Fatalf("Error creating Agent: %s",
			err.Error())
	}

	// dl.Start()
} // func TestCreateDownloader(t *testing.T)

func TestDownload(t *testing.T) {
	var addr = urlRoot + "/index.html"
	var item = feed.Item{
		ID:  42,
		URL: addr,
	}

	dl.processPage(&item)
} // func TestDownload(t *testing.T)
