// /home/krylon/go/src/ticker/prefetch/prefetch.go
// -*- mode: go; coding: utf-8; -*-
// Created on 31. 05. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-01 01:07:41 krylon>

// Package prefetch processes items received via RSS/Atom feeds
// and checks if they contain image links.
// If they do, it loads those images, saves them locally and adjusts
// the item to reference the local image.
// We may do this for other resources, too.
package prefetch

import (
	"log"
	"sync"
	"ticker/common"
	"ticker/database"
	"ticker/feed"
	"ticker/logdomain"
	"time"
)

const (
	delay     = time.Second * 10
	batchSize = 5
)

type processedItem struct {
	item feed.Item
	body string
}

// Prefetcher takes care of processing news items to prefetch and locally store images.
// Maybe other stuff, too.
type Prefetcher struct {
	log     *log.Logger
	lock    sync.RWMutex
	procQ   chan feed.Item
	resQ    chan processedItem
	cnt     int
	running bool
}

// Create creates a new instance of the Prefetcher, prepared to use up to cnt
// concurrent goroutines for fetching objects.
func Create(cnt int) (*Prefetcher, error) {
	var (
		err error
		pre *Prefetcher
	)

	pre = &Prefetcher{cnt: cnt}

	if pre.log, err = common.GetLogger(logdomain.Prefetch); err != nil {
		return nil, err
	} /* else if pre.db, err = database.Open(common.DbPath); err != nil {
		pre.log.Printf("[ERROR] Cannot open Database %s: %s\n",
			common.DbPath,
			err.Error())
		return nil, err
	} */

	pre.procQ = make(chan feed.Item, cnt)
	pre.resQ = make(chan processedItem, cnt)
	return pre, nil
} // func Create() (*Prefetcher, error)

// IsRunning returns true if the Prefetcher is active.
func (p *Prefetcher) IsRunning() bool {
	p.lock.RLock()
	var b = p.running
	p.lock.RUnlock()
	return b
} // func (p *Prefetcher) IsRunning() bool

// Start sets the Prefetcher on its path.
func (p *Prefetcher) Start() error {
	var (
		err          error
		dbSrc, dbDst *database.Database
	)

	p.lock.Lock()
	defer p.lock.Unlock()

	p.running = true

	if dbSrc, err = database.Open(common.DbPath); err != nil {
		p.log.Printf("[ERROR] Cannot open database for feeder loop: %s\n",
			err.Error())
		return err
	} else if dbDst, err = database.Open(common.DbPath); err != nil {
		dbSrc.Close() // nolint: errcheck
		p.log.Printf("[ERROR] Cannot open database for receiver loop: %s\n",
			err.Error())
		return err
	}

	go p.feeder(dbSrc)
	go p.receiver(dbDst)
	// dbSrc = nil
	// dbDst = nil

	for i := 0; i < p.cnt; i++ {
		go p.worker()
	}

	return nil
} // func (p *Prefetcher) Start() error

func (p *Prefetcher) feeder(db *database.Database) {
	defer db.Close() // nolint: errcheck

	for p.IsRunning() {
		var (
			err   error
			items []feed.Item
		)

		if items, err = db.ItemGetPrefetch(batchSize); err != nil {
			p.log.Printf("[ERROR] Cannot get unprocessed items from database: %s\n",
				err.Error())
		}

		for _, i := range items {
			p.procQ <- i
		}

		time.Sleep(delay)
	}
} // func (p *Prefetcher) feeder(db *database.Database)

func (p *Prefetcher) receiver(db *database.Database) {
	defer db.Close()

	var ticker = time.NewTicker(delay)
	defer ticker.Stop()

	for p.IsRunning() {
		var (
			err error
			i   processedItem
		)
		select {
		case <-ticker.C:
			continue
		case i = <-p.resQ:
			// ... do something about it:
			if err = db.ItemPrefetchSet(&i.item, i.body); err != nil {
				p.log.Printf("[ERROR] Cannot update prefetched Item %d (%q): %s\n",
					i.item.ID,
					i.item.Title,
					err.Error())
			}
		}
	}
} // func (p *Prefetcher) receiver(db *database.Database)

func (p *Prefetcher) worker() {
	for p.IsRunning() {
	}
} // func (p *Prefetcher) worker()
