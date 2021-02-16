// /home/krylon/go/src/ticker/web/web.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-16 18:19:23 krylon>

package web

import (
	"errors"
	"fmt"
	"krylib"
	"log"
	"net/http"
	"strconv"
	"text/template"
	"ticker/common"
	"ticker/database"
	"ticker/feed"
	"ticker/logdomain"
	"time"

	"github.com/gorilla/mux"
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
	}

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

	srv.router.HandleFunc("/ajax/beacon", srv.handleBeacon)

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

	// if data.Notes, err = db.NoteGetRecent(5); err != nil {
	// 	msg = fmt.Sprintf("Lookup of recent Notes failed: %s",
	// 		err.Error())
	// 	srv.log.Println("[ERROR] " + msg)
	// 	srv.SendMessage(msg)
	// 	data.Notes = make([]zettel.Zettel, 0)
	// }

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
	srv.log.Printf("[TRACE] Handle request for %s\n",
		r.URL.EscapedPath())

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
