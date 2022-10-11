// /home/krylon/go/src/ticker/classifier/classifier.go
// -*- mode: go; coding: utf-8; -*-
// Created on 17. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-05-22 16:38:09 krylon>

// Package classifier implements semi-automatic rating of news items.
package classifier

import (
	"log"
	"regexp"
	"github.com/blicero/ticker/common"
	"github.com/blicero/ticker/database"
	"github.com/blicero/ticker/feed"
	"github.com/blicero/ticker/logdomain"

	"github.com/n3integration/classifier/naive"
)

// Good and Bad are the two categories for rating news items.
const (
	Good = "good"
	Bad  = "bad"
)

var nonword = regexp.MustCompile(`\W+`)

// Classifier is a classical Bayesian classifier that semi-automatically
// rates news Items.
type Classifier struct {
	db  *database.Database
	rev *naive.Classifier
	log *log.Logger
}

// New creates a new Classifier.
func New() (*Classifier, error) {
	var (
		err error
		c   = new(Classifier)
	)

	if c.log, err = common.GetLogger(logdomain.Classifier); err != nil {
		return nil, err
	} else if c.db, err = database.Open(common.DbPath); err != nil {
		c.log.Printf("[ERROR] Cannnot open database at %s: %s\n",
			common.DbPath,
			err.Error())
		return nil, err
	}

	c.rev = naive.New()

	return c, nil
} // func New() (*Classifier, error)

// Train trains the model. Duh.
func (c *Classifier) Train() error {
	var (
		err   error
		items []feed.Item
	)

	if items, err = c.db.ItemGetRated(); err != nil {
		c.log.Printf("[ERROR] Cannot load rated Items: %s\n",
			err.Error())
		return err
	}

	c.rev = naive.New()
	c.learn(items)

	return nil
} // func (c *Classifier) Train() error

func (c *Classifier) learn(items []feed.Item) {
	for _, item := range items {
		var (
			err   error
			class string
			s     = nonword.ReplaceAllString(item.Plaintext(), " ")
		)

		if item.Rating >= 0.5 {
			class = Good
		} else {
			class = Bad
		}

		if err = c.rev.TrainString(s, class); err != nil {
			c.log.Printf("[ERROR] Cannot traing Item %s (%d): %s\n",
				item.Title,
				item.ID,
				err.Error())
		}
	}
} // func (c *Classifier) learn(items []feed.Item)

// Classify attempts to find a rating for a news item.
func (c *Classifier) Classify(item *feed.Item) (string, error) {
	var (
		err    error
		rating string
		p      = nonword.ReplaceAllString(item.Plaintext(), " ")
	)

	if rating, err = c.rev.ClassifyString(p); err != nil {
		c.log.Printf("[ERROR] Cannot classify Item %s (%d): %s\n",
			item.Title,
			item.ID,
			err.Error())
		return "", err
	}

	return rating, nil
} // func (c *Classifier) Classify(item *feed.Item) (map[bayesian.Class]float64, bayesian.Class, bool)

// func (c *Classifier) tokenize(item *feed.Item) []string {
// 	var (
// 		err        error
// 		body, lang string
// 	)

// 	// body = item.Title + " " + item.Description
// 	body = item.Plaintext()

// 	if lang, err = guesslanguage.Guess(body); err != nil {
// 		c.log.Printf("[ERROR] Cannot determine language of Item %q: %s\n",
// 			item.Title,
// 			err.Error())
// 		lang = "en"
// 	}

// 	body = stopwords.CleanString(body, lang, true)

// 	var words = nonword.Split(body, -1)

// 	var tokens = make([]string, len(words))

// 	for i, w := range words {
// 		var s = stemWord(w, lang)
// 		tokens[i] = s
// 	}

// 	return tokens
// } // func (c *Classifier) tokenize(item *feed.Item) []string

// func stemWord(word, lang string) string {
// 	switch lang {
// 	case "de":
// 		return german.Stemmer.Stem(word)
// 	case "en":
// 		return porter2.Stemmer.Stem(word)
// 	default:
// 		// I will try this first, if it does now work out,
// 		// I return word verbatim.
// 		return porter2.Stemmer.Stem(word)
// 	}
// } // func stem_word(word, lang string) string
