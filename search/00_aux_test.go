// /home/krylon/go/src/ticker/search/00_aux_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 21. 06. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-21 23:05:29 krylon>

package search

import "time"

func mkdate(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
} // func mkdate(year, month, day) time.Time
