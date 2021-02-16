// /home/krylon/go/src/ticker/feed/item.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-16 17:43:50 krylon>

package feed

import (
	"fmt"
	"math"
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

// IsRated returns true if the Item has a Rating.
func (i *Item) IsRated() bool {
	return !math.IsNaN(i.Rating)
} // func (i *Item) IsRated() bool

// RatingString returns the Item's rating as a string.
func (i *Item) RatingString() string {
	if math.IsNaN(i.Rating) {
		return ""
	}

	return fmt.Sprintf("%.2f", i.Rating)
} // func (i *Item) RatingString() string
