// /home/krylon/go/src/ticker/database/00_database_aux_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-03 19:50:07 krylon>

package database

import "ticker/feed"

func feedEqual(f1, f2 *feed.Feed) bool {
	if f1 == f2 {
		return true
	}

	return f1.ID == f2.ID &&
		f1.Name == f2.Name &&
		f1.URL == f2.URL &&
		f1.Interval == f2.Interval &&
		f1.LastUpdate.Equal(f2.LastUpdate) &&
		f1.Active == f2.Active
} // func feedEqual(f1, f2 *feed.Feed) bool
