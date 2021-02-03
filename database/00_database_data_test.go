// /home/krylon/go/src/ticker/database/00_database_data_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-02 21:28:15 krylon>

package database

import (
	"ticker/feed"
	"time"
)

var db *Database

var list = []*feed.Feed{
	&feed.Feed{
		Name:     "Tagesschau",
		URL:      "http://www.tagesschau.de/xml/rss2",
		Interval: time.Minute * 15,
		Active:   true,
	},
	&feed.Feed{
		Name:     "Deutschlandfunk Nachrichten",
		URL:      "https://www.deutschlandfunk.de/die-nachrichten.353.de.rss",
		Interval: time.Minute * 60,
		Active:   true,
	},
}
