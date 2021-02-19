// /home/krylon/go/src/ticker/feed/item.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-19 13:20:47 krylon>

package feed

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"ticker/common"
	"time"

	"github.com/jaytaylor/html2text"
)

var whitespace *regexp.Regexp = regexp.MustCompile(`[\s\t\n\r]+`)

// Item represents a single news item from an RSS Feed.
type Item struct {
	ID            int64
	FeedID        int64
	URL           string
	Title         string
	Description   string
	Timestamp     time.Time
	Read          bool
	Rating        float64
	ManuallyRated bool
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
	return i.ManuallyRated
} // func (i *Item) IsRated() bool

// RatingString returns the Item's rating as a string.
func (i *Item) RatingString() string {
	if math.IsNaN(i.Rating) {
		return ""
	}

	return fmt.Sprintf("%.2f", i.Rating)
} // func (i *Item) RatingString() string

// Plaintext returns the complete text of the Item, cleansed of any HTML.
func (i *Item) Plaintext() string {
	var tmp = make([]string, 2)
	var err error

	if tmp[0], err = html2text.FromString(i.Title); err != nil {
		tmp[0] = i.Title
	}

	if tmp[1], err = html2text.FromString(i.Description); err != nil {
		tmp[1] = i.Description
	}

	tmp[0] = whitespace.ReplaceAllString(tmp[0], " ")
	tmp[1] = whitespace.ReplaceAllString(tmp[1], " ")

	return strings.Join(tmp, " ")
} // func (self *NewsItem) Plaintext() string
