// /home/krylon/go/src/ticker/feed/feed.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-01 09:57:38 krylon>

// Package feed provides the basic data type and logic to represent and interact
// with RSS feeds.
package feed

import "time"

// Feed represents an RSS feed.
type Feed struct {
	ID         int64
	Name       string
	URL        string
	Interval   time.Duration
	LastUpdate time.Time
	Active     bool
}

// IsDue returns true if the Feed is due for a refresh.
func (f *Feed) IsDue() bool {
	return !f.LastUpdate.Add(f.Interval).After(time.Now())
} // func (f *Feed) IsDue() bool
