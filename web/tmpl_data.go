// /home/krylon/go/src/ticker/web/tmpl_data.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 05. 2020 by Benjamin Walkenhorst
// (c) 2020 Benjamin Walkenhorst
// Time-stamp: <2021-02-14 21:26:50 krylon>
//
// This file contains data structures to be passed to HTML templates.

package web

import (
	"crypto/sha512"
	"fmt"
	"ticker/common"
	"ticker/feed"
	"time"

	"github.com/hashicorp/logutils"
)

type message struct {
	Timestamp time.Time
	Level     logutils.LogLevel
	Message   string
}

func (m *message) TimeString() string {
	return m.Timestamp.Format(common.TimestampFormat)
} // func (m *Message) TimeString() string

func (m *message) Checksum() string {
	var str = m.Timestamp.Format(common.TimestampFormat) + "##" +
		string(m.Level) + "##" +
		m.Message

	var hash = sha512.New()
	hash.Write([]byte(str)) // nolint: gosec,errcheck

	var cksum = hash.Sum(nil)
	var ckstr = fmt.Sprintf("%x", cksum)

	return ckstr
} // func (m *message) Checksum() string

type tmplDataBase struct {
	Title      string
	Messages   []message
	Debug      bool
	TestMsgGen bool
	URL        string
}

type tmplDataIndex struct {
	tmplDataBase
	Feeds []feed.Feed
	Items []feed.Item
}

type tmplDataFeedDetails struct {
	tmplDataBase
	Feed  *feed.Feed
	Items []feed.Item
}

// Local Variables:  //
// compile-command: "go generate && go vet && go build -v -p 16 && gometalinter && go test -v" //
// End: //
