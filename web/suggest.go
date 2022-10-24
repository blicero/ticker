// /home/krylon/go/src/ticker/web/suggest.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 03. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2022-10-24 19:25:28 krylon>

package web

import (
	"github.com/blicero/ticker/advisor"
	"github.com/blicero/ticker/feed"
)

const maxSuggestions = 10

func (srv *Server) suggestTags(items []feed.Item) (map[int64]map[string]advisor.SuggestedTag, error) {
	var sugg = make(map[int64]map[string]advisor.SuggestedTag, len(items))

	for _, item := range items {
		sugg[item.ID] = srv.clsTags.Suggest(&item, maxSuggestions)
	}

	return sugg, nil
} // func (srv *Server) suggestTags(items []feed.Item) error
