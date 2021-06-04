// /home/krylon/go/src/ticker/prefetch/01_prefetch_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 06. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-04 16:09:58 krylon>

package prefetch

import (
	"net/url"
	"strings"
	"testing"
	"ticker/common"
	"ticker/database"
	"ticker/feed"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

var (
	pre *Prefetcher
	tdb *database.Database
)

func TestSanitize(t *testing.T) {
	var (
		err   error
		items []feed.Item
	)

	if tdb, err = database.Open(common.DbPath); err != nil {
		t.Fatalf("Error opening database: %s", err.Error())
	} else if pre, err = Create(1); err != nil {
		tdb.Close() // nolint: errcheck
		tdb = nil
		t.Fatalf("Error creating Prefetcher: %s", err.Error())
	} else if items, err = tdb.ItemGetPrefetch(batchSize); err != nil {
		t.Fatalf("Cannot fetch Items: %s", err.Error())
	} else if len(items) != batchSize {
		t.Errorf("Unexpected number of items: %d (expected %d)",
			len(items),
			batchSize)
	}

	for _, i := range items {
		var body string
		if body, err = pre.sanitize(&i); err != nil {
			t.Errorf("Error sanitizing Item %d (%s): %s",
				i.ID,
				i.Title,
				err.Error())
		}

		t.Logf("Sanitized body of item %d: %s",
			i.ID,
			body)

		var (
			doc *html.Node
			rdr *strings.Reader
		)

		rdr = strings.NewReader(body)

		if doc, err = html.Parse(rdr); err != nil {
			t.Errorf("Cannot parse back body of item: %s\n%s",
				err.Error(),
				body)
			continue
		}

		for _, node := range dom.GetElementsByTagName(doc, "img") {
			var uri *url.URL
			var href = dom.GetAttribute(node, "src")

			if uri, err = url.Parse(href); err != nil {
				t.Errorf("Cannot parse img source %q: %s",
					href,
					err.Error())
			} else if uri.IsAbs() {
				t.Errorf("URL should not be absolute: %q", href)
			}
		}

		for _, node := range dom.GetElementsByTagName(doc, "script") {
			t.Errorf("<script> should have been removed: %s", node.Data)
		}
	}
} // func TestSanitize(t *testing.T)
