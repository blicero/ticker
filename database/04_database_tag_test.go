// /home/krylon/go/src/ticker/database/04_database_tag_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 24. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-25 17:59:41 krylon>

package database

import (
	"testing"
	"ticker/feed"
	"ticker/tag"
)

func processTestTag(t testTag) []tag.Tag {
	var tags = []tag.Tag{t.t}

	for _, c := range t.children {
		tags = append(tags, processTestTag(c)...)
	}

	return tags
} // func processTestTag(t testTag) []tag.Tag

func TestTagCreate(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	var (
		err  error
		tags = make([]tag.Tag, 0)
	)

	for _, tt := range testTags {
		tags = append(tags, processTestTag(tt)...)
	}

	for _, tt := range tags {
		var label *tag.Tag
		if label, err = db.TagCreate(tt.Name, tt.Description, 0); err != nil {
			t.Errorf("Cannot create tag %s: %s",
				tt.Name,
				err.Error())
		} else if label == nil {
			t.Error("Database did not return fresh Tag")
		} else if label.Name != tt.Name {
			t.Errorf("New Tag has unexpected name %q (expected %q)",
				label.Name,
				tt.Name)
		}
	}
} // func TestTagCreate(t *testing.T)

func TestTagLinkCreate(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	// We are being deliberately stupid and just attach all existing tags
	// to all Items.
	var (
		err   error
		items []feed.Item
		tags  []tag.Tag
	)

	if items, err = db.ItemGetAll(-1, 0); err != nil {
		t.Fatalf("Cannot load all Items: %s", err.Error())
	} else if tags, err = db.TagGetAll(); err != nil {
		t.Fatalf("Cannot load all Tags: %s", err.Error())
	}

	for _, i := range items {
		for _, tt := range tags {
			if err = db.TagLinkCreate(i.ID, tt.ID); err != nil {
				t.Fatalf("Cannot attach Tag %s (%d) to Item %q (%d): %s",
					tt.Name,
					tt.ID,
					i.Title,
					i.ID,
					err.Error())
			}
		}
	}
} // func TestTagLinkCreate(t *testing.T)
