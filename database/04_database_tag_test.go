// /home/krylon/go/src/ticker/database/04_database_tag_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 24. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-27 18:26:37 krylon>

package database

import (
	"testing"
	"ticker/feed"
	"ticker/tag"

	"github.com/davecgh/go-spew/spew"
)

func processTag(t *tag.Tag) error {
	var (
		err error
		tt  *tag.Tag
	)

	if tt, err = db.TagCreate(t.Name, t.Description, t.Parent); err != nil {
		return err
	}

	t.ID = tt.ID

	for idx := range t.Children {
		tt = &t.Children[idx]
		tt.Parent = t.ID
		if err = processTag(tt); err != nil {
			return err
		}
	}

	return nil
} // func processTag(t *tag.Tag) error

func TestTagCreate(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	var (
		err error
	)

	for idx := range testTags {
		var tt = &testTags[idx]
		if err = processTag(tt); err != nil {
			t.Fatalf("Error processing Tag %s: %s",
				tt.Name,
				err.Error())
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

func TestTagChildren(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	const (
		name     = "IT"
		childCnt = 8
	)

	var (
		err      error
		label    *tag.Tag
		children []tag.Tag
	)

	if label, err = db.TagGetByName(name); err != nil {
		t.Fatalf("Cannot load Tag %s: %s",
			name,
			err.Error())
	} else if children, err = db.TagGetChildren(label.ID); err != nil {
		t.Fatalf("Cannot load Children of Tag %s (%d): %s",
			label.Name,
			label.ID,
			err.Error())
	} else if len(children) != childCnt {
		t.Errorf("Unexpected number of children for Tag %s: %d (expected %d)",
			label.Name,
			len(children),
			childCnt)
	}
} // func TestTagChildren(t *testing.T)

func TestTagHierarchy(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	var (
		err  error
		tags []tag.Tag
	)

	if tags, err = db.TagGetHierarchy(); err != nil {
		t.Fatalf("Cannot get Tag hierarchy: %s", err.Error())
	} else if len(tags) != len(testTags) {
		t.Fatalf("Unexpected number of root tags: %d (expected %d)",
			len(tags),
			len(testTags))
	}

	t.Logf("Tag hierarchy:\n%s\n",
		spew.Sdump(tags))
} // func TestTagHierarchy(t *testing.T)
