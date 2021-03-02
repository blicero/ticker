// /home/krylon/go/src/ticker/database/05_database_later_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 03. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-03-02 18:20:56 krylon>

package database

import (
	"fmt"
	"testing"
	"ticker/feed"
	"time"
)

func TestReadLaterCreate(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	var (
		err      error
		items    []feed.Item
		deadline = time.Now().Add(time.Hour * -24)
	)

	if items, err = db.ItemGetAll(-1, 0); err != nil {
		t.Fatalf("Cannot get all Items: %s",
			err.Error())
	}

	for _, item := range items {
		var later *feed.ReadLater

		if later, err = db.ReadLaterAdd(
			&item,
			fmt.Sprintf("Read Later %d", item.ID),
			deadline); err != nil {
			t.Fatalf("Error adding ReadLater note to Item %d (%s): %s",
				item.ID,
				item.Title,
				err.Error())
		} else if later.ItemID != item.ID {
			t.Fatalf("ReadLater.ItemID has unexpected Item ID %d (expected %d)",
				later.ItemID,
				item.ID)
		}
	}
} // func TestReadLaterCreate(t *testing.T)
