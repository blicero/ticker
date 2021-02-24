// /home/krylon/go/src/ticker/database/04_database_tag_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 24. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-24 21:01:30 krylon>

package database

import (
	"testing"
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
