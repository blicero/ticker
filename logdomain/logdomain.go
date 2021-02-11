// /home/krylon/go/src/ticker/logdomain/logdomain.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-11 17:44:44 krylon>

// Package logdomain provides constants to identify the different
// "areas" of the application that perform logging.
package logdomain

//go:generate stringer -type=ID

// ID represents an area of concern.
type ID uint8

// These constants identify the various logging domains.
const (
	Common ID = iota
	DBPool
	Database
	Feed
	Reader
)

// AllDomains returns a slice of all the known log sources.
func AllDomains() []ID {
	return []ID{
		Common,
		DBPool,
		Database,
		Feed,
		Reader,
	}
} // func AllDomains() []ID
