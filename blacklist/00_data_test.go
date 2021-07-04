// /home/krylon/go/src/ticker/blacklist/00_data_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-07-04 13:36:57 krylon>

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
	blacklistTestCase{
		patterns: []string{
			"(?i)vgwort[.]de",
			"(?i)ioam[.]",
			"(?i)outbrain[.]",
		},
		samples: []itemTestCase{
			itemTestCase{
				sample: "https://www.vgwort.de/fuck-you.js",
				match:  true,
			},
			itemTestCase{
				sample: "https://en.wikipedia.org/wiki/Polar_bear",
				match:  false,
			},
		},
	},
	blacklistTestCase{
		patterns: []string{
			"(?zebu",
		},
		err: true,
	},
}
