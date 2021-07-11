// /home/krylon/go/src/ticker/web/tmpl_data.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 05. 2020 by Benjamin Walkenhorst
// (c) 2020 Benjamin Walkenhorst
// Time-stamp: <2021-07-11 21:21:04 krylon>
//
// This file contains data structures to be passed to HTML templates.

package web

import (
	"crypto/sha512"
	"fmt"
	"ticker/advisor"
	"ticker/cluster"
	"ticker/common"
	"ticker/feed"
	"ticker/tag"
	"time"

	"github.com/hashicorp/logutils"
)

type message struct {
	Timestamp time.Time
	Level     logutils.LogLevel
	Message   string
}

func (m *message) TimeString() string {
	return m.Timestamp.Format(common.TimestampFormat)
} // func (m *Message) TimeString() string

func (m *message) Checksum() string {
	var str = m.Timestamp.Format(common.TimestampFormat) + "##" +
		string(m.Level) + "##" +
		m.Message

	var hash = sha512.New()
	hash.Write([]byte(str)) // nolint: gosec,errcheck

	var cksum = hash.Sum(nil)
	var ckstr = fmt.Sprintf("%x", cksum)

	return ckstr
} // func (m *message) Checksum() string

type tmplDataBase struct {
	Title          string
	Messages       []message
	Debug          bool
	TestMsgGen     bool
	URL            string
	TrainStamp     time.Time
	AllTags        []tag.Tag
	AllClusters    []cluster.Cluster
	TagHierarchy   []tag.Tag
	TagSuggestions map[int64]map[string]advisor.SuggestedTag
}

// TagLinkData returns data for use in the tag_link_form template.
func (t *tmplDataBase) TagLinkData(i feed.Item) *tmplDataTagLinkData {
	return &tmplDataTagLinkData{
		Item: i,
		Tags: t.AllTags,
	}
} // func (t *tmplDataBase) TagLinkData() *tmplDataTagLinkData

type tmplDataIndex struct {
	tmplDataBase
	FeedMap  map[int64]feed.Feed
	Feeds    []feed.Feed
	Items    []feed.Item
	Clusters map[int64][]cluster.Cluster
}

type tmplDataItems struct {
	tmplDataBase
	Items    []feed.Item
	FeedMap  map[int64]feed.Feed
	Clusters map[int64][]cluster.Cluster
	Next     string
	Prev     string
	PageCnt  int64
}

type tmplDataClusterForm struct {
	Item        feed.Item
	Clusters    []cluster.Cluster
	AllClusters []cluster.Cluster
}

// ClusterFormData returns the data required to populate the Cluster form for a specific Item.
func (t *tmplDataItems) ClusterFormData(id int64) *tmplDataClusterForm {
	var data = &tmplDataClusterForm{
		Clusters:    t.Clusters[id],
		AllClusters: t.AllClusters,
	}

	// ???
	for _, i := range t.Items {
		if i.ID == id {
			data.Item = i
			return data
		}
	}

	return nil
} // func (t *tmplDataItems) ClusterFormData(id int64) *tmplDataClusterForm

// TagLinkData returns data for use in the tag_link_form template.
func (t *tmplDataItems) TagLinkData(i feed.Item) *tmplDataTagLinkData {
	return &tmplDataTagLinkData{
		Item: i,
		Tags: t.AllTags,
	}
} // func (t *tmplDataItems) TagLinkData() *tmplDataTagLinkData

type tmplDataTagLinkData struct {
	Item feed.Item
	Tags []tag.Tag
}

type tmplDataTags struct {
	tmplDataBase
	Tags        []tag.Tag
	TaggedItems map[int64][]feed.Item
}

type tmplDataTagDetails struct {
	tmplDataBase
	Tag      *tag.Tag
	Items    []feed.Item
	Children []tag.Tag
	FeedMap  map[int64]feed.Feed
}

type tmplDataReadLater struct {
	tmplDataBase
	Items   []feed.ReadLater
	FeedMap map[int64]feed.Feed
}

type tmplDataClusterList struct {
	tmplDataBase
	Clusters []cluster.Cluster
}

type tmplDataClusterItems struct {
	tmplDataItems
	Cluster *cluster.Cluster
}

// Local Variables:  //
// compile-command: "go generate && go vet && go build -v -p 16 && gometalinter && go test -v" //
// End: //
