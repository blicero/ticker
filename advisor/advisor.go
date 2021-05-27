// /home/krylon/go/src/ticker/tag/advisor.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 03. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-05-27 14:20:07 krylon>

// Package advisor provides suggestions on what Tags one might want to attach
// to news Items.
package advisor

import (
	"log"
	"regexp"
	"runtime"
	"ticker/common"
	"ticker/database"
	"ticker/feed"
	"ticker/logdomain"
	"ticker/tag"

	bayesian "github.com/CyrusF/go-bayesian"
	"github.com/bbalet/stopwords"
	"github.com/dchest/stemmer/german"
	"github.com/dchest/stemmer/porter2"
	"github.com/endeveit/guesslanguage"
)

var nonword = regexp.MustCompile(`\W+`)

// SuggestedTag is a suggestion to attach a specific Tag to a specific Item.
type SuggestedTag struct {
	tag.Tag
	Score float64
}

// Advisor can suggest Tags for News Items.
type Advisor struct {
	db   *database.Database
	log  *log.Logger
	cls  bayesian.Classifier
	tags map[string]tag.Tag
}

// NewAdvisor returns a new Advisor, but it does not train it, yet.
func NewAdvisor() (*Advisor, error) {
	var (
		err error
		adv = &Advisor{
			// cls: bayesian.NewClassifier(bayesian.MultinomialTf),
		}
	)

	if adv.log, err = common.GetLogger(logdomain.Tag); err != nil {
		return nil, err
	} else if adv.db, err = database.Open(common.DbPath); err != nil {
		adv.log.Printf("[ERROR] Cannot open database: %s\n",
			err.Error())
		return nil, err
	} else if err = adv.loadTags(); err != nil {
		return nil, err
	}

	return adv, nil
} // func NewAdvisor() (*Advisor, error)

func (adv *Advisor) loadTags() error {
	var (
		err  error
		tags []tag.Tag
	)

	if tags, err = adv.db.TagGetAll(); err != nil {
		adv.log.Printf("[ERROR] Cannot load all Tags from database: %s\n",
			err.Error())
		return err
	}

	adv.tags = make(map[string]tag.Tag, len(tags))

	for _, t := range tags {
		adv.tags[t.Name] = t
	}

	return nil
} // func (adv *advisor) loadTags() error

// Train trains the Advisor based on the Tags that have been attached to
// Items previously.
func (adv *Advisor) Train() error {
	var (
		err   error
		items []feed.Item
	)

	// XXX This approach is grossly inefficient.

	if items, err = adv.db.ItemGetAll(-1, 0); err != nil {
		adv.log.Printf("[ERROR] Cannot load all Tags: %s\n",
			err.Error())
		return err
	}

	var docs = make([]bayesian.Document, 0, 256)

	for _, item := range items {
		if len(item.Tags) == 0 {
			continue
		}

		var tokens = adv.tokenize(&item)

		for _, t := range item.Tags {
			var doc = bayesian.Document{
				Tokens: tokens,
				Class:  bayesian.Class(t.Name),
			}

			docs = append(docs, doc)
		}
	}

	adv.cls = bayesian.NewClassifier(bayesian.MultinomialTf)
	adv.cls.Learn(docs...)

	return nil
} // func (adv *Advisor) Train() error

// Suggest returns a map Tags and how likely they apply to the given Item.
func (adv *Advisor) Suggest(item *feed.Item) map[string]SuggestedTag {
	var (
		sugg map[string]SuggestedTag
		res  map[bayesian.Class]float64
	)

	res, _, _ = adv.cls.Classify(adv.tokenize(item)...)

	sugg = make(map[string]SuggestedTag, len(res))

	for c, r := range res {
		adv.log.Printf("[TRACE] SUGGEST Item %q (%d): Tag %q -> %.2f\n",
			item.Title,
			item.ID,
			c,
			r)

		var t = adv.tags[string(c)]

		sugg[t.Name] = SuggestedTag{
			Tag:   t,
			Score: r,
		}
	}

	return sugg
} // func (adv *Advisor) Suggest(item *feed.Item) map[string]float64

func (adv *Advisor) tokenize(item *feed.Item) []string {
	var (
		body, lang string
	)

	// body = item.Title + " " + item.Description

	lang, body = adv.getLanguage(item.Title, item.Description)

	body = stopwords.CleanString(body, lang, true)

	var words = nonword.Split(body, -1)

	var tokens = make([]string, len(words))

	for i, w := range words {
		var s = stemWord(w, lang)
		tokens[i] = s
	}

	return tokens
} // func (c *Advisor) tokenize(item *feed.Item) []string

func (adv *Advisor) getLanguage(title, description string) (lng, fullText string) {
	const defaultLang = "en"
	var (
		err        error
		lang, body string
	)

	body = title + " " + description

	defer func() {
		if x := recover(); x != nil {
			var buf [2048]byte
			var cnt = runtime.Stack(buf[:], false)
			adv.log.Printf("[CRITICAL] Panic in getLanguage for Item %q: %s\n%s",
				title,
				x,
				string(buf[:cnt]))
			lng = defaultLang
			fullText = body
		}
	}()

	if lang, err = guesslanguage.Guess(body); err != nil {
		adv.log.Printf("[ERROR] Cannot determine language of Item %q: %s\n",
			title,
			err.Error())
		lang = defaultLang
	}

	return lang, body
} // func getLanguage(title, description string) (string, string)

func stemWord(word, lang string) string {
	switch lang {
	case "de":
		return german.Stemmer.Stem(word)
	case "en":
		return porter2.Stemmer.Stem(word)
	default:
		// I will try this first, if it does now work out,
		// I return word verbatim.
		return porter2.Stemmer.Stem(word)
	}
} // func stem_word(word, lang string) string
