// /home/krylon/go/src/ticker/web/suggest.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 03. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-05-23 17:26:21 krylon>

package web

import (
	"ticker/advisor"
	"ticker/feed"
)

func (srv *Server) suggestTags(items []feed.Item) (map[int64]map[string]advisor.SuggestedTag, error) {
	// var (
	// 	err error
	// 	adv *advisor.Advisor
	// )

	// if adv, err = advisor.NewAdvisor(); err != nil {
	// 	srv.log.Printf("[ERROR] Cannot create Advisor: %s\n",
	// 		err.Error())
	// 	return nil, err
	// } else if err = adv.Train(); err != nil {
	// 	srv.log.Printf("[ERROR] Cannot train Advisor: %s\n",
	// 		err.Error())
	// 	return nil, err
	// }

	var sugg = make(map[int64]map[string]advisor.SuggestedTag, len(items))

	for _, item := range items {
		sugg[item.ID] = srv.clsTags.Suggest(&item)
	}

	return sugg, nil
} // func (srv *Server) suggestTags(items []feed.Item) error
