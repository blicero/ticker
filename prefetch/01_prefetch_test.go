// /home/krylon/go/src/ticker/prefetch/01_prefetch_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 06. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-03 12:33:31 krylon>

package prefetch

import (
	"testing"
	"ticker/common"
	"ticker/database"
	"ticker/feed"
)

var (
	pre *Prefetcher
	tdb *database.Database
)

func TestSanitize(t *testing.T) {
	var (
		err   error
		items []feed.Item
	)

	if tdb, err = database.Open(common.DbPath); err != nil {
		t.Fatalf("Error opening database: %s", err.Error())
	} else if pre, err = Create(1); err != nil {
		tdb.Close() // nolint: errcheck
		tdb = nil
		t.Fatalf("Error creating Prefetcher: %s", err.Error())
	} else if items, err = tdb.ItemGetPrefetch(batchSize); err != nil {
		t.Fatalf("Cannot fetch Items: %s", err.Error())
	} else if len(items) != batchSize {
		t.Errorf("Unexpected number of items: %d (expected %d)",
			len(items),
			batchSize)
	}

	for _, i := range items {
		var body string
		if body, err = pre.sanitize(&i); err != nil {
			t.Errorf("Error sanitizing Item %d (%s): %s",
				i.ID,
				i.Title,
				err.Error())
		}

		t.Logf("Sanitized body of item %d: %s",
			i.ID,
			body)
	}
} // func TestSanitize(t *testing.T)
