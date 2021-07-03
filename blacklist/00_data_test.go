// /home/krylon/go/src/ticker/blacklist/00_data_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-07-03 22:05:33 krylon>

package blacklist

type itemTestCase struct {
	sample string
	match  bool
}

type blacklistTestCase struct {
	patterns []string
	samples  []itemTestCase
	err      bool // nolint: structcheck,unused
}

var testCases = []blacklistTestCase{
	blacklistTestCase{
		patterns: []string{
			"(?i)doubleclick[.](?:net|com|biz|de)",
			"(?i)google-analytics[.]",
			"(?i)facebook[.]com",
		},
		samples: []itemTestCase{
			itemTestCase{
				sample: "www.google-analytics.com/bigbrother.js",
				match:  true,
			},
			itemTestCase{
				sample: "blog.fefe.de/?q=Merkel",
				match:  false,
			},
			itemTestCase{
				sample: "www.facebook.com/mark-zuckerberg-is-an-asshole.js",
				match:  true,
			},
		},
	},
}
