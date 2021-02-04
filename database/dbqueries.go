// /home/krylon/go/src/ticker/database/dbqueries.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-04 09:20:20 krylon>

package database

import "ticker/query"

var dbQueries = map[query.ID]string{
	query.FeedAdd: `
INSERT INTO feed (name, url, refresh_interval) 
VALUES           (   ?,   ?,                ?)
`,
	query.FeedGetAll: `
SELECT
     id,
     name,
     url,
     refresh_interval,
     refresh_timestamp,
     active
FROM feed
`,
	query.FeedGetByID: `
SELECT
     name,
     url,
     refresh_interval,
     refresh_timestamp,
     active
FROM feed
WHERE id = ?
`,
	query.FeedSetTimestamp: `
UPDATE feed
SET refresh_timestamp = ?
WHERE id = ?
`,
}
