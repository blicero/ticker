// /home/krylon/go/src/ticker/tag/advisor.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 03. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2022-10-24 19:51:03 krylon>

// Package advisor provides suggestions on what Tags one might want to attach
// to news Items.
package advisor

import (
	"log"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/blicero/shield"
	"github.com/blicero/ticker/common"
	"github.com/blicero/ticker/database"
	"github.com/blicero/ticker/feed"
	"github.com/blicero/ticker/logdomain"
	"github.com/blicero/ticker/tag"

	"github.com/endeveit/guesslanguage"
)

// var nonword = regexp.MustCompile(`\W+`)

// SuggestedTag is a suggestion to attach a specific Tag to a specific Item.
type SuggestedTag struct {
	tag.Tag
	Score float64
}

// Advisor can suggest Tags for News Items.
type Advisor struct {
	db     *database.Database
	log    *log.Logger
	shield map[string]shield.Shield
	tags   map[string]tag.Tag
}

// NewAdvisor returns a new Advisor, but it does not train it, yet.
func NewAdvisor() (*Advisor, error) {
	var (
		err error
		adv = &Advisor{
			shield: map[string]shield.Shield{
				"de": shield.New(
					shield.NewGermanTokenizer(),
					shield.NewLevelDBStore(filepath.Join(
						common.AdvisorDir,
						"de")),
				),
				"en": shield.New(
					shield.NewEnglishTokenizer(),
					shield.NewLevelDBStore(
						filepath.Join(
							common.AdvisorDir,
							"en",
						),
					),
				),
			},
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

	for k, v := range adv.shield {
		adv.log.Printf("[DEBUG] Reset Shield instance for %s\n",
			k)
		if err = v.Reset(); err != nil {
			adv.log.Printf("[ERROR] Cannot reset Shield/%s: %s\n",
				k,
				err.Error())
			return err
		}
	}

	if items, err = adv.db.ItemGetAll(-1, 0); err != nil {
		adv.log.Printf("[ERROR] Cannot load all Tags: %s\n",
			err.Error())
		return err
	}

	// var docs = make([]bayesian.Document, 0, 256)

	for _, item := range items {
		var (
			lng, body string
			s         shield.Shield
		)

		if len(item.Tags) == 0 {
			continue
		}

		lng, body = adv.getLanguage(&item)

		if s = adv.shield[lng]; s == nil {
			s = adv.shield["en"]
		}

		for _, t := range item.Tags {
			if err = s.Learn(t.Name, body); err != nil {
				adv.log.Printf("[ERROR] Failed to learn Item %d (%q): %s\n",
					item.ID,
					item.Title,
					err.Error())
				return err
			}
		}
	}

	return nil
} // func (adv *Advisor) Train() error

// Learn adds a single item to the Advisor's training corpus.
func (adv *Advisor) Learn(t *tag.Tag, i *feed.Item) error {
	var (
		err       error
		lng, body string
		s         shield.Shield
	)

	lng, body = adv.getLanguage(i)

	if s = adv.shield[lng]; s == nil {
		s = adv.shield["en"]
	}

	if err = s.Learn(t.Name, body); err != nil {
		adv.log.Printf("[ERROR] Failed to learn Item %d (%q): %s\n",
			i.ID,
			i.Title,
			err.Error())
		return err
	}

	return nil
} // func (adv *Advisor) Learn(t *tag.Tag, i *feed.Item) error

func (adv *Advisor) Unlearn(t *tag.Tag, i *feed.Item) error {
	var (
		err       error
		lng, body string
		s         shield.Shield
	)

	lng, body = adv.getLanguage(i)

	if s = adv.shield[lng]; s == nil {
		s = adv.shield["en"]
	}

	if err = s.Forget(t.Name, body); err != nil {
		adv.log.Printf("[ERROR] Failed to learn Item %d (%q): %s\n",
			i.ID,
			i.Title,
			err.Error())
		return err
	}

	return nil
} // func (adv *Advisor) Unlearn(t *tag.Tag, i *feed.Item) error

type suggList []SuggestedTag

func (s suggList) Len() int           { return len(s) }
func (s suggList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s suggList) Less(i, j int) bool { return s[j].Score < s[i].Score }

// Suggest returns a map Tags and how likely they apply to the given Item.
func (adv *Advisor) Suggest(item *feed.Item, n int) map[string]SuggestedTag {
	var (
		err        error
		sugg       map[string]SuggestedTag
		res        map[string]float64
		lang, body string
		s          shield.Shield
	)

	lang, body = adv.getLanguage(item)

	if s = adv.shield[lang]; s == nil {
		s = adv.shield["en"]
	}

	if res, err = s.Score(body); err != nil {
		adv.log.Printf("[ERROR] Failed to Score Item %d (%q): %s\n",
			item.ID,
			item.Title,
			err.Error())
		return nil
	}

	var list = make(suggList, 0, len(res))

	for c, r := range res {
		var t = adv.tags[c]

		if t == nil || t.ID == 0 {
			adv.log.Printf("[CRITICAL] Invalid tag suggested for Item %q (%d):\n%#v\n",
				item.Title,
				item.ID,
				res)
			continue
		}

		var s = SuggestedTag{Tag: t, Score: r * 100}
		list = append(list, s)
	}

	sort.Sort(list)
	sugg = make(map[string]SuggestedTag, n)

	for _, s := range list[:n] {
		sugg[s.Tag.Name] = s
	}

	return sugg
} // func (adv *Advisor) Suggest(item *feed.Item) map[string]float64

// func (adv *Advisor) tokenize(item *feed.Item) []string {
// 	var (
// 		body, lang string
// 	)

// 	lang, body = adv.getLanguage(item)

// 	body = stopwords.CleanString(body, lang, true)

// 	var words = nonword.Split(body, -1)

// 	var tokens = make([]string, len(words))

// 	for i, w := range words {
// 		var s = stemWord(w, lang)
// 		tokens[i] = s
// 	}

// 	return tokens
// } // func (c *Advisor) tokenize(item *feed.Item) []string

func (adv *Advisor) getLanguage(item *feed.Item) (lng, fullText string) {
	const (
		defaultLang = "en"
	)

	var (
		err        error
		lang, body string
		blString   = []string{
			"Lauren Boebert buried in ridicule after claim about 1930s Germany",
			"GOP's Madison Cawthorn ruthlessly mocked for wailing about 'scary' proof of vaccination",
		}
	)

	body = item.Plaintext()

	defer func() {
		if x := recover(); x != nil {
			var m bool
			for _, bl := range blString {
				if strings.Contains(item.Title, bl) {
					m = true
					break
				}
			}
			if !m {
				var buf [2048]byte
				var cnt = runtime.Stack(buf[:], false)
				adv.log.Printf("[CRITICAL] Panic in getLanguage for Item %q: %s\n%s",
					item.Title,
					x,
					string(buf[:cnt]))
			}
			lng = defaultLang
			fullText = body
		}
	}()

	if lang, err = guesslanguage.Guess(body); err != nil {
		adv.log.Printf("[ERROR] Cannot determine language of Item %q: %s\n",
			item.Title,
			err.Error())
		lang = defaultLang
	}

	return lang, body
} // func getLanguage(title, description string) (string, string)

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
