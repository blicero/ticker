// /home/krylon/go/src/ticker/download/blacklist.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-07-03 17:35:07 krylon>

// Package blacklist implements a generic blacklist for strings.
// It is intended for use in the Downloader and Prefetcher to
// filter out undesirable URLs (e.g. Google Analytics, ad networks, ...).
package blacklist

import (
	"regexp"
	"sort"
)

type blacklistItem struct {
	pat    *regexp.Regexp
	hitCnt int
}

// Blacklist is a list of regular expressions against which a string can be checked.
type Blacklist []blacklistItem

// NewBlacklist creates a fresh Blacklist from the given list of patterns.
func NewBlacklist(patterns []string) (Blacklist, error) {
	var (
		err error
		bl  Blacklist
	)

	bl = make(Blacklist, len(patterns))

	for idx, pat := range patterns {
		if bl[idx].pat, err = regexp.Compile(pat); err != nil {
			return nil, err
		}
	}

	return bl, nil
} // func NewBlacklist(patterns []string) (Blacklist, error)

func (bl Blacklist) Len() int           { return len(bl) }
func (bl Blacklist) Swap(i, j int)      { bl[i], bl[j] = bl[j], bl[i] }
func (bl Blacklist) Less(i, j int) bool { return bl[j].hitCnt < bl[i].hitCnt }

// Match returns true if the given string is matched by one of the patterns
// comprising the Blacklist.
func (bl Blacklist) Match(href string) bool {
	for idx, item := range bl {
		if item.pat.MatchString(href) {
			bl[idx].hitCnt++
			sort.Sort(bl)
			return true
		}
	}

	return false
} // func (bl blacklist) Match(href string) bool
