// /home/krylon/go/src/ticker/database/00_database_data_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-03 20:16:04 krylon>

package database

import (
	"ticker/feed"
	"time"
)

var db *Database

// Some might argue that using live feeds for testing is a bad idead
// on various levels.
// I'm going to do it anyway.
// However, I am going to use only RSS feeds by German public broadcast
// stations, since I already support them by paying Rundfunkgebühren.

var testFeeds = []*feed.Feed{
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
	&feed.Feed{
		Name:     "NDR Nachrichten",
		URL:      "http://www.ndr.de/home/index-rss.xml",
		Interval: time.Minute * 60,
		Active:   true,
	},
}
