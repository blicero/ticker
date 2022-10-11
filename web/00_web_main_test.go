// /home/krylon/go/src/ticker/web/00_web_main_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 03. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-03-11 11:02:03 krylon>

package web

import (
	"fmt"
	"os"
	"testing"
	"github.com/blicero/ticker/common"
	"time"
)

func TestMain(m *testing.M) {
	var (
		err     error
		result  int
		baseDir = time.Now().Format("/tmp/ticker_web_test_20060102_150405")
	)

	if err = common.SetBaseDir(baseDir); err != nil {
		fmt.Printf("Cannot set base directory to %s: %s\n",
			baseDir,
			err.Error())
		os.Exit(1)
		// } else if err = prepareDatabase(); err != nil {
		// 	fmt.Printf("ERROR preparing database: %s\n",
		// 		err.Error())
		// 	fmt.Printf(">>> TEST DIRECTORY: %s\n", baseDir)
		// 	os.Exit(1)
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
