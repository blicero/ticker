// /home/krylon/go/src/ticker/reader/reader.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-15 14:03:49 krylon>

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
	db       *database.Database
	log      *log.Logger
	active   bool
	lock     deadlock.RWMutex
	msgQueue chan<- string
}

// New creates a new Reader.
func New(q chan<- string) (*Reader, error) {
	var (
		err error
		msg string
		r   = &Reader{msgQueue: q}
	)

	if r.log, err = common.GetLogger(logdomain.Reader); err != nil {
		msg = fmt.Sprintf("Cannot create Logger for %s: %s",
			logdomain.Reader,
			err.Error())
		q <- msg
		fmt.Fprintln(os.Stderr, msg)
		return nil, err
	} else if r.db, err = database.Open(common.DbPath); err != nil {
		msg = fmt.Sprintf("Cannot open database at %s: %s",
			common.DbPath,
			err.Error())
		r.log.Printf("[ERROR] %s\n", msg)
		q <- msg
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

	r.lock.Lock()
	r.active = true
	r.lock.Unlock()

	defer func() {
		r.lock.Lock()
		r.active = false
		r.lock.Unlock()

		r.log.Println("[TRACE] Reader.Loop() is finished.")
		r.msgQueue <- "Reader Loop is finished"
	}()

	for r.Active() {
		r.log.Println("[TRACE] Reader Loop calling refresh.")

		if err := r.refresh(); err != nil {
			var msg = fmt.Sprintf("Failed to refresh Feeds: %s",
				err.Error())
			r.log.Printf("[ERROR] %s\n", msg)
			r.msgQueue <- msg
			return err
		}

		time.Sleep(checkDelay)
	}

	return nil
} // func (r *Reader) Loop() error

func (r *Reader) refresh() error {
	var (
		err   error
		feeds []feed.Feed
	)

	r.log.Println("[TRACE] Check/Refresh Feeds")

	defer func() {
		r.log.Println("[TRACE] Reader.refresh() is finished.")
	}()

	if feeds, err = r.db.FeedGetAll(); err != nil {
		var msg = fmt.Sprintf("Cannot get all Feeds: %s",
			err.Error())
		r.log.Printf("[ERROR] %s\n", msg)
		r.msgQueue <- msg
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
			var msg = fmt.Sprintf("Failed to refresh Feed %s: %s",
				f.Name,
				err.Error())
			r.log.Printf("[ERROR] %s\n", msg)
			r.msgQueue <- msg
			continue
		}

		r.log.Printf("[TRACE] Feed %s: Process %d Items\n",
			f.Name,
			len(items))

		for _, i := range items {
			var ref *feed.Item
			r.log.Printf("[TRACE] Process Item %s (%s)\n",
				i.Title,
				i.URL)
			if ref, err = r.db.ItemGetByURL(i.URL); err != nil {
				var msg = fmt.Sprintf("Cannot check if Item %s is in databse: %s",
					i.URL,
					err.Error())
				r.log.Printf("[ERROR] %s\n", msg)
				r.msgQueue <- msg
				return err
			} else if ref != nil {
				continue
			}

			if err = r.db.ItemAdd(&i); err != nil {
				var msg = fmt.Sprintf("Cannot save Item %q to database: %s",
					i.Title,
					err.Error())
				r.log.Printf("[ERROR] %s\n", msg)
				r.msgQueue <- msg
				return err
			}
		}

		if err = r.db.FeedSetTimestamp(&f, time.Now()); err != nil {
			var msg = fmt.Sprintf("Cannot update timestamp on Feed %s: %s",
				f.Name,
				err.Error())
			r.log.Printf("[ERROR] %s\n", msg)
			r.msgQueue <- msg
			continue
		}
	}

	return nil
} // func (r *Reader) refresh() error
