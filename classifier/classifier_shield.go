// /home/krylon/go/src/github.com/blicero/ticker/classifier/classifier_shield.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 10. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-10-15 16:47:35 krylon>

package classifier

import (
	"log"
	"path/filepath"

	"github.com/blicero/shield"
	"github.com/blicero/ticker/common"
	"github.com/blicero/ticker/database"
	"github.com/blicero/ticker/feed"
	"github.com/blicero/ticker/logdomain"
	"github.com/endeveit/guesslanguage"
)

// ClassifierShield is an implementation of a classifier that uses shield as
// its Bayes-engine, so to speak.
type ClassifierShield struct {
	pool   *database.Pool
	log    *log.Logger
	shield map[string]shield.Shield
}

// NewShield creates and returns a new ClassifierShield.
func NewShield(pool *database.Pool) (*ClassifierShield, error) {
	var (
		err error
		c   = &ClassifierShield{
			shield: map[string]shield.Shield{
				"de": shield.New(
					shield.NewGermanTokenizer(),
					shield.NewLevelDBStore(filepath.Join(
						common.ClassifierDir,
						"de")),
				),
				"en": shield.New(
					shield.NewEnglishTokenizer(),
					shield.NewLevelDBStore(
						filepath.Join(
							common.ClassifierDir,
							"en",
						),
					),
				),
			},
			pool: pool,
		}
	)

	if c.log, err = common.GetLogger(logdomain.Classifier); err != nil {
		return nil, err
	}

	return c, nil
} // func NewShield() (*ClassifierShield, error)

// Trains trains the Classifier.
func (c *ClassifierShield) Train() error {
	var (
		err   error
		items []feed.Item
		db    *database.Database
	)

	db = c.pool.Get()
	defer c.pool.Put(db)

	if items, err = db.ItemGetRated(); err != nil {
		c.log.Printf("[ERROR] Cannot load rated Items: %s\n",
			err.Error())
		return err
	}

	for _, i := range items {
		var (
			s           shield.Shield
			lang, class string
			body        = i.Plaintext()
		)

		if lang, err = guesslanguage.Guess(body); err != nil {
			c.log.Printf("[ERROR] Cannot determine language of Item %q: %s\n",
				i.Title,
				err.Error())
			lang = "en"
		}

		s = c.shield[lang]

		if i.Rating >= 0.5 {
			class = Good
		} else {
			class = Bad
		}

		if err = s.Learn(class, body); err != nil {
			c.log.Printf("[ERROR] Failed to learn Item %d (%s): %s\n",
				i.ID,
				i.Title,
				err.Error())
			return err
		}
	}

	return nil
} // func (c *ClassifierShield) Train() error

// Classify attempts to find a rating for a news item.
func (c *ClassifierShield) Classify(item *feed.Item) (string, error) {
	var (
		err          error
		rating, lang string
		body         = item.Plaintext()
	)

	if lang, err = guesslanguage.Guess(body); err != nil {
		return "", err
	} else if rating, err = c.shield[lang].Classify(body); err != nil {
		return "", err
	}

	return rating, nil
} // func (c *ClassifierShield) Classify(item *feed.Item) (string, error)
