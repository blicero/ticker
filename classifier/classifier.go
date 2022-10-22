// /home/krylon/go/src/github.com/blicero/ticker/classifier/classifier.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 10. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-10-22 16:37:04 krylon>

// Package classifier implements semi-automatic rating of news items.
package classifier

import (
	"regexp"

	"github.com/blicero/ticker/feed"
)

// Good and Bad are the two categories for rating news items.
const (
	Good = "good"
	Bad  = "bad"
)

var nonword = regexp.MustCompile(`\W+`)

// Classifier is a ... well, classifier that tries to tell interesting news
// from boring ones.
// Since I am trying to replace it, this type has become an interface so I can
// swap out several implementations without upsetting the rest of the
// application's code.
type Classifier interface {
	Train() error
	Classify(item *feed.Item) (string, error)
	Learn(class string, item *feed.Item) error
	Unlearn(class string, item *feed.Item) error
}
