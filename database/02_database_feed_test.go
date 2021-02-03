// /home/krylon/go/src/ticker/database/02_database_feed_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-02 21:28:20 krylon>

package database

import (
	"testing"
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
