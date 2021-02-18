// /home/krylon/go/src/ticker/database/01_database_init_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-18 18:36:28 krylon>

package database

import (
	"testing"
	"ticker/common"
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
