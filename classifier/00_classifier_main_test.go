// /home/krylon/go/src/ticker/classifier/00_classifier_main_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 18. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2022-10-12 19:05:09 krylon>

package classifier

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/blicero/ticker/common"
)

func TestMain(m *testing.M) {
	var (
		err     error
		result  int
		baseDir = time.Now().Format("/tmp/ticker_classifier_test_20060102_150405")
	)

	if err = common.SetBaseDir(baseDir); err != nil {
		fmt.Printf("Cannot set base directory to %s: %s\n",
			baseDir,
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
