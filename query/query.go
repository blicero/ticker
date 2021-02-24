// /home/krylon/go/src/ticker/query/query.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-24 20:01:24 krylon>

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
	ItemGetByTag
	ItemRatingSet
	ItemRatingClear
	FTSClear
	TagCreate
	TagDelete
	TagNameUpdate
	TagDescriptionUpdate
	TagParentSet
	TagParentClear
	TagLinkCreate
	TagLinkDelete
	TagLinkGetByItem
)
