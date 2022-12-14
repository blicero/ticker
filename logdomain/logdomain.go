// /home/krylon/go/src/ticker/logdomain/logdomain.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-23 16:19:52 krylon>

// Package logdomain provides constants to identify the different
// "areas" of the application that perform logging.
package logdomain

//go:generate stringer -type=ID

// ID represents an area of concern.
type ID uint8

// These constants identify the various logging domains.
const (
	Common ID = iota
	Classifier
	DBPool
	Database
	Download
	Feed
	Prefetch
	Reader
	Search
	Tag
	Web
)

// AllDomains returns a slice of all the known log sources.
func AllDomains() []ID {
	return []ID{
		Common,
		Classifier,
		DBPool,
		Database,
		Download,
		Feed,
		Prefetch,
		Reader,
		Search,
		Tag,
		Web,
	}
} // func AllDomains() []ID
