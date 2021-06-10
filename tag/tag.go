// /home/krylon/go/src/ticker/tag/tag.go
// -*- mode: go; coding: utf-8; -*-
// Created on 24. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-10 18:57:51 krylon>

// Package tag provides ... well, Tags to attach to Items.
//
// Tags can have parents, so they can form a hierarchy.
package tag

import "strings"

// Tag is a label one can attach to Items.
type Tag struct {
	ID          int64
	Name        string
	Description string
	Parent      int64
	Level       int
	Path        string
	Children    []Tag
}

func (t *Tag) SortName() string {
	return strings.Repeat(" ", int(t.Level)) + t.Name
} // func (t *Tag) SortName() string
