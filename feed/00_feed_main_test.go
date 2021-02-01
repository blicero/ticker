// /home/krylon/go/src/ticker/feed/00_feed_main_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-01 16:09:04 krylon>

package feed

import (
	"fmt"
	"log"
	"os"
	"testing"
	"ticker/common"
	"ticker/logdomain"
	"time"
)

var flog *log.Logger

// TestMain runs the test suite.
func TestMain(m *testing.M) {
	var (
		err      error
		testPath = time.Now().Format("/tmp/ticker_feed_test_20060102_150405")
	)

	if err = common.SetBaseDir(testPath); err != nil {
		fmt.Printf("Cannot initialize testing directory %s: %s\n",
			testPath,
			err.Error())
		os.Exit(1)
	} else if flog, err = common.GetLogger(logdomain.Feed); err != nil {
		fmt.Printf("Cannot get Logger for %s: %s\n",
			logdomain.Feed,
			err.Error())
		os.Exit(1)
	}

	// defer profile.Start(profile.CPUProfile).Stop()

	var result int

	if result = m.Run(); result == 0 {
		// If any test failed, we keep the test directory (and the
		// database inside it) around, so we can manually inspect it
		// if needed.
		// If all tests pass, OTOH, we can safely remove the directory.

		fmt.Printf("Removing BaseDir %s\n",
			testPath)
		_ = os.RemoveAll(testPath) // nolint: gosec
	} else {
		fmt.Printf(">>> TEST DIRECTORY: %s\n", testPath)
	}

	os.Exit(result)
} // func TestMain(m *testing.M)
