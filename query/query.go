// /home/krylon/go/src/ticker/query/query.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-07 14:23:35 krylon>

//go:generate stringer -type=ID

// Package query provides symbolic constants to identify SQL queries.
package query

type ID uint8

const (
	FeedAdd ID = iota
	FeedGetAll
	FeedGetByID
	FeedSetTimestamp
	FeedDelete
	ItemAdd
	ItemGetRecent
	ItemGetByID
	ItemGetByURL
)