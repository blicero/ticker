// /home/krylon/go/src/ticker/search/02_search_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 22. 06. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-22 14:41:58 krylon>

package search

import (
	"testing"
	"github.com/blicero/ticker/common"
	"github.com/blicero/ticker/database"
	"github.com/blicero/ticker/feed"
)

func TestOpenDB(t *testing.T) {
	var err error

	if testdb, err = database.Open(common.DbPath); err != nil {
		t.Fatalf("Cannot open database at %s: %s",
			common.DbPath,
			err.Error())
		testdb = nil
	}
} // func TestOpenDB(t *testing.T)

func TestSearch001(t *testing.T) {
	if testdb == nil {
		t.SkipNow()
	}

	type testCase struct {
		qstr string
		err  bool
		res  []int64
	}

	var tlist = []testCase{
		testCase{
			qstr: "google",
			res: []int64{
				25014,
				24967,
				23303,
				22562,
				21805,
				20709,
				20711,
				19793,
				18630,
				18248,
				18249,
				17334,
				16981,
				15299,
				15094,
				14498,
				12548,
				11303,
				11274,
				10013,
				9893,
				9640,
				9367,
				8821,
				8778,
				6969,
				6953,
				6496,
				6338,
				4965,
				5107,
				4538,
				4202,
				3583,
				4665,
				3142,
				4666,
				2690,
				3247,
				2455,
				6264,
				1901,
				1902,
				1529,
				3248,
				4673,
				732,
				4674,
				739,
				4677,
				3254,
				3256,
			},
		},
		testCase{
			qstr: `google tag:"Operating Systems"`,
			res: []int64{
				25014,
				22562,
				19793,
				4965,
			},
		},
	}

	for _, c := range tlist {
		var (
			err error
			q   *Query
			res []feed.Item
		)

		if q, err = ParseQueryStr(testdb, c.qstr); err != nil {
			t.Errorf("Error parsing query string %q: %s",
				c.qstr,
				err.Error())
			continue
		} else if res, err = q.Execute(); err != nil {
			if !c.err {
				t.Errorf("Error executing search query %q: %s",
					c.qstr,
					err.Error())
				continue
			}
		} else if len(res) != len(c.res) {
			t.Errorf("Unexpected number of results for query %q: %d (expect %d)",
				c.qstr,
				len(res),
				len(c.res))
			continue
		}
	}
} // func TestSearch001(t *testing.T)
