// /home/krylon/go/src/ticker/web/01_web_create_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 03. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-03-11 11:32:12 krylon>

package web

import "testing"

const addr = "[::1]:7766"

var srv *Server

func TestServerCreate(t *testing.T) {
	var err error

	if srv, err = Create(addr, true); err != nil {
		srv = nil
		t.Fatalf("Cannot create Server: %s",
			err.Error())
	}
} // func TestServerCreate(t *testing.T)
