// /home/krylon/go/src/ticker/database/03_database_item_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-05 16:23:55 krylon>

package database

import (
	"fmt"
	"net/url"
	"testing"
	"github.com/blicero/ticker/feed"
	"time"
)

const testItemCnt = 100

func TestItemAdd(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	const (
		interval = time.Minute * 15
	)
	var baseDate = time.Now().Add(interval * -testItemCnt)

	for _, f := range testFeeds {
		var stamp = baseDate
		for i := 0; i < testItemCnt; i++ {
			var err error
			var item = feed.Item{
				FeedID: f.ID,
				URL: fmt.Sprintf("https://www.example.com/%s/item%04d",
					url.PathEscape(f.Name),
					i),
				Title: fmt.Sprintf("News Item %04d", i),
				Description: fmt.Sprintf("This is test News Item %s %04d",
					f.Name,
					i),
				Timestamp: stamp,
			}

			if err = db.ItemAdd(&item); err != nil {
				t.Fatalf("Error adding Item %s/%04d: %s",
					f.Name,
					i,
					err.Error())
			} else if item.ID == 0 {
				t.Fatalf("News Item %s/%04d has no ID",
					f.Name,
					i)
			}

			stamp = stamp.Add(interval)
		}
	}
} // func TestItemAdd(t *testing.T)

func TestItemGetRecent(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	var (
		err   error
		items []feed.Item
		limit = testItemCnt * len(testFeeds)
	)

	if items, err = db.ItemGetRecent(limit); err != nil {
		t.Fatalf("Error getting %d recent items: %s",
			limit,
			err.Error())
	} else if len(items) != limit {
		t.Fatalf("Unexpected number of Items: %d (expected %d)",
			len(items),
			limit)
	}
} // func TestItemGetRecent(t *testing.T)
