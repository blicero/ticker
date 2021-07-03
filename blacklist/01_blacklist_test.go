// /home/krylon/go/src/ticker/blacklist/01_blacklist_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-07-03 22:02:31 krylon>

package blacklist

import "testing"

var l Blacklist

func TestListCreate(t *testing.T) {
	var err error

	if l, err = NewBlacklist(testCases[0].patterns); err != nil {
		l = nil
		t.Fatalf("Error creating Blacklist 01: %s",
			err.Error())
	}
} // func TestCreateList(t *testing.T)

func TestListMatch(t *testing.T) {
	for _, s := range testCases[0].samples {
		if res := l.Match(s.sample); res != s.match {
			t.Errorf("Unexpected result for %q: %t (expected %t)",
				s.sample,
				res,
				s.match)
		}
	}
} // func TestListMatch(t *testing.T)
