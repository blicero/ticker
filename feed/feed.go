// /home/krylon/go/src/ticker/feed/feed.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-01 15:50:36 krylon>

// Package feed provides the basic data type and logic to represent and interact
// with RSS feeds.
package feed

import (
	"errors"
	"log"
	"ticker/common"
	"ticker/logdomain"
	"time"

	"github.com/SlyMarbo/rss"
)

// ErrInactive indicates that a Feed is not active.
var ErrInactive = errors.New("feed is not active")

// Feed represents an RSS feed.
type Feed struct {
	ID         int64
	Name       string
	URL        string
	Interval   time.Duration
	LastUpdate time.Time
	Active     bool
	rfeed      *rss.Feed
	log        *log.Logger
}

// New creates a new Feed.
func New(id int64, name, url string, interval time.Duration, active bool) (*Feed, error) {
	var err error
	var f = &Feed{
		ID:       id,
		Name:     name,
		URL:      url,
		Interval: interval,
		Active:   active,
	}

	if f.log, err = common.GetLogger(logdomain.Feed); err != nil {
		return nil, err
	}

	return f, nil
} // func New(name, url string, interval time.Duration, active bool) (*Feed, error)

// IsDue returns true if the Feed is due for a refresh.
func (f *Feed) IsDue() bool {
	return !f.Next().After(time.Now())
} // func (f *Feed) IsDue() bool

// Next returns the Timestamp when the Feed is next due for a refresh.
func (f *Feed) Next() time.Time {
	return f.LastUpdate.Add(f.Interval)
} // func (f *Feed) Next() time.Time

// Fetch fetches a Feed.
func (f *Feed) Fetch() (*rss.Feed, error) {
	var (
		err error
		fd  *rss.Feed
	)

	if !f.Active {
		return nil, ErrInactive
	}

	if f.rfeed != nil {
		fd = f.rfeed
	} else if fd, err = rss.Fetch(f.URL); err != nil {
		f.log.Printf("[ERROR] Error fetching %s (%s): %s\n",
			f.Name,
			f.URL,
			err.Error())
		return nil, err
	} else {
		f.rfeed = fd
		f.LastUpdate = time.Now()
	}

	if f.IsDue() {
		if err = fd.Update(); err != nil {
			f.log.Printf("[ERROR] Cannot update %s (%s): %s\n",
				f.Name,
				f.URL,
				err.Error())
			return nil, err
		}
	}

	return fd, nil
} // func (f *Feed) Fetch() (*rss.Feed, error)
