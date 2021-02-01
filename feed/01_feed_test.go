// /home/krylon/go/src/ticker/feed/01_feed_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-01 16:23:35 krylon>

package feed

import (
	"testing"
	"time"

	"github.com/SlyMarbo/rss"
)

func TestFeedIsDue(t *testing.T) {
	type testCase struct {
		f   Feed
		due bool
	}

	var cases = []testCase{
		testCase{
			f: Feed{
				Name:       "test01",
				Interval:   time.Minute * 10,
				LastUpdate: time.Date(2021, time.January, 1, 12, 0, 0, 0, time.Local),
			},
			due: true,
		},
		testCase{
			f: Feed{
				Name:       "test02",
				Interval:   time.Minute * 30,
				LastUpdate: time.Now(),
			},
			due: false,
		},
	}

	for _, c := range cases {
		if c.due != c.f.IsDue() {
			t.Errorf("Error in IsDue() for Feed %s: %t (expect %t)",
				c.f.Name,
				c.f.IsDue(),
				c.due)
		}
	}
} // func TestFeedIsDue(t *testing.T)

func TestFeedFetch(t *testing.T) {
	type testCase struct {
		feed Feed
		due  bool
		err  bool
	}

	var cases = []testCase{
		testCase{
			feed: Feed{
				ID:       1,
				Name:     "No Such Feed",
				URL:      "http://www.example.com/feed.xml",
				Interval: time.Minute,
				Active:   true,
				log:      flog,
			},
			due: true,
			err: true,
		},
		testCase{
			feed: Feed{
				ID:       2,
				Name:     "WDR Nachrichten",
				URL:      "http://www1.wdr.de/wissen/uebersicht-nachrichten-100.feed",
				Interval: time.Minute * 15,
				Active:   true,
				log:      flog,
			},
			due: true,
			err: false,
		},
		testCase{
			feed: Feed{
				ID:       3,
				Name:     "WDR Nachrichten",
				URL:      "http://www1.wdr.de/wissen/uebersicht-nachrichten-100.feed",
				Interval: time.Minute * 15,
				Active:   false,
				log:      flog,
			},
			due: true,
			err: true,
		},
		testCase{
			feed: Feed{
				ID:       4,
				Name:     "Tagesschau",
				URL:      "http://www.tagesschau.de/xml/rss2",
				Interval: time.Minute * 15,
				Active:   true,
				log:      flog,
			},
			due: true,
			err: false,
		},
	}

	for _, c := range cases {
		var (
			err error
			f   *rss.Feed
		)

		if f, err = c.feed.Fetch(); err != nil {
			if c.err {
				continue
			}

			t.Errorf("Error fetching Feed %d (%s): %s",
				c.feed.ID,
				c.feed.Name,
				err.Error())
		} else if f == nil {
			t.Errorf("Error fetching Feed %d (%s): Fetch returned nil",
				c.feed.ID,
				c.feed.Name)
		}
	}
} // func TestFeedFetch(t *testing.T)
