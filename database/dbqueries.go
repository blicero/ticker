// /home/krylon/go/src/ticker/database/dbqueries.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-18 18:30:03 krylon>

package database

import "ticker/query"

var dbQueries = map[query.ID]string{
	query.FeedAdd: `
INSERT INTO feed (name, url, homepage, refresh_interval) 
VALUES           (   ?,   ?,        ?,                ?)
`,
	query.FeedGetAll: `
SELECT
     id,
     name,
     url,
     homepage,
     refresh_interval,
     refresh_timestamp,
     active
FROM feed
`,
	query.FeedGetDue: `
SELECT
     id,
     name,
     url,
     homepage,
     refresh_interval,
     refresh_timestamp,
     active
FROM feed
WHERE refresh_timestamp + refresh_interval < ?
`,
	query.FeedGetByID: `
SELECT
     name,
     url,
     homepage,
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
	query.FeedDelete: "DELETE FROM feed WHERE id = ?",
	query.ItemAdd: `
INSERT INTO item (feed_id, link, title, description, timestamp)
VALUES           (      ?,    ?,     ?,           ?,         ?)
`,
	query.ItemGetRecent: `
SELECT
    id,
    feed_id,
    link,
    title,
    description,
    timestamp,
    read,
    rating
FROM item
ORDER BY timestamp DESC
LIMIT ?
`,

	query.ItemGetRated: `
SELECT
    id,
    feed_id,
    link,
    title,
    description,
    timestamp,
    read,
    rating
FROM item
WHERE rating IS NOT NULL
`,
	query.ItemGetByID: `
SELECT
    feed_id,
    link,
    title,
    description,
    timestamp,
    read,
    rating
FROM item
WHERE id = ?
`,
	query.ItemGetByURL: `
SELECT
    id,
    feed_id,
    title,
    description,
    timestamp,
    read,
    rating
FROM item
WHERE link = ?
`,
	query.ItemGetByFeed: `
SELECT
    id,
    link,
    title,
    description,
    timestamp,
    read,
    rating
FROM item
WHERE feed_id = ?
ORDER BY timestamp DESC
LIMIT ?
`,
	query.ItemGetAll: `
SELECT
    id,
    feed_id,
    link,
    title,
    description,
    timestamp,
    read,
    rating
FROM item
ORDER BY timestamp DESC
LIMIT ? OFFSET ?
`,
	query.ItemRatingSet:   "UPDATE item SET rating = ? WHERE id = ?",
	query.ItemRatingClear: "UPDATE item SET rating = NULL WHERE id = ?",
}
