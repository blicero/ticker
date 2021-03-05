// /home/krylon/go/src/ticker/feed/later.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 03. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-03-04 18:05:46 krylon>

package feed

import "time"

// ReadLater is a note/reminder to read a particular news Item at a later time.
type ReadLater struct {
	ID        int64
	Item      *Item
	ItemID    int64
	Note      string
	Timestamp time.Time
	Deadline  time.Time
	Read      bool
}

// IsDue returns true if the receiver's deadline has passed.
func (r *ReadLater) IsDue() bool {
	return !r.Read && r.Deadline.Before(time.Now())
} // func (r *ReadLater) IsDue() bool
