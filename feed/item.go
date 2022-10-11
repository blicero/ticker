// /home/krylon/go/src/ticker/feed/item.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-07-02 09:03:00 krylon>

package feed

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"github.com/blicero/ticker/common"
	"github.com/blicero/ticker/tag"
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
	Tags          []tag.Tag
	tagMap        map[string]bool
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

// IsBoring returns true if the Item has been rated or classified as boring.
func (i *Item) IsBoring() bool {
	return i.Rating <= 0
} // func (i *Item) IsBoring() bool

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
} // func (i *Item) Plaintext() string

// HasTag returns true if the Tag with the given ID is attached to the
// receiver Item.
func (i *Item) HasTag(tagID int64) bool {
	for _, t := range i.Tags {
		if t.ID == tagID {
			return true
		}
	}

	return false
} // func (i *Item) HasTag(tagID int64) bool

// HasTagNamed returns true if the receiver carries the Tag with the given name.
func (i *Item) HasTagNamed(name string) bool {
	if len(i.Tags) == 0 {
		return false
	} else if len(i.tagMap) != len(i.Tags) {
		i.tagMap = make(map[string]bool, len(i.Tags))

		for _, t := range i.Tags {
			i.tagMap[t.Name] = true
		}
	}

	return i.tagMap[name]
} // func (i *Item) HasTagNamed(name string) bool

// IsDownloaded returns true if the Item's linked URL has been downloaded
// to the local archive.
func (i *Item) IsDownloaded() (result bool) {
	var (
		err        error
		info       os.FileInfo
		pageFolder = filepath.Join(
			common.ArchiveDir,
			strconv.FormatInt(i.ID, 10))
	)

	defer func() {
		if x := recover(); x != nil {
			result = false
		}
	}()

	if info, err = os.Stat(pageFolder); err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Printf("XXX Error checking if page %d (%q) has been downloaded to %s: %s\n",
			i.ID,
			i.Title,
			pageFolder,
			err.Error())
		return false
	}

	return info.IsDir()
} // func (i *Item) IsDownloaded() bool
