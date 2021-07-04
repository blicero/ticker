// /home/krylon/go/src/ticker/blacklist/default.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 07. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-07-04 14:15:54 krylon>

package blacklist

var defaultPatterns = []string{
	"(?i)vgwort[.]",
	"(?i)ioam[.]",
	"(?i)google-analytics[.]",
	"(?i)newrelic[.]",
	"(?i)doubleclick[.]",
	"(?i)google-?syndication[.]",
	"(?i)sensic[.]net",
	"(?i)xiti[.]com",
	"(?i)tracker",
	"(?i:facebook|twitter|linkedin|instagram|youtube)[.]",
}

// DefaultList creates a new Blacklist using the default list of patterns.
func DefaultList() Blacklist {
	if bl, err := NewBlacklist(defaultPatterns); err != nil {
		panic(err)
	} else {
		return bl
	}
} // func DefaultList() Blacklist
