// /home/krylon/go/src/ticker/search/01_parse_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 21. 06. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-21 22:28:48 krylon>

package search

import "testing"

func TestParseQueryString(t *testing.T) {
	type testCase struct {
		qstr string
		res  Query
		err  bool
	}

	var qlist = []testCase{
		testCase{
			qstr: "something and something else",
			res: Query{
				Query: []string{
					"and",
					"else",
					"something",
					"something",
				},
			},
		},
		testCase{
			qstr: "sqlite regex tag:Database tag:Programming",
			res: Query{
				Query: []string{
					"regex",
					"sqlite",
				},
				Tags: []string{
					"Database",
					"Programming",
				},
			},
		},
	}

	for _, c := range qlist {
		var (
			err error
			q   *Query
		)

		if q, err = ParseQueryStr(nil, c.qstr); err != nil {
			if !c.err {
				t.Errorf("Cannot parse Query %q: %s",
					c.qstr,
					err.Error())
			}
		} else if !q.Equal(&c.res) {
			t.Errorf(`Unexpected result for query %q:
Expected: %v
Got:      %v`,
				c.qstr,
				&c.res,
				q)
		}
	}
}
