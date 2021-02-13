// /home/krylon/go/src/ticker/feed/item.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-13 17:18:13 krylon>

package feed

import (
	"fmt"
	"ticker/common"
	"time"
)

// Item represents a single news item from an RSS Feed.
type Item struct {
	ID          int64
	FeedID      int64
	URL         string
	Title       string
	Description string
	Timestamp   time.Time
	Read        bool
	Rating      float64
}

func (i *Item) String() string {
	return fmt.Sprintf("Item{ ID: %d, URL: %q, Title: %q, Timestamp: %q, Rating: %d }",
		i.ID,
		i.URL,
		i.Title,
		i.Timestamp.Format(common.TimestampFormat),
		int(i.Rating*100))
} // func (i *Item) String() string
