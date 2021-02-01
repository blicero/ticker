// /home/krylon/go/src/ticker/feed/01_feed_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-01 10:13:24 krylon>

package feed

import (
	"testing"
	"time"
)

func TestIsDue(t *testing.T) {
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
} // func TestIsDue(t *testing.T)
