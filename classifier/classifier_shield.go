// /home/krylon/go/src/github.com/blicero/ticker/classifier/classifier_shield.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 10. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-10-17 21:52:54 krylon>

package classifier

import (
	"log"
	"path/filepath"
	"runtime"
	"strings"

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

	// for k, v := range c.shield {
	// 	c.log.Printf("[DEBUG] Shield Classifier for %s: %s\n",
	// 		k,
	// 		spew.Sdump(v))
	// }

	return c, nil
} // func NewShield() (*ClassifierShield, error)

// Trains trains the Classifier.
func (c *ClassifierShield) Train() error {
	var (
		err   error
		items []feed.Item
		db    *database.Database
	)

	for k, v := range c.shield {
		c.log.Printf("[DEBUG] Reset Shield instance for %s\n",
			k)
		if err = v.Reset(); err != nil {
			c.log.Printf("[ERROR] Cannot reset Shield/%s: %s\n",
				k,
				err.Error())
			return err
		}
	}

	db = c.pool.Get()
	defer c.pool.Put(db)

	if items, err = db.ItemGetRated(); err != nil {
		c.log.Printf("[ERROR] Cannot load rated Items: %s\n",
			err.Error())
		return err
	}

	for _, i := range items {
		var (
			s                 shield.Shield
			lang, class, body string
		)

		lang, body = c.getLanguage(&i)

		if s = c.shield[lang]; s == nil {
			s = c.shield["en"]
		}

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
		err                error
		rating, lang, body string
		s                  shield.Shield
	)

	lang, body = c.getLanguage(item)

	if s = c.shield[lang]; s == nil {
		s = c.shield["en"]
	}

	if rating, err = s.Classify(body); err != nil {
		return "", err
	}

	return rating, nil
} // func (c *ClassifierShield) Classify(item *feed.Item) (string, error)

func (c *ClassifierShield) getLanguage(item *feed.Item) (lng, fullText string) {
	const (
		defaultLang = "en"
		blString    = "Lauren Boebert buried in ridicule after claim about 1930s Germany"
	)

	var (
		err        error
		lang, body string
	)

	body = item.Plaintext()

	defer func() {
		if x := recover(); x != nil {
			if !strings.Contains(item.Title, blString) {
				var buf [2048]byte
				var cnt = runtime.Stack(buf[:], false)
				c.log.Printf("[CRITICAL] Panic in getLanguage for Item %q: %s\n%s",
					item.Title,
					x,
					string(buf[:cnt]))
			}
			lng = defaultLang
			fullText = body
		}
	}()

	if lang, err = guesslanguage.Guess(body); err != nil {
		c.log.Printf("[ERROR] Cannot determine language of Item %q: %s\n",
			item.Title,
			err.Error())
		lang = defaultLang
	}

	return lang, body
} // func getLanguage(title, description string) (string, string)
