// /home/krylon/go/src/ticker/blacklist/01_blacklist_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-07-04 12:57:51 krylon>

package blacklist

import "testing"

func TestBlacklist(t *testing.T) {
	for idx, c := range testCases {
		var (
			err error
			bl  Blacklist
		)

		if bl, err = NewBlacklist(c.patterns); err != nil {
			if !c.err {
				t.Errorf("Cannot compile Blacklist %d: %s",
					idx+1,
					err.Error())
			}
			continue
		}

		for _, sample := range c.samples {
			if m := bl.Match(sample.sample); m != sample.match {
				t.Errorf("Unexpected result for input %q: %t",
					sample.sample,
					m)
			}
		}
	}
} // func TestBlacklist(t *testing.T)
