// /home/krylon/go/src/ticker/search/00_main_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 21. 06. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-22 11:01:12 krylon>

package search

import (
	"fmt"
	"os"
	"testing"
	"github.com/blicero/ticker/common"
	"time"

	"github.com/blicero/krylib"
)

func TestMain(m *testing.M) {
	var (
		err     error
		result  int
		baseDir = time.Now().Format("/tmp/ticker_search_test_20060102_150405")
	)

	if err = common.SetBaseDir(baseDir); err != nil {
		fmt.Printf("Cannot set base directory to %s: %s\n",
			baseDir,
			err.Error())
		os.Exit(1)
	} else if err = krylib.CopyFile(dbPath, common.DbPath); err != nil {
		fmt.Printf("Failed to copy test database to %s: %s\n",
			common.DbPath,
			err.Error())
		os.Exit(1)
	} else if result = m.Run(); result == 0 {
		// If any test failed, we keep the test directory (and the
		// database inside it) around, so we can manually inspect it
		// if needed.
		// If all tests pass, OTOH, we can safely remove the directory.
		fmt.Printf("Removing BaseDir %s\n",
			baseDir)
		_ = os.RemoveAll(baseDir)
	} else {
		fmt.Printf(">>> TEST DIRECTORY: %s\n", baseDir)
	}

	os.Exit(result)
} // func TestMain(m *testing.M)
