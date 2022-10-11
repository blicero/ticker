// /home/krylon/go/src/ticker/database/02_database_feed_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-04 18:56:43 krylon>

package database

import (
	"testing"
	"github.com/blicero/ticker/feed"
	"time"
)

func TestFeedAdd(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	for _, f := range testFeeds {
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
	} else if len(feeds) != len(testFeeds) {
		t.Fatalf("FeedGetAll returned an unexpected number of Feeds: %d (expected %d)",
			len(feeds),
			len(testFeeds))
	}

	var ref = make(map[int64]*feed.Feed, len(testFeeds))

	for _, f := range testFeeds {
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

func TestFeedGetByID(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	for _, r := range testFeeds {
		var (
			err error
			f   *feed.Feed
		)

		if f, err = db.FeedGetByID(r.ID); err != nil {
			t.Errorf("Cannot get Feed %s by ID (%d): %s",
				r.Name,
				r.ID,
				err.Error())
		} else if f == nil {
			t.Errorf("Did not find Feed %s by ID (%d)",
				r.Name,
				r.ID)
		} else if !feedEqual(r, f) {
			t.Errorf(`Feed %s as returned by FeedGetByID does not equal reference Feed:
Expected: %s
Got:      %s
`,
				r.Name,
				r,
				f)
		}
	}
} // func TestFeedGetByID(t *testing.T)

func TestFeedSetTimestamp(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	for _, r := range testFeeds {
		var (
			err error
			f   *feed.Feed
			now = time.Now()
		)

		if err = db.FeedSetTimestamp(r, now); err != nil {
			t.Errorf("Cannot set Timestamp for Feed %s (%d): %s",
				r.Name,
				r.ID,
				err.Error())
		} else if f, err = db.FeedGetByID(r.ID); err != nil {
			t.Errorf("Cannot get Feed %s by ID (%d): %s",
				r.Name,
				r.ID,
				err.Error())
		} else if f == nil {
			t.Errorf("Did not find Feed %s by ID (%d)",
				r.Name,
				r.ID)
		} else if !feedEqual(r, f) {
			t.Errorf(`Feed %s as returned by FeedGetByID does not equal reference Feed:
Expected: %s
Got:      %s
`,
				r.Name,
				r,
				f)
		}
	}
} // func TestFeedSetTimestamp(t *testing.T)

func TestFeedDelete(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	var err error

	for _, f := range testFeeds {
		if err = db.FeedDelete(f.ID); err != nil {
			t.Fatalf("Error deleting Feed %s (%d): %s",
				f.Name,
				f.ID,
				err.Error())
		}
	}

	var feeds []feed.Feed

	if feeds, err = db.FeedGetAll(); err != nil {
		t.Fatalf("Error getting all Feeds from database: %s",
			err.Error())
	} else if len(feeds) != 0 {
		t.Fatalf("FeedGetAll returned unexpected number of Feeds after deleting all Feeds: %d (expected 0)",
			len(feeds))
	}

	for _, f := range testFeeds {
		f.ID = 0

		if err = db.FeedAdd(f); err != nil {
			t.Fatalf("Error re-adding Feed %s: %s",
				f.Name,
				err.Error())
		}
	}

	if feeds, err = db.FeedGetAll(); err != nil {
		t.Fatalf("Error getting all Feeds from database: %s",
			err.Error())
	} else if len(feeds) != len(testFeeds) {
		t.Fatalf("FeedGetAll returned unexpected number of Feeds after deleting all Feeds: %d (expected %d)",
			len(feeds),
			len(testFeeds))
	}
} // func TestFeedDelete(t *testing.T)
