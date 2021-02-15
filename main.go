// /home/krylon/go/src/ticker/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-15 14:19:10 krylon>

package main

import (
	"fmt"
	"os"
	"ticker/common"
	"ticker/reader"
	"ticker/web"
)

func main() {
	fmt.Printf("%s %s, built on %s\n",
		common.AppName,
		common.Version,
		common.BuildStamp)

	var (
		err  error
		rdr  *reader.Reader
		srv  *web.Server
		msgq = make(chan string, 5)
	)

	if err = common.InitApp(); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Cannot initialize directory %s: %s\n",
			common.BaseDir,
			err.Error(),
		)
		os.Exit(1)
	}

	if rdr, err = reader.New(msgq); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Cannot create RSS Reader: %s\n",
			err.Error())
		os.Exit(1)
	} else if srv, err = web.Create(":7777", true); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Cannnot create web server: %s\n",
			err.Error())
		os.Exit(1)
	}

	go forwardMsg(msgq, srv)

	go rdr.Loop()
	srv.ListenAndServe()
} // func main()

func forwardMsg(q <-chan string, srv *web.Server) {
	for {
		var m = <-q
		srv.SendMessage(m)
	}
} // func forwardMsg(q <-chan string, srv *web.Server)
