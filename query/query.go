// /home/krylon/go/src/ticker/query/query.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2022-10-10 18:10:20 krylon>

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
	ItemGetSearchExtended
	ItemGetContent
	ItemGetByTag
	ItemGetByTagRecursive
	ItemGetPrefetch
	ItemGetTotalCnt
	ItemRatingSet
	ItemRatingClear
	ItemHasDuplicate
	ItemPrefetchSet
	FTSClear
	TagCreate
	TagDelete
	TagGetAll
	TagGetRoots
	TagGetByID
	TagGetByName
	TagGetChildren
	TagGetChildrenImmediate
	TagGetAllByHierarchy
	TagGetByItem
	TagNameUpdate
	TagDescriptionUpdate
	TagParentSet
	TagParentClear
	TagUpdate
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
