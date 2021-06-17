// /home/krylon/go/src/ticker/cluster/cluster.go
// -*- mode: go; coding: utf-8; -*-
// Created on 16. 06. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-16 11:49:59 krylon>

// Package cluster provides the ability to group several Items together.
package cluster

import (
	"fmt"
	"ticker/common"
	"ticker/feed"
	"time"
)

// Cluster represents a group of Items
type Cluster struct {
	ID          int64
	Name        string
	Timestamp   time.Time
	Description string
	Items       []feed.Item
}

func (c *Cluster) String() string {
	return fmt.Sprintf("Cluster{ ID: %d, Name: %q, Timestamp: %q}",
		c.ID,
		c.Name,
		c.Timestamp.Format(common.TimestampFormat))
} // func (c *Cluster) String() string
