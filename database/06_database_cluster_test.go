// /home/krylon/go/src/ticker/database/06_database_cluster_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 16. 06. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-16 15:43:24 krylon>

package database

import (
	"math/rand"
	"testing"
	"ticker/cluster"
	"ticker/feed"
)

var clist = []cluster.Cluster{
	cluster.Cluster{
		Name:        "Cluster01",
		Description: "My first cluster",
	},
	cluster.Cluster{
		Name:        "Cluster02",
		Description: "My second cluster",
	},
}

func TestClusterCreate(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	for idx, cl := range clist {
		var (
			err error
			c   *cluster.Cluster
		)

		if c, err = db.ClusterCreate(cl.Name, cl.Description); err != nil {
			t.Fatalf("Failed to add cluster %s to Database: %s",
				cl.String(),
				err.Error())
		}

		clist[idx] = *c
	}
} // func TestClusterCreate(t *testing.T)

var linkedItems []feed.Item

func TestClusterLinkAdd(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	const (
		prob   = 0.65
		maxCnt = testItemCnt / 50
	)

	var (
		err   error
		items []feed.Item
		lcnt  int
	)

	if items, err = db.ItemGetAll(-1, 0); err != nil {
		t.Fatalf("Cannot load all Items: %s", err.Error())
	}

	linkedItems = make([]feed.Item, 0, maxCnt)

	for _, i := range items {
		if lcnt >= maxCnt || rand.Float64() >= prob {
			continue
		}

		if err = db.ClusterLinkAdd(clist[0].ID, i.ID); err != nil {
			t.Fatalf("Error adding Item %s (%d) to Cluster %s (%d): %s",
				i.Title,
				i.ID,
				clist[0].Name,
				clist[0].ID,
				err.Error())
		} else {
			linkedItems = append(linkedItems, i)
		}
	}
} // func TestClusterLinkAdd(t *testing.T)
