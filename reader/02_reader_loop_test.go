// /home/krylon/go/src/ticker/reader/02_reader_loop_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 08. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-08 19:31:56 krylon>

package reader

import (
	"testing"
	"time"
)

func TestReaderLoop(t *testing.T) {
	if rdr == nil {
		t.Log("Reader has not been initialized. Bail.\n")
		t.SkipNow()
	}

	go func() {
		time.Sleep(checkDelay * 5)
		rdr.Stop()
	}()

	rdr.active = true

	var err error

	if err = rdr.Loop(); err != nil {
		t.Fatalf("Failed to refresh Feeds: %s",
			err.Error())
	}
} // func TestReaderLoop(t *testing.T)
