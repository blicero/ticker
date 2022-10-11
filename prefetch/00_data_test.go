// /home/krylon/go/src/ticker/prefetch/00_data_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 06. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-04 15:04:06 krylon>

package prefetch

import (
	"errors"
	"fmt"
	"math/rand"
	"github.com/blicero/ticker/common"
	"github.com/blicero/ticker/database"
	"github.com/blicero/ticker/feed"
	"time"
)

const (
	itemCnt     = 100
	period      = 5400 // 5400 seconds = 90 minutes
	itemTmplImg = `
Bla Bla Bla
<img src="/images/img%03d.jpg" />
Bla Bla Bla
`
	itemTmplScript = `
Bla Bla Bla<br />
Bla Bla Bla<br />
Bla <b>Bla</b> Bla<br />
<script src="http://www.example.com/fartscroll.js"></script>
Bla Bla Bla
`
)

var epoch = time.Now().Add(time.Second * -period)

func randTime() time.Time {
	return epoch.Add(time.Second * time.Duration(rand.Intn(period)))
} // func randTime() time.Time

var myFeed = feed.Feed{
	Name:       "Test Feed",
	Homepage:   "http://www.example.com/",
	Interval:   time.Second * 300,
	LastUpdate: time.Now().Add(time.Second * -180),
	Active:     true,
}

var items []feed.Item

func prepareItems() error {
	var (
		err    error
		db     *database.Database
		status bool
	)

	if srv == nil || srv.URL == "" {
		return errors.New("mock Server is not ready")
	} else if db, err = database.Open(common.DbPath); err != nil {
		return err
	}

	defer db.Close() // nolint: errcheck

	if err = db.Begin(); err != nil {
		return err
	}

	defer func() {
		if status {
			db.Commit() // nolint: errcheck
		} else {
			db.Rollback() // nolint: errcheck
		}
	}()

	if err = db.FeedAdd(&myFeed); err != nil {
		return err
	}

	items = make([]feed.Item, 0, itemCnt)

	var baseURL = srv.URL

	for idx := 0; idx < itemCnt; idx++ {
		var i = feed.Item{
			FeedID:      myFeed.ID,
			URL:         fmt.Sprintf("%s/item%04d", baseURL, idx),
			Title:       fmt.Sprintf("Breaking News: Something happened %d times", idx+1),
			Description: fmt.Sprintf(itemTmplImg, idx),
			Timestamp:   randTime(),
		}

		if idx&1 == 1 {
			i.Description += itemTmplScript
		}

		if err = db.ItemAdd(&i); err != nil {
			return err
		}
	}

	status = true
	return nil
} // func prepareItems() error
