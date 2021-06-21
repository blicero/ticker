// /home/krylon/go/src/ticker/database/search.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 06. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-21 23:39:08 krylon>

// Package search implements the handling of search queries. Duh.
package search

import (
	"log"
	"regexp"
	"sort"
	"strings"
	"ticker/common"
	"ticker/database"
	"ticker/logdomain"
	"time"

	shlex "github.com/anmitsu/go-shlex"
)

// var tagPat = regexp.MustCompile(`(?i)tag:(\w+|"[^"]+")`)
var metaPat = regexp.MustCompile(`(?i)(\w+):(\S+|"[^"]+")`)

// Query represents a ... you guessed it: a search query.
// Using multiple
type Query struct {
	Tags      []string
	DateBegin time.Time
	DateEnd   time.Time
	Query     []string
	db        *database.Database
	log       *log.Logger
}

// ParseQueryStr parses a query string and returns a SearchQuery object.
func ParseQueryStr(d *database.Database, s string) (*Query, error) {
	var (
		err           error
		tokens, terms []string
		q             = &Query{db: d}
	)

	if q.log, err = common.GetLogger(logdomain.Search); err != nil {
		return nil, err
	} else if tokens, err = shlex.Split(s, true); err != nil {
		q.log.Printf("[ERROR] Cannot parse query string %q: %s\n",
			s,
			err.Error())
		return nil, err
	}

	terms = make([]string, 0, len(tokens))

	for _, t := range tokens {
		var (
			tword string
			match []string
		)

		if match = metaPat.FindStringSubmatch(t); match == nil {
			terms = append(terms, t)
			continue
		}

		switch strings.ToLower(match[1]) {
		case "tag":
			if match[2][0] == '"' {
				tword = match[2][1 : len(match[2])-1]
			} else {
				tword = match[2]
			}

			q.Tags = append(q.Tags, tword)

		case "datemin":
			tword = match[2]
			if tword[0] == '"' {
				tword = tword[1 : len(tword)-1]
			}

			if q.DateBegin, err = time.ParseInLocation(common.TimestampFormatDate, tword, time.Local); err != nil {
				q.log.Printf("[ERROR] Cannot parse date %q: %s\n",
					tword,
					err.Error())
				return nil, err
			}

		case "datemax":
			tword = match[2]
			if tword[0] == '"' {
				tword = tword[1 : len(tword)-1]
			}

			if q.DateEnd, err = time.ParseInLocation(common.TimestampFormatDate, tword, time.Local); err != nil {
				q.log.Printf("[ERROR] Cannot parse date %q: %s\n",
					tword,
					err.Error())
				return nil, err
			}

		default:
			q.log.Printf("[ERROR] Unknown metadata query %q will be ignored.\n",
				t)
		}
	}

	q.Query = terms

	sort.Strings(q.Query)
	sort.Strings(q.Tags)

	return q, nil
} // func ParseQueryStr(s string) (*SearchQuery, error)

// Equal returns true if the given SearchQuery is structurally identical to
// the receiver.
func (q *Query) Equal(other *Query) bool {
	if other == nil {
		return q == nil
	}

	if len(q.Tags) != len(other.Tags) || len(q.Query) != len(other.Query) {
		q.log.Printf("[TRACE] Tags: %d != %d || Query: %d != %d\n",
			len(q.Tags),
			len(other.Tags),
			len(q.Query),
			len(other.Query))
		return false
	} else if !(q.DateBegin.Equal(other.DateBegin) && q.DateEnd.Equal(other.DateEnd)) {
		q.log.Printf(`[TRACE] Dates differ:
Begin: %s
       %s
End:   %s
       %s
`,
			q.DateBegin.Format(common.TimestampFormatSubSecond),
			other.DateBegin.Format(common.TimestampFormatSubSecond),
			q.DateEnd.Format(common.TimestampFormatSubSecond),
			other.DateEnd.Format(common.TimestampFormatSubSecond))
		return false
	}

	for i, t := range q.Tags {
		if t != other.Tags[i] {
			q.log.Printf("[TRACE] Tag #%d differs: %q != %q\n",
				i,
				t,
				other.Tags[i])
			return false
		}
	}

	for i, t := range q.Query {
		if t != other.Query[i] {
			q.log.Printf("[TRACE] Query token %d differs: %q != %q\n",
				i,
				t,
				other.Query[i])
			return false
		}
	}

	return true
} // func (q *SearchQuery) Equal(other *SearchQuery) bool
