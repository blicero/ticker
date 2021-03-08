// /home/krylon/go/src/ticker/query/query.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-03-08 22:14:25 krylon>

//go:generate stringer -type=ID

// Package query provides symbolic constants to identify SQL queries.
package query

type ID uint8

const (
	FeedAdd ID = iota
	FeedGetAll
	FeedGetDue
	FeedGetByID
	FeedSetActive
	FeedSetTimestamp
	FeedDelete
	FeedModify
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
	ItemHasDuplicate
	FTSClear
	TagCreate
	TagDelete
	TagGetAll
	TagGetRoots
	TagGetByID
	TagGetByName
	TagGetChildren
	TagGetChildrenImmediate
	TagGetByItem
	TagNameUpdate
	TagDescriptionUpdate
	TagParentSet
	TagParentClear
	TagLinkCreate
	TagLinkDelete
	TagLinkGetByItem
	ReadLaterAdd
	ReadLaterGetByItem
	ReadLaterGetAll
	ReadLaterGetUnread
	ReadLaterMarkRead
	ReadLaterMarkUnread
	ReadLaterDelete
	ReadLaterDeleteRead
	ReadLaterSetDeadine
	ReadLaterSetNote
)
