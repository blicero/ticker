// /home/krylon/go/src/ticker/database/dbqueries.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-24 20:22:13 krylon>

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
	query.ItemInsertFTS: "INSERT INTO item_index (link, body) VALUES (?, ?)",
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
	query.ItemGetFTS: `
SELECT
    i.id,
    i.feed_id,
    i.link,
    i.title,
    i.description,
    i.timestamp,
    i.read,
    i.rating
FROM item_index x
INNER JOIN item i ON x.link = i.link
WHERE item_index MATCH ?
ORDER BY i.timestamp DESC, i.title ASC
`,
	query.ItemGetContent: `
SELECT
    link,
    title || ' ' || description AS body
FROM item
`,
	// TODO As Tags can form a hierarchy, I would really like this query to
	//      also return Items that are linked with Tags that are *children*
	//      of the given Tag.
	//      That will most likely involve a recursive CTE, and I think
	//      I have done something vaguely like it before, but seeing as how
	//      I am a lazy person, I will postpone that until the rest of the
	//      Tag related stuff is working.
	query.ItemGetByTag: `
SELECT
    i.id,
    i.feed_id,
    i.link,
    i.title,
    i.description,
    i.timestamp,
    i.read,
    i.rating
FROM tag_link t
INNER JOIN item i ON t.item_id = i.id
WHERE t.tag_id = ?
`,
	query.ItemRatingSet:        "UPDATE item SET rating = ? WHERE id = ?",
	query.ItemRatingClear:      "UPDATE item SET rating = NULL WHERE id = ?",
	query.FTSClear:             "DELETE FROM item_index",
	query.TagCreate:            "INSERT INTO tag (name, description, parent) VALUES (?, ?, ?)",
	query.TagDelete:            "DELETE FROM tag WHERE id = ?",
	query.TagNameUpdate:        "UPDATE tag SET name = ? WHERE id = ?",
	query.TagDescriptionUpdate: "UPDATE tag SET description = ? WHERE id = ?",
	query.TagParentSet:         "UPDATE tag SET parent = ? WHERE id = ?",
	query.TagParentClear:       "UPDATE tag SET parent = NULL WHERE id = ?",
	query.TagLinkCreate:        "INSERT INTO tag_link (tag_id, item_id) VALUES (?, ?)",
	query.TagLinkDelete:        "DELETE FROM tag_link WHERE tag_id = ? AND item_id = ?",
	query.TagLinkGetByItem:     "SELECT tag_id FROM tag_link WHERE item_id = ?",
}
