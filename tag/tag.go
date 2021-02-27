// /home/krylon/go/src/ticker/tag/tag.go
// -*- mode: go; coding: utf-8; -*-
// Created on 24. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-26 19:25:33 krylon>

// Package tag provides ... well, Tags to attach to Items.
//
// Tags can have parents, so they can form a hierarchy.
package tag

// Tag is a label one can attach to Items.
type Tag struct {
	ID          int64
	Name        string
	Description string
	Parent      int64
	Children    []Tag
}
