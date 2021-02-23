// /home/krylon/go/src/ticker/query/query.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-23 22:00:37 krylon>

//go:generate stringer -type=ID

// Package query provides symbolic constants to identify SQL queries.
package query

type ID uint8

const (
	FeedAdd ID = iota
	FeedGetAll
	FeedGetDue
	FeedGetByID
	FeedSetTimestamp
	FeedDelete
	ItemAdd
	ItemInsertFTS
	ItemGetRecent
	ItemGetRated
	ItemGetByID
	ItemGetByURL
	ItemGetByFeed
	ItemGetAll
	ItemGetFTS
	ItemGetContent
	ItemRatingSet
	ItemRatingClear
	FTSClear
)
