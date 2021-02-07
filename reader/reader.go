// /home/krylon/go/src/ticker/reader/reader.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-07 18:27:58 krylon>

// Package reader implements the periodic updates of RSS feeds.
package reader

import (
	"fmt"
	"log"
	"os"
	"ticker/common"
	"ticker/database"
	"ticker/feed"
	"ticker/logdomain"
	"time"

	deadlock "github.com/sasha-s/go-deadlock"
)

const checkDelay = time.Second // nolint: deadcode,varcheck,unused

// Reader regularly checks the subscribed Feeds and stores any new Items in
// the database.
type Reader struct {
	db     *database.Database
	log    *log.Logger
	active bool
	lock   deadlock.RWMutex
}

// New creates a new Reader.
func New() (*Reader, error) {
	var (
		err error
		r   = new(Reader)
	)

	if r.log, err = common.GetLogger(logdomain.Reader); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create Logger for %s: %s\n",
			logdomain.Reader,
			err.Error())
		return nil, err
	} else if r.db, err = database.Open(common.DbPath); err != nil {
		r.log.Printf("[ERROR] Cannot open database at %s: %s\n",
			common.DbPath,
			err.Error())
		return nil, err
	}

	return r, nil
} // func New() (*Reader, error)

// Active returns true if the Reader is still active.
func (r *Reader) Active() bool {
	r.lock.RLock()
	var status = r.active
	r.lock.RUnlock()
	return status
} // func (r *Reader) Active() bool

// Start sets the Reader to active and starts its main loop.
func (r *Reader) Start() {
	r.lock.Lock()
	r.active = true
	r.lock.Unlock()

	go r.Loop() //nolint: errcheck
} // func (r *Reader) Start()

// Stop tells the Reader to stop.
func (r *Reader) Stop() {
	r.lock.Lock()
	r.active = false
	r.lock.Unlock()
} // func (r *Reader) Stop()

// Loop implements the Reader's main loop.
func (r *Reader) Loop() error {
	// const maxErrCnt = 10
	// var errCnt = 0

	defer func() {
		r.lock.Lock()
		r.active = false
		r.lock.Unlock()
	}()

	for r.Active() {
		if err := r.refresh(); err != nil {
			r.log.Printf("[ERROR] Failed to refresh Feeds: %s\n",
				err.Error())
			return err
		}
	}

	return nil
} // func (r *Reader) Loop() error

func (r *Reader) refresh() error {
	var (
		err   error
		feeds []feed.Feed
	)

	r.log.Println("[TRACE] Check/Refresh Feeds")

	if feeds, err = r.db.FeedGetAll(); err != nil {
		r.log.Printf("[ERROR] Cannot get all Feeds: %s\n",
			err.Error())
		return err
	}

	for _, f := range feeds {
		var items []feed.Item

		r.log.Printf("[TRACE] Check Feed %s\n", f.Name)

		if !f.IsDue() {
			r.log.Printf("[TRACE] Feed %s is next due at %s\n",
				f.Name,
				f.Next().Format(common.TimestampFormat))
			continue
		} else if items, err = f.Fetch(); err != nil {
			r.log.Printf("[ERROR] Failed to refresh Feed %s: %s\n",
				f.Name,
				err.Error())
			continue
		}

		r.log.Printf("[TRACE] Feed %s: Process %d Items\n",
			f.Name,
			len(items))

		for _, i := range items {
			r.log.Printf("[TRACE] Process Item %s (%s)\n",
				i.Title,
				i.URL)
		}
	}

	return nil
} // func (r *Reader) refresh() error
