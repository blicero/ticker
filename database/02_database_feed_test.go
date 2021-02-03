// /home/krylon/go/src/ticker/database/02_database_feed_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-03 19:59:25 krylon>

package database

import (
	"testing"
	"ticker/feed"
)

func TestFeedAdd(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	for _, f := range list {
		var err error

		if err = db.FeedAdd(f); err != nil {
			t.Errorf("Cannot add Feed %s: %s",
				f.Name,
				err.Error())
		} else if f.ID == 0 {
			t.Errorf("Error adding Feed %s: No ID",
				f.Name)
		}
	}
} // func TestFeedAdd(t *testing.T)

func TestFeedGetAll(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	var (
		err   error
		feeds []feed.Feed
	)

	if feeds, err = db.FeedGetAll(); err != nil {
		t.Fatalf("Cannot get all Feeds: %s", err.Error())
	} else if len(feeds) != len(list) {
		t.Fatalf("FeedGetAll returned an unexpected number of Feeds: %d (expected %d)",
			len(feeds),
			len(list))
	}

	var ref = make(map[int64]*feed.Feed, len(list))

	for _, f := range list {
		ref[f.ID] = f
	}

	for _, f := range feeds {
		var (
			r  *feed.Feed
			ok bool
		)

		if r, ok = ref[f.ID]; !ok {
			t.Fatalf("GetFeedAll returned unknown Feed %s", f.Name)
		} else if !feedEqual(&f, r) {
			t.Fatalf(`Feeds are not equal:
Database:	%s
Expected:       %s
`,
				&f,
				r)
		}
	}
} // func TestFeedGetAll(t *testing.T)
