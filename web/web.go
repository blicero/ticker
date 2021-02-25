// /home/krylon/go/src/ticker/web/web.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-25 02:55:04 krylon>

package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"text/template"
	"ticker/classifier"
	"ticker/common"
	"ticker/database"
	"ticker/feed"
	"ticker/logdomain"
	"ticker/tag"
	"time"

	"github.com/blicero/krylib"

	"github.com/CyrusF/go-bayesian"
	"github.com/gorilla/mux"
	"github.com/hashicorp/logutils"
)

//go:generate go run ./build_templates.go

const (
	defaultPoolSize = 4
	recentCnt       = 20
)

// type message struct {
// 	Timestamp time.Time
// 	Message   string
// 	Level     string
// }

// Server implements the web interface
type Server struct {
	Addr      string
	web       http.Server
	log       *log.Logger
	msgBuf    *krylib.MessageBuffer
	router    *mux.Router
	tmpl      *template.Template
	mimeTypes map[string]string
	pool      *database.Pool
	// rev       *classifier.Classifier
}

// Create creates a new Server instance.
func Create(addr string, keepAlive bool) (*Server, error) {
	var (
		err error
		msg string
		srv = &Server{
			Addr:   addr,
			msgBuf: krylib.CreateMessageBuffer(),
			mimeTypes: map[string]string{
				".css": "text/css",
				".js":  "text/javascript",
				".png": "image/png",
			},
		}
	)

	if srv.log, err = common.GetLogger(logdomain.Web); err != nil {
		return nil, err
	} else if srv.pool, err = database.NewPool(defaultPoolSize); err != nil {
		srv.log.Printf("[ERROR] Cannot create DB pool: %s\n",
			err.Error())
		return nil, err
	} /* else if srv.rev, err = classifier.New(); err != nil {
		srv.log.Printf("[ERROR] Cannot create Classifier: %s\n",
			err.Error())
		srv.pool.Close()
		return nil, err
	} */

	srv.tmpl = template.New("").Funcs(funcmap)
	for name, body := range htmlData.Templates {
		if srv.tmpl, err = srv.tmpl.Parse(body); err != nil {
			msg = fmt.Sprintf("Could not parse template %s: %s",
				name,
				err.Error())
			srv.log.Println("[CRITICAL] " + msg)
			return nil, errors.New(msg)
		} else if common.Debug {
			srv.log.Printf("[TRACE] Template \"%s\" was parsed successfully.\n",
				name)
		}
	}

	srv.router = mux.NewRouter()
	srv.web.Addr = addr
	srv.web.ErrorLog = srv.log
	srv.web.Handler = srv.router

	srv.router.HandleFunc("/static/{file}", srv.handleStaticFile)
	srv.router.HandleFunc("/{page:(?i)(?:index|main)?$}", srv.handleIndex)

	srv.router.HandleFunc("/feed/form", srv.handleFeedForm)
	srv.router.HandleFunc("/feed/subscribe", srv.handleFeedSubscribe)
	srv.router.HandleFunc("/feed/{id:(?:\\d+)$}", srv.handleFeedDetails)

	srv.router.HandleFunc("/items/{page:(?:\\d+|all)$}", srv.handleItems)

	srv.router.HandleFunc("/search", srv.handleSearch)

	srv.router.HandleFunc("/tag/all", srv.handleTagList)
	srv.router.HandleFunc("/tag/create", srv.handleTagCreate)
	srv.router.HandleFunc("/tag/{id:(?:\\d+)$}", srv.handleTagDetails)

	srv.router.HandleFunc("/ajax/beacon", srv.handleBeacon)
	srv.router.HandleFunc("/ajax/get_messages", srv.handleGetNewMessages)
	srv.router.HandleFunc("/ajax/rate_item", srv.handleRateItem)
	srv.router.HandleFunc("/ajax/unrate_item/{id:(?:\\d+)$}", srv.handleUnrateItem)
	srv.router.HandleFunc("/ajax/rebuild_fts", srv.handleRebuildFTS)

	if !common.Debug {
		srv.web.SetKeepAlivesEnabled(keepAlive)
	}

	return srv, nil
} // func Create(addr string, keepAlive bool) (*Server, error)

// ListenAndServe enters the HTTP server's main loop, i.e.
// this method must be called for the Web frontend to handle
// requests.
func (srv *Server) ListenAndServe() {
	var err error

	defer srv.log.Println("[INFO] Web server is shutting down")

	srv.log.Printf("[INFO] Web frontend is going online at %s\n", srv.Addr)
	http.Handle("/", srv.router)

	if err = srv.web.ListenAndServe(); err != nil {
		if err.Error() != "http: Server closed" {
			srv.log.Printf("[ERROR] ListenAndServe returned an error: %s\n",
				err.Error())
		} else {
			srv.log.Println("[INFO] HTTP Server has shut down.")
		}
	}
} // func (srv *Server) ListenAndServe()

// SendMessage send a message to the server's message queue
func (srv *Server) SendMessage(msg string) {
	srv.msgBuf.AddMessage(msg)

	if common.Debug {
		srv.log.Printf("[DEBUG] MessageBuffer holds %d messages\n",
			srv.msgBuf.Count())
	}
} // func (srv *Server) SendMessage(msg string)

// Close shuts down the server.
func (srv *Server) Close() error {
	var err error

	if err = srv.pool.Close(); err != nil {
		srv.log.Printf("[ERROR] Cannot close database pool: %s\n",
			err.Error())
		return err
	} else if err = srv.web.Close(); err != nil {
		srv.log.Printf("[ERROR] Cannot shutdown HTTP server: %s\n",
			err.Error())
		return err
	}

	return nil
} // func (srv *Server) Close() error

// nolint: unused
func (srv *Server) getMessages() []message {
	var m1 = srv.msgBuf.GetAllMessages()
	var m2 = make([]message, len(m1))

	for idx, m := range m1 {
		m2[idx] = message{
			Timestamp: m.Stamp,
			Message:   m.Msg,
			Level:     "DEBUG",
		}
	}

	return m2
} // func (srv *Server) getMessages() []krylib.Message

////////////////////////////////////////////////////////////////////////////////
//// URL handlers //////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

/////////////////////////////////////////
////////////// Web UI ///////////////////
/////////////////////////////////////////

func (srv *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle request for %s\n",
		r.URL.EscapedPath())

	const (
		tmplName  = "index"
		recentCnt = 20
	)

	var (
		err  error
		msg  string
		db   *database.Database
		tmpl *template.Template
		data = tmplDataIndex{
			tmplDataBase: tmplDataBase{
				Title: "Main",
				Debug: common.Debug,
				URL:   r.URL.String(),
			},
		}
	)

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Could not find template %q", tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if data.Feeds, err = db.FeedGetAll(); err != nil {
		msg = fmt.Sprintf("Cannot query all Feeds: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if data.Items, err = db.ItemGetRecent(recentCnt); err != nil {
		msg = fmt.Sprintf("Cannot query all Items: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	data.FeedMap = make(map[int64]feed.Feed, len(data.Feeds))
	for _, f := range data.Feeds {
		data.FeedMap[f.ID] = f
	}

	data.Messages = srv.getMessages()

	w.Header().Set("Cache-Control", "no-cache")
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		srv.SendMessage(msg)
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleIndex(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleFeedForm(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle %s from %s\n",
		r.URL,
		r.RemoteAddr)

	const (
		tmplName = "subscribe"
	)

	var (
		err  error
		tmpl *template.Template
		msg  string
		data = tmplDataIndex{
			tmplDataBase: tmplDataBase{
				Title: "Subscribe to Feed",
				Debug: common.Debug,
				URL:   r.URL.String(),
			},
		}
	)

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Could not find template %q", tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	data.Messages = srv.getMessages()

	w.Header().Set("Cache-Control", "no-cache")
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		srv.SendMessage(msg)
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleFeedForm(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleFeedSubscribe(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle %s from %s\n",
		r.URL,
		r.RemoteAddr)

	var (
		err       error
		msg, iStr string
		f         feed.Feed
		interval  int64
		db        *database.Database
	)

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if err = r.ParseForm(); err != nil {
		msg = fmt.Sprintf("Could not parse form data: %s", err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, "/index", http.StatusFound)
		return
	}

	f.Name = r.FormValue("name")
	f.URL = r.FormValue("url")
	f.Homepage = r.FormValue("homepage")
	iStr = r.FormValue("interval")

	if interval, err = strconv.ParseInt(iStr, 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse interval %q: %s",
			iStr,
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, "/index", http.StatusFound)
		return
	}

	f.Interval = time.Second * time.Duration(interval)

	if err = db.FeedAdd(&f); err != nil {
		msg = fmt.Sprintf("Cannot add Feed %s (%s): %s",
			f.Name,
			f.URL,
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, "/index", http.StatusFound)
		return
	}

	var dstURL = fmt.Sprintf("/feed/%d", f.ID)

	http.Redirect(w, r, dstURL, http.StatusFound)
} // func (srv *Server) handleFeedSubscribe(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleFeedDetails(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle %s from %s\n",
		r.URL,
		r.RemoteAddr)

	const tmplName = "feed_details"

	var (
		err        error
		msg, idStr string
		id         int64
		db         *database.Database
		tmpl       *template.Template
		data       = tmplDataFeedDetails{
			tmplDataBase: tmplDataBase{
				Debug: common.Debug,
				URL:   r.URL.String(),
			},
		}
	)

	vars := mux.Vars(r)

	idStr = vars["id"]

	if id, err = strconv.ParseInt(idStr, 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse ID %q: %s",
			idStr,
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, "/index", http.StatusFound)
		return
	}

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Could not find template %q", tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if data.Feed, err = db.FeedGetByID(id); err != nil {
		msg = fmt.Sprintf("Cannot query all Feeds: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if data.Items, err = db.ItemGetByFeed(id, recentCnt); err != nil {
		msg = fmt.Sprintf("Cannot query all Items: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	data.FeedMap = map[int64]feed.Feed{id: *data.Feed}

	data.Title = data.Feed.Name
	data.Messages = srv.getMessages()

	w.Header().Set("Cache-Control", "no-cache")
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		srv.SendMessage(msg)
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleFeedDetails(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleItems(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle request for %s\n",
		r.URL.EscapedPath())

	const (
		tmplName = "all_items"
		itemCnt  = 50
	)

	var (
		err                 error
		msg, pageSpec       string
		pageNo, cnt, offset int64
		db                  *database.Database
		tmpl                *template.Template
		rev                 *classifier.Classifier
		data                = tmplDataItems{
			tmplDataBase: tmplDataBase{
				Title: "Main",
				Debug: common.Debug,
				URL:   r.URL.String(),
			},
		}
	)

	vars := mux.Vars(r)
	pageSpec = vars["page"]

	if pageSpec == "all" {
		pageNo = -1
	} else if pageNo, err = strconv.ParseInt(pageSpec, 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse page number %q: %s",
			pageSpec,
			err.Error())
		srv.log.Printf("[CANTHAPPEN] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	if pageNo != -1 {
		cnt = itemCnt
		offset = itemCnt * pageNo
	} else {
		cnt = -1
	}

	if pageNo > 0 {
		data.Prev = strconv.FormatInt(pageNo-1, 10)
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if rev, err = classifier.New(); err != nil {
		msg = fmt.Sprintf("Cannot create Classifier: %s",
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, "/index", http.StatusFound)
		return
	} else if data.Items, err = db.ItemGetAll(cnt, offset); err != nil {
		msg = fmt.Sprintf("Cannot load Items (%d / offset %d) from database: %s",
			itemCnt,
			offset,
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, "/index", http.StatusFound)
		return
	} else if pageNo != -1 && len(data.Items) == itemCnt {
		data.Next = strconv.FormatInt(pageNo+1, 10)
	}

	if err = rev.Train(); err != nil {
		msg = fmt.Sprintf("Cannot train Classifier: %s",
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, "/index", http.StatusFound)
		return
	}

	var feeds []feed.Feed

	if feeds, err = db.FeedGetAll(); err != nil {
		msg = fmt.Sprintf("Cannot get all Feeds: %s",
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, "/index", http.StatusFound)
		return
	}

	data.FeedMap = make(map[int64]feed.Feed, len(feeds))

	for _, f := range feeds {
		data.FeedMap[f.ID] = f
	}

	for idx, item := range data.Items {
		var (
			certain bool
			score   map[bayesian.Class]float64
			class   bayesian.Class
		)

		if !math.IsNaN(item.Rating) {
			continue
		}

		score, class, certain = rev.Classify(&item)

		if certain {
			switch class {
			case classifier.Good:
				data.Items[idx].Rating = score[class]
			case classifier.Bad:
				data.Items[idx].Rating = -score[class]
			default:
				srv.log.Printf("[CANTHAPPEN] Unexpected classification for news Item %d (%q): %s\n",
					item.ID,
					item.Title,
					class)
				continue
			}

			srv.log.Printf("[TRACE] Rate Item %d (%q) - %f\n",
				item.ID,
				item.Title,
				data.Items[idx].Rating)
		}
	}

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Could not find template %q", tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	data.Messages = srv.getMessages()

	w.Header().Set("Cache-Control", "no-cache")
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		srv.SendMessage(msg)
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleItems(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle request for %s\n",
		r.URL.EscapedPath())

	const (
		tmplName = "all_items"
		itemCnt  = 50
	)

	var (
		err       error
		msg, qstr string
		db        *database.Database
		tmpl      *template.Template
		data      = tmplDataItems{
			tmplDataBase: tmplDataBase{
				Title: "Main",
				Debug: common.Debug,
				URL:   r.URL.String(),
			},
		}
	)

	if err = r.ParseForm(); err != nil {
		msg = fmt.Sprintf("Could not parse form data: %s", err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, "/index", http.StatusFound)
		return
	}

	qstr = r.FormValue("query")

	srv.log.Printf("[TRACE] Receive query for %q\n",
		qstr)

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Could not find template %q", tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	var feeds []feed.Feed

	if feeds, err = db.FeedGetAll(); err != nil {
		msg = fmt.Sprintf("Cannot get all Feeds: %s",
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, "/index", http.StatusFound)
		return
	}

	data.FeedMap = make(map[int64]feed.Feed, len(feeds))

	for _, f := range feeds {
		data.FeedMap[f.ID] = f
	}

	if data.Items, err = db.ItemGetFTS(qstr); err != nil {
		msg = fmt.Sprintf("Cannot search database for %q: %s",
			qstr,
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, "/index", http.StatusFound)
		return
	}

	data.Messages = srv.getMessages()

	w.Header().Set("Cache-Control", "no-cache")
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		srv.SendMessage(msg)
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleSearch(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleTagList(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle request for %s\n",
		r.URL.EscapedPath())

	const (
		tmplName = "tags"
		itemCnt  = 5
	)

	var (
		err  error
		msg  string
		tmpl *template.Template
		db   *database.Database
		data = tmplDataTags{
			tmplDataBase: tmplDataBase{
				Title: "All Tags",
				Debug: common.Debug,
				URL:   r.URL.String(),
			},
		}
	)

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if data.Tags, err = db.TagGetAll(); err != nil {
		msg = fmt.Sprintf("Cannot load list of all Tags: %s",
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	} else if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Cannot find Template %s",
			tmplName)
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	data.Messages = srv.getMessages()
	w.Header().Set("Cache-Control", "no-cache")
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		srv.SendMessage(msg)
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleTagList(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleTagCreate(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle request for %s\n",
		r.URL.EscapedPath())

	const (
		parentID = 0
	)

	var (
		err             error
		msg, name, desc string
		t               *tag.Tag
		db              *database.Database
	)

	if err = r.ParseForm(); err != nil {
		msg = fmt.Sprintf("Cannnot parse form data: %s",
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	name = r.FormValue("name")
	desc = r.FormValue("description")
	// TODO Parent!

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if t, err = db.TagCreate(name, desc, parentID); err != nil {
		msg = fmt.Sprintf("Cannot create Tag %q: %s",
			name,
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	var addr = fmt.Sprintf("/tag/%d", t.ID)
	http.Redirect(w, r, addr, http.StatusFound)
} // func (srv *Server) handleTagCreate(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleTagDetails(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle request for %s\n",
		r.URL.EscapedPath())

	const (
		tmplName = "tag_details"
		itemCnt  = 5
	)

	var (
		err        error
		msg, idStr string
		id         int64
		tmpl       *template.Template
		db         *database.Database
		data       = tmplDataTagDetails{
			tmplDataBase: tmplDataBase{
				Debug: common.Debug,
				URL:   r.URL.String(),
			},
		}
	)

	vars := mux.Vars(r)
	idStr = vars["id"]

	if id, err = strconv.ParseInt(idStr, 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse Tag ID %q: %s",
			idStr,
			err.Error())
		srv.log.Println("[CANTHAPPEN] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if data.Tag, err = db.TagGetByID(id); err != nil {
		msg = fmt.Sprintf("Cannot load Tag %d: %s",
			id,
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	} else if data.Items, err = db.ItemGetByTag(data.Tag); err != nil {
		msg = fmt.Sprintf("Cannot load Items tagged as %s: %s",
			data.Tag.Name,
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	} else if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Did not find template %s",
			tmplName)
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	data.Title = fmt.Sprintf("Details for Tag %s", data.Tag.Name)
	data.Messages = srv.getMessages()
	w.Header().Set("Cache-Control", "no-cache")
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		srv.SendMessage(msg)
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleTagDetails(w http.ResponseWriter, r *http.Request)

/////////////////////////////////////////
////////////// Other ////////////////////
/////////////////////////////////////////

func (srv *Server) handleStaticFile(w http.ResponseWriter, request *http.Request) {
	srv.log.Printf("[TRACE] Handle request for %s\n",
		request.URL.EscapedPath())

	// Since we controll what static files the server has available, we
	// can easily map MIME type to slice. Soon.

	vars := mux.Vars(request)
	filename := vars["file"]

	var mimeType string

	srv.log.Printf("[TRACE] Delivering static file %s to client\n", filename)

	var match []string

	if match = common.SuffixPattern.FindStringSubmatch(filename); match == nil {
		mimeType = "text/plain"
	} else if mime, ok := srv.mimeTypes[match[1]]; ok {
		mimeType = mime
	} else {
		srv.log.Printf("[ERROR] Did not find MIME type for %s\n", filename)
	}

	w.Header().Set("Content-Type", mimeType)

	if common.Debug {
		w.Header().Set("Cache-Control", "no-cache")
	} else {
		w.Header().Set("Cache-Control", "max-age=7200")
	}

	if body, ok := htmlData.Static[filename]; ok {
		w.WriteHeader(200)
		_, _ = w.Write(body) // nolint: gosec
	} else {
		msg := fmt.Sprintf("ERROR - cannot find file %s", filename)
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleStaticFile(w http.ResponseWriter, request *http.Request)

func (srv *Server) sendErrorMessage(w http.ResponseWriter, msg string) {
	html := `
<!DOCTYPE html>
<html>
  <head>
    <title>Internal Error</title>
  </head>
  <body>
    <h1>Internal Error</h1>
    <hr />
    We are sorry to inform you an internal application error has occured:<br />
    %s
    <p>
    Back to <a href="/index">Homepage</a>
    <hr />
    &copy; 2018 <a href="mailto:krylon@gmx.net">Benjamin Walkenhorst</a>
  </body>
</html>
`

	srv.log.Printf("[ERROR] %s\n", msg)

	output := fmt.Sprintf(html, msg)
	w.WriteHeader(500)
	_, _ = w.Write([]byte(output)) // nolint: gosec
} // func (srv *Server) sendErrorMessage(w http.ResponseWriter, msg string)

////////////////////////////////////////////////////////////////////////////////
//// Ajax handlers /////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

// const success = "Success"

func (srv *Server) handleBeacon(w http.ResponseWriter, r *http.Request) {
	// srv.log.Printf("[TRACE] Handle %s from %s\n",
	// 	r.URL,
	// 	r.RemoteAddr)
	var timestamp = time.Now().Format(common.TimestampFormat)
	const appName = common.AppName + " " + common.Version
	var jstr = fmt.Sprintf(`{ "Status": true, "Message": "%s", "Timestamp": "%s", "Hostname": "%s" }`,
		appName,
		timestamp,
		hostname())
	var response = []byte(jstr)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(200)
	w.Write(response) // nolint: errcheck,gosec
} // func (srv *Web) handleBeacon(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleGetNewMessages(w http.ResponseWriter, r *http.Request) {
	// srv.log.Printf("[TRACE] Handle %s from %s\n",
	// 	r.URL,
	// 	r.RemoteAddr)

	type msgItem struct {
		Time    string
		Level   logutils.LogLevel
		Message string
	}

	type resBody struct {
		Status   bool
		Message  string
		Messages []msgItem
	}

	var messages = srv.getMessages()
	var res = resBody{
		Status:   true,
		Messages: make([]msgItem, len(messages)),
	}

	for idx, i := range messages {
		res.Messages[idx] = msgItem{
			Time:    i.TimeString(),
			Level:   i.Level,
			Message: i.Message,
		}
	}

	var (
		err error
		msg string
		buf []byte
	)

	if buf, err = json.Marshal(&res); err != nil {
		msg = fmt.Sprintf("Error serializing response: %s",
			err.Error())
		srv.SendMessage(msg)
		res.Message = msg
		buf = errJSON(msg)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(200)
	if _, err = w.Write(buf); err != nil {
		msg = fmt.Sprintf("Failed to send result: %s",
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
	}
} // func (srv *Server) handleGetNewMessages(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleRateItem(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle %s from %s\n",
		r.URL,
		r.RemoteAddr)

	var (
		err                     error
		db                      *database.Database
		idStr, rStr, msg, reply string
		id                      int64
		item                    *feed.Item
		rating                  float64
	)

	if err = r.ParseForm(); err != nil {
		msg = fmt.Sprintf("Could not parse form data: %s", err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		return
	}

	idStr = r.PostFormValue("ID")
	rStr = r.PostFormValue("Rating")

	srv.log.Printf("[DEBUG] Rate Item %s - %s\n",
		idStr,
		rStr)

	if id, err = strconv.ParseInt(idStr, 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse ID %q: %s", idStr, err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		goto SEND_ERROR_MESSAGE
	} else if rating, err = strconv.ParseFloat(rStr, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse Rating %q: %s", rStr, err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		goto SEND_ERROR_MESSAGE
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if item, err = db.ItemGetByID(id); err != nil {
		msg = fmt.Sprintf("Cannot load Item by ID %d: %s",
			id,
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		goto SEND_ERROR_MESSAGE
	} else if item == nil {
		msg = fmt.Sprintf("No such Item: %d",
			id)
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		goto SEND_ERROR_MESSAGE
	} else if err = db.ItemRatingSet(item, rating); err != nil {
		msg = fmt.Sprintf("Cannot set Rating of Item %d to %.2f: %s",
			item.ID,
			rating,
			err.Error())
		srv.log.Println("[ERROR] " + msg)
		srv.SendMessage(msg)
		goto SEND_ERROR_MESSAGE
	}

	reply = fmt.Sprintf(`{ "Status": true, "ID": %d, "Rating": %f, "Message": "Success" }`,
		id,
		rating)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(reply)) // nolint: errcheck
	return

SEND_ERROR_MESSAGE:
	reply = fmt.Sprintf(`{ "Status": false, "ID": %d, "Rating": %f, "Message": "%s" }`,
		id,
		math.NaN(),
		msg)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(reply)) // nolint: errcheck
} // func (srv *Server) handleRateItem(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleUnrateItem(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle %s from %s\n",
		r.URL,
		r.RemoteAddr)

	var (
		err               error
		db                *database.Database
		idStr, msg, reply string
		id                int64
		item              *feed.Item
	)

	vars := mux.Vars(r)

	idStr = vars["id"]

	if id, err = strconv.ParseInt(idStr, 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse ID %q: %s",
			idStr,
			err.Error())
		goto SEND_ERROR_MESSAGE
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if item, err = db.ItemGetByID(id); err != nil {
		msg = fmt.Sprintf("Cannot load Item %d: %s",
			id,
			err.Error())
		goto SEND_ERROR_MESSAGE
	} else if item == nil {
		msg = fmt.Sprintf("No such Item: %d", id)
		goto SEND_ERROR_MESSAGE
	} else if err = db.ItemRatingClear(item); err != nil {
		msg = fmt.Sprintf("Cannot clear Rating of Item %d (%q): %s",
			id,
			item.Title,
			err.Error())
		goto SEND_ERROR_MESSAGE
	}

	reply = fmt.Sprintf(`{ "Status": true, "ID": %d, "Message": "Success" }`,
		id)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(reply)) // nolint: errcheck
	return

SEND_ERROR_MESSAGE:
	srv.log.Printf("[ERROR] %s\n", msg)
	srv.SendMessage(msg)
	reply = fmt.Sprintf(`{ "Status": false, "ID": %d, "Message": "%s" }`,
		id,
		msg)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(reply)) // nolint: errcheck
} // func (srv *Server) handleUnrateItem(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleRebuildFTS(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handle %s from %s\n",
		r.URL,
		r.RemoteAddr)

	var (
		err        error
		db         *database.Database
		msg, reply string
		status     bool
	)

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if err = db.FTSRebuild(); err != nil {
		msg = fmt.Sprintf("Cannot rebuild FTS index: %s",
			err.Error())
	} else {
		msg = "FTS index was rebuilt successfully."
		status = true
	}

	reply = fmt.Sprintf(`{ "Status": %t, "Message": "%s" }`,
		status,
		msg)
	w.Header().Set("Content-Type", "application/json")
	// w.Header().Set("Content-Length", strconv.FormatInt(len(reply), 10))
	w.WriteHeader(200)
	w.Write([]byte(reply)) // nolint: errcheck
} // func (srv *Server) handleRebuildFTS(w http.ResponseWriter, r *http.request)
