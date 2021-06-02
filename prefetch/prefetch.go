// /home/krylon/go/src/ticker/prefetch/prefetch.go
// -*- mode: go; coding: utf-8; -*-
// Created on 31. 05. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-02 10:49:15 krylon>

// Package prefetch processes items received via RSS/Atom feeds
// and checks if they contain image links.
// If they do, it loads those images, saves them locally and adjusts
// the item to reference the local image.
// We may do this for other resources, too.
package prefetch

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"ticker/common"
	"ticker/database"
	"ticker/feed"
	"ticker/logdomain"
	"time"

	"github.com/go-shiori/dom"

	"golang.org/x/net/html"
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
	var ticker = time.NewTicker(delay)
	defer ticker.Stop()

	for p.IsRunning() {
		var (
			err  error
			body string
		)

		select {
		case <-ticker.C:
			continue
		case i := <-p.procQ:
			if body, err = p.sanitize(&i); err != nil {
				p.log.Printf("[ERROR] Cannot process item %d (%q): %s\n",
					i.ID,
					i.Title,
					err.Error())
			} else {
				p.resQ <- processedItem{item: i, body: body}
			}
		}
	}
} // func (p *Prefetcher) worker()

func (p *Prefetcher) sanitize(i *feed.Item) (string, error) {
	var (
		err error
		rdr *strings.Reader
		doc *html.Node
		lnk *url.URL
	)

	if lnk, err = url.Parse(i.URL); err != nil {
		p.log.Printf("[ERROR] Cannot parse Item URL %q: %s\n",
			i.URL,
			err.Error())
		return "", err
	}

	p.log.Printf("[TRACE] Sanitize Item %d (%s)\n",
		i.ID,
		i.Title)

	rdr = strings.NewReader(i.Description)

	if doc, err = html.Parse(rdr); err != nil {
		return "", err
	}

	for _, node := range dom.GetElementsByTagName(doc, "img") {
		var (
			uri                 *url.URL
			localpath, basename string
			href                = dom.GetAttribute(node, "src")
		)

		if uri, err = url.Parse(href); err != nil {
			p.log.Printf("[ERROR] Cannot process URL %q: %s\n",
				href,
				err.Error())
			continue
		} else if !uri.IsAbs() {
			uri = lnk.ResolveReference(uri)
		}

		if localpath, err = p.fetchImage(uri.String()); err != nil {
			p.log.Printf("[ERROR] Error fetching image %q: %s\n",
				uri,
				err.Error())
			return "", err
		}

		basename = path.Base(localpath)

		href = "/cache/" + basename

		dom.SetAttribute(node, "src", href)
	}

	var buf bytes.Buffer

	if err = html.Render(&buf, doc); err != nil {
		p.log.Printf("[ERROR] Cannot render DOM tree back to HTML: %s\n",
			err.Error())
	}

	return buf.String(), nil
} // func (p *Prefetcher) sanitize(i *feed.Item) (string, error)

// Chances are pretty low, I suppose, but we could make a little effort to avoid
// duplicates.
// ... Not today, though! (2021-06-01)

func (p *Prefetcher) fetchImage(href string) (string, error) {
	var (
		err                      error
		resp                     *http.Response
		localPath, cksum, suffix string
		fh                       *os.File
	)

	if resp, err = http.Get(href); err != nil {
		p.log.Printf("[ERROR] Failed to fetch %q: %s\n",
			href,
			err.Error())
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Failed to fetch %q: %s",
			href,
			resp.Status)
		p.log.Printf("[ERROR] %s\n", err.Error())
		return "", err
	} else if cksum, err = common.GetChecksum([]byte(href)); err != nil {
		p.log.Printf("[ERROR] Cannot compute checksum of URL %q: %s\n",
			href,
			err.Error())
		return "", err
	} else if suffix, err = getFileSuffix(resp); err != nil {
		p.log.Printf("[ERROR] %s\n", err.Error())
		return "", err
	}

	localPath = filepath.Join(
		common.CacheDir,
		fmt.Sprintf("%s.%s",
			cksum,
			suffix))

	p.log.Printf("[TRACE] Save %q to %s\n",
		href,
		localPath)

	if fh, err = os.Create(localPath); err != nil {
		p.log.Printf("[ERROR] Cannot create local file %s: %s\n",
			localPath, err.Error())
		return "", err
	}

	defer fh.Close() // nolint: errcheck

	if _, err = io.Copy(fh, resp.Body); err != nil {
		p.log.Printf("[ERROR] Failed to save HTTP response for %q to %s: %s\n",
			href,
			localPath,
			err.Error())
		os.Remove(localPath) // nolint: errcheck
		return "", err
	}

	return localPath, nil
} // func (p *Prefetcher) fetchImage(href string) (string, error)

var suffixPattern = regexp.MustCompile("(?i)^image/([a-z]+)$")

func getFileSuffix(resp *http.Response) (string, error) {
	var (
		mime  string
		match []string
	)

	mime = resp.Header["Content-Type"][0]

	if match = suffixPattern.FindStringSubmatch(mime); match == nil {
		return "", fmt.Errorf("Could not parse MIME type %q", mime)
	}

	return match[1], nil
} // func getFileSuffix(resp *http.response) string