// /home/krylon/go/src/ticker/download/download.go
// -*- mode: go; coding: utf-8; -*-
// Created on 23. 06. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-07-09 17:32:23 krylon>

// Package download downloads and archives web pages.
package download

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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"github.com/blicero/ticker/blacklist"
	"github.com/blicero/ticker/common"
	"github.com/blicero/ticker/feed"
	"github.com/blicero/ticker/logdomain"
	"time"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

const wkInterval = time.Millisecond * 2500

var mimePat = regexp.MustCompile(`^([^/]+)/(\w+)`)

// Agent is the nexus of download activity.
type Agent struct {
	log       *log.Logger
	lock      sync.RWMutex
	active    bool
	PageQ     chan *feed.Item
	workerCnt int
}

// NewAgent creates an Agent with the given number of workers.
// The returned Agent is inactive initially.
func NewAgent(cnt int) (*Agent, error) {
	var (
		err error
		ag  *Agent
	)

	ag = &Agent{
		PageQ:     make(chan *feed.Item, cnt),
		workerCnt: cnt,
	}

	if ag.log, err = common.GetLogger(logdomain.Download); err != nil {
		return nil, err
	}

	return ag, nil
} // func NewAgent(cnt int) (*Agent, error)

// Start creates the worker goroutines and sets the Agent to active.
func (ag *Agent) Start() {
	ag.lock.Lock()
	defer ag.lock.Unlock()

	for i := 0; i < ag.workerCnt; i++ {
		go ag.worker(i)
	}

	ag.active = true
} // func (ag *Agent) Start()

// Stop sets the Agent to inactive.
func (ag *Agent) Stop() {
	ag.lock.Lock()
	ag.active = false
	ag.lock.Unlock()
} // func (ag *Agent) Stop()

// IsActive returns the status of the Agent.
func (ag *Agent) IsActive() bool {
	ag.lock.RLock()
	var status = ag.active
	ag.lock.RUnlock()
	return status
} // func (ag *Agent) IsActive() bool

func (ag *Agent) worker(idx int) {
	defer func() {
		if x := recover(); x != nil {
			var buf [2048]byte
			var cnt = runtime.Stack(buf[:], false)
			ag.log.Printf("[CRITICAL] Panic in Agent.worker: %s\n%s\n\n",
				x,
				string(buf[:cnt]))
		}
	}()

	var (
		bl     = blacklist.DefaultList()
		ticker = time.NewTicker(wkInterval)
	)

	defer ticker.Stop()

	for ag.IsActive() {
		select {
		case <-ticker.C:
			continue
		case item := <-ag.PageQ:
			ag.processPage(item, bl)
		}
	}
} // func (ag *Agent) worker(idx int)

func (ag *Agent) processPage(i *feed.Item, bl blacklist.Blacklist) {
	var (
		err     error
		resp    *http.Response
		pageDir string
	)

	// First we determine the local path:
	pageDir = filepath.Join(common.ArchiveDir, strconv.FormatInt(i.ID, 10))

	if err = os.Mkdir(pageDir, 0755); err != nil && !os.IsExist(err) {
		ag.log.Printf("[ERROR] Cannot create archive dir for Item %d (%s): %s\n",
			i.ID,
			i.Title,
			err.Error())
		return
	} else if resp, err = http.Get(i.URL); err != nil {
		ag.log.Printf("[ERROR] Error fetching Item %d (%s) from %q: %s\n",
			i.ID,
			i.Title,
			i.URL,
			err.Error())
		os.RemoveAll(pageDir) // nolint: errcheck
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		ag.log.Printf("[ERROR] Error fetching Item %d (%s) (%q): %s\n",
			i.ID,
			i.Title,
			i.URL,
			resp.Status)
		os.RemoveAll(pageDir) // nolint: errcheck
		return
	} else if !strings.HasPrefix(resp.Header["Content-Type"][0], "text/html") {
		ag.log.Printf("[ERROR] Unexpected content type for Item %d (%s) (%q): %s\n",
			i.ID,
			i.Title,
			i.URL,
			resp.Header["Content-Type"][0])
		os.RemoveAll(pageDir) // nolint: errcheck
		return
	}

	// I should feed the response body directly to the HTML parser,
	// storing it on disk has no advantages.

	var (
		fh       *os.File
		pageFile = filepath.Join(pageDir, "index.html")
	)

	if fh, err = os.Create(pageFile); err != nil {
		ag.log.Printf("[ERROR] Cannot create file to save Item %d (%s) at %s: %s\n",
			i.ID,
			i.Title,
			pageFile,
			err.Error())
		os.RemoveAll(pageDir) // nolint: errcheck
		return
	}

	defer fh.Close() // nolint: errcheck

	if _, err = io.Copy(fh, resp.Body); err != nil {
		ag.log.Printf("[ERROR] Failed to save Item %d (%s) (%q) to %s: %s\n",
			i.ID,
			i.Title,
			i.URL,
			pageFile,
			err.Error())
		os.RemoveAll(pageDir) // nolint: errcheck
		return
	} else if _, err = fh.Seek(0, 0); err != nil {
		ag.log.Printf("[ERROR] Cannot rewing file %s: %s\n",
			pageFile,
			err.Error())
		os.RemoveAll(pageDir) // nolint: errcheck
		return
	}

	var (
		doc *html.Node
		lnk *url.URL
	)

	if lnk, err = url.Parse(i.URL); err != nil {
		ag.log.Printf("[ERROR] Cannot parse Item URL %q: %s\n",
			i.URL,
			err.Error())
		os.RemoveAll(pageDir) // nolint: errcheck
	} else if doc, err = html.Parse(fh); err != nil {
		ag.log.Printf("[ERROR] Cannot parse response from %q: %s\n",
			i.URL,
			err.Error())
		os.RemoveAll(pageDir) // nolint: errcheck
		return
	}

	for _, node := range dom.GetElementsByTagName(doc, "img") {
		var (
			uri                 *url.URL
			localpath, basename string
			href                = dom.GetAttribute(node, "src")
		)

		ag.log.Printf("[DEBUG] Process image node: %s\n",
			node.Data)

		if uri, err = url.Parse(href); err != nil {
			ag.log.Printf("[ERROR] Cannot parse Image URL %q: %s\n",
				href,
				err.Error())
			continue
		} else if !uri.IsAbs() {
			var oldURI = uri
			uri = lnk.ResolveReference(uri)
			ag.log.Printf("[TRACE] Resolve relative URI %s to %s\n",
				oldURI,
				uri)
		}

		if bl.Match(uri.String()) {
			node.Parent.RemoveChild(node)
			continue
		} else if localpath, err = ag.fetchImage(uri, pageDir); err != nil {
			ag.log.Printf("[ERROR] Cannot fetch image %s: %s\n",
				uri,
				err.Error())
			continue
		}

		basename = path.Base(localpath)
		href = fmt.Sprintf("/archive/%d/%s",
			i.ID,
			basename)

		dom.SetAttribute(node, "src", href)
	}

	// for _, node := range dom.GetElementsByTagName(doc, "script") {
	for _, node := range getAssets(doc) {
		var (
			src, localpath, tagName string
			uri                     *url.URL
		)

		switch tagName = dom.TagName(node); tagName {
		case "script":
			src = dom.GetAttribute(node, "src")
		case "link":
			src = dom.GetAttribute(node, "href")
		case "a":
			src = dom.GetAttribute(node, "href")
		case "video", "audio", "iframe":
			node.Parent.RemoveChild(node)
			continue
		default:
			ag.log.Printf("[ERROR] Don't know how to handle Node %s\n",
				tagName)
			continue
		}

		ag.log.Printf("[DEBUG] Process asset %q\n",
			src)

		if src == "" {
			continue
		} else if uri, err = url.Parse(src); err != nil {
			ag.log.Printf("[ERROR] Cannot parse URL %q: %s\n",
				src,
				err.Error())
			switch tagName {
			case "script":
				//dom.SetAttribute(node, "src", "")
				fallthrough
			case "link":
				node.Parent.RemoveChild(node)
			default:
				ag.log.Printf("[CANTHAPPEN] STILL Don't know how to handle %s\n",
					tagName)
			}

			continue
		} else if !uri.IsAbs() {
			var old = uri
			uri = lnk.ResolveReference(uri)
			ag.log.Printf("[TRACE] Resolve relative URI %s to %s\n",
				old,
				uri)
		}

		if bl.Match(uri.String()) {
			node.Parent.RemoveChild(node)
			ag.log.Printf("[DEBUG] URI %q is blacklisted.\n",
				uri)
			continue
		} else if localpath, err = ag.fetchScript(uri, pageDir); err != nil {
			ag.log.Printf("[ERROR] Cannot fetch %q: %s\n",
				uri,
				err.Error())
			node.Parent.RemoveChild(node)
			continue
		} else {
			var basename = path.Base(localpath)
			var href = fmt.Sprintf("/archive/%d/%s",
				i.ID,
				basename)
			dom.SetAttribute(node, "src", href)
		}
	}

	for _, node := range dom.GetElementsByTagName(doc, "iframe") {
		node.Parent.RemoveChild(node)
	}

	var buf bytes.Buffer

	if err = html.Render(&buf, doc); err != nil {
		ag.log.Printf("[ERROR] Cannot render DOM tree back to HTML: %s\n",
			err.Error())
		os.RemoveAll(pageDir) // nolint: errcheck
	} else if _, err = fh.Seek(0, 0); err != nil {
		ag.log.Printf("[ERROR] Cannot rewind filehandle %s: %s\n",
			pageFile,
			err.Error())
		os.RemoveAll(pageDir) // nolint: errcheck
	} else if err = fh.Truncate(0); err != nil {
		ag.log.Printf("[ERROR] Cannot truncate file %s: %s\n",
			pageFile,
			err.Error())
		os.RemoveAll(pageDir) // nolint: errcheck
	} else if _, err = fh.Write(buf.Bytes()); err != nil {
		ag.log.Printf("[ERROR] Cannot write to file %s: %s\n",
			pageFile,
			err.Error())
		os.RemoveAll(pageDir) // nolint: errcheck
	}
} // func (ag *Agent) processPage(i *feed.Item)

func (ag *Agent) fetchImage(addr *url.URL, folder string) (string, error) {
	var (
		err                      error
		filename, aStr, mimetype string
		resp                     *http.Response
		fh                       *os.File
	)

	aStr = addr.String()

	if resp, err = http.Get(aStr); err != nil {
		ag.log.Printf("[ERROR] Failed to get %q: %s\n",
			aStr,
			err.Error())
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		ag.log.Printf("[ERROR] Failed to fetch %q: %s\n",
			aStr,
			resp.Status)
		return "", fmt.Errorf("Cannot download %q: %s",
			aStr,
			resp.Status)
	}

	mimetype = resp.Header.Get("Content-Type")

	if mimetype[:6] != "image/" {
		err = fmt.Errorf("Unexpected content type for %q: %s",
			addr,
			mimetype)
		ag.log.Printf("[ERROR] %s\n", err.Error())
		return "", err
	}
	var base = path.Base(addr.EscapedPath())
	filename = filepath.Join(
		folder,
		base)

	ag.log.Printf("[DEBUG] Save %q to %s\n",
		addr,
		filename)

	if fh, err = os.Create(filename); err != nil {
		ag.log.Printf("[ERROR] Cannot create file %s: %s\n",
			filename,
			err.Error())
		return "", err
	}

	defer fh.Close()

	if _, err = io.Copy(fh, resp.Body); err != nil {
		ag.log.Printf("[ERROR] Failed to save HTTP response for %q to %s: %s\n",
			aStr,
			filename,
			err.Error())
		os.Remove(filename) // nolint: errcheck
		return "", err
	}

	return filename, nil
} // func (ag *Agent) fetchImage(addr *url.URL, folder string) (string, error)

func (ag *Agent) fetchScript(href *url.URL, folder string) (string, error) {
	var (
		err                       error
		localpath, astr, filename string
		fh                        *os.File
		resp                      *http.Response
	)

	astr = href.String()
	filename = path.Base(href.EscapedPath())
	localpath = filepath.Join(folder, filename)

	if fh, err = os.Create(localpath); err != nil {
		ag.log.Printf("[ERROR] Cannot create local file %s: %s\n",
			localpath,
			err.Error())
		return "", err
	}

	defer fh.Close() // nolint: errcheck

	if resp, err = http.Get(astr); err != nil {
		ag.log.Printf("[ERROR] Cannot retrieve %q: %s\n",
			astr,
			err.Error())
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Failed to retrieve %q: %s",
			astr,
			resp.Status)
		ag.log.Printf("[ERROR] %s\n",
			err.Error())
		return "", err
	}

	var match = mimePat.FindStringSubmatch(resp.Header.Get("Content-Type"))

	if match == nil {
		err = fmt.Errorf("Cannot parse MIME type for %q: %q",
			astr,
			resp.Header.Get("Content-Type"))
		ag.log.Printf("[ERROR] %s\n",
			err.Error())
		return "", err
	}

	switch strings.ToLower(match[2]) {
	case "javascript", "css", "json", "x-icon", "png":
		// Proceed
	case "html":
		os.Remove(localpath) // nolint: errcheck
		return "", nil
	default:
		err = fmt.Errorf("Unexpected content type for %q: %q",
			astr,
			resp.Header.Get("Content-Type"))
		ag.log.Printf("[ERROR] %s\n",
			err.Error())
		return "", err
	}

	if _, err = io.Copy(fh, resp.Body); err != nil {
		ag.log.Printf("[ERROR] Cannot save %q to %s: %s\n",
			astr,
			localpath,
			err.Error())
		return "", err
	}

	return localpath, nil
} // func (ag *Agent) fetchScript(href, folder string) (string, error)

func getAssets(doc *html.Node) []*html.Node {
	return dom.GetAllNodesWithTag(doc, "a", "script", "iframe", "link", "video", "audio")
	// var nodes = dom.GetElementsByTagName(doc, "script")

	// for _, n := range dom.GetElementsByTagName(doc, "link") {
	// 	if dom.GetAttribute(n, "rel") == "stylesheet" {
	// 		nodes = append(nodes, n)
	// 	}
	// }

	// return nodes
} // func getAssets(doc *html.Node) []*html.Node
