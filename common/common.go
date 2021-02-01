// /home/krylon/go/src/ticker/common/common.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-01 09:15:00 krylon>

// Package common contain definitions used throughout the application
package common

import (
	"vm/logdomain"

	"github.com/hashicorp/logutils"
)

//go:generate ./build_time_stamp.pl

// AppName is the name under which the application identifies itself.
// Version is the version number.
// Debug, if true, causes the application to log additional messages and perform
// additional sanity checks.
// TimestampFormat is the default format for timestamp used throughout the
// application.
const (
	AppName                  = "Ticker"
	Version                  = "0.0.1"
	Debug                    = true
	TimestampFormatMinute    = "2006-01-02 15:04"
	TimestampFormat          = "2006-01-02 15:04:05"
	TimestampFormatSubSecond = "2006-01-02 15:04:05.0000 MST"
)

// LogLevels are the names of the log levels supported by the logger.
var LogLevels = []logutils.LogLevel{
	"TRACE",
	"DEBUG",
	"INFO",
	"WARN",
	"ERROR",
	"CRITICAL",
	"CANTHAPPEN",
	"SILENT",
}

// PackageLevels defines minimum log levels per package.
var PackageLevels = make(map[logdomain.ID]logutils.LogLevel, len(LogLevels))

func init() {
	for _, id := range logdomain.AllDomains() {
		PackageLevels[id] = MinLogLevel
	}
} // func init()

// MinLogLevel is the minimum level a log message must
// have to be written out to the log.
// This value is configurable to reduce log verbosity
// in regular use.
var MinLogLevel logutils.LogLevel = "TRACE"
