// /home/krylon/go/src/ticker/reader/00_reader_aux_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 08. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-08 13:32:30 krylon>

package reader

import (
	"ticker/common"
	"ticker/database"
	"ticker/feed"
	"time"
)

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

func prepareDatabase() error {
	var (
		err error
		db  *database.Database
	)

	if db, err = database.Open(common.DbPath); err != nil {
		return err
	}

	defer db.Close()

	db.Begin() // nolint: errcheck

	defer func() {
		if err != nil {
			db.Rollback() // nolint: errcheck
		} else {
			db.Commit() // nolint: errcheck
		}
	}()

	for _, f := range testFeeds {
		if err = db.FeedAdd(f); err != nil {
			return err
		}
	}

	return nil
} // func prepareDatabase() error
