// /home/krylon/go/src/ticker/database/01_database_init_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-25 18:08:45 krylon>

package database

import (
	"testing"
	"github.com/blicero/ticker/common"
)

func TestCreateDatabase(t *testing.T) {
	var err error

	if db, err = Open(common.DbPath); err != nil {
		db = nil
		t.Fatalf("Cannot open database at %s: %s",
			common.DbPath,
			err.Error())
	}
} // func TestCreateDatabase(t *testing.T)

// We prepare each query once to make sure there are no syntax errors in the SQL.
func TestPrepareQueries(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	for id := range dbQueries {
		var err error
		if _, err = db.getQuery(id); err != nil {
			t.Errorf("Cannot prepare query %s: %s",
				id,
				err.Error())
		}
	}
} // func TestPrepareQueries(t *testing.T)
