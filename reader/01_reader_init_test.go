// /home/krylon/go/src/ticker/reader/01_reader_init_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 07. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-08 13:37:14 krylon>

package reader

import "testing"

func TestReaderNew(t *testing.T) {
	var err error

	if rdr, err = New(); err != nil {
		rdr = nil
		t.Fatalf("Error creating Reader: %s",
			err.Error())
	}
} // func TestReaderNew(t *testing.T)
