// /home/krylon/go/src/ticker/web/web.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-11 17:32:35 krylon>

package web

import (
	"html/template"
	"krylib"
	"log"
	"net/http"
	"zettelkasten/database"
)

//go:generate go run ./build_templates.go

// Server implements the web interface
type Server struct {
	Addr   string
	web    http.Server
	log    *log.Logger
	msgBuf *krylib.MessageBuffer
	router *mux.router
	tmpl   *template.Template
	pool   *database.Pool
}
