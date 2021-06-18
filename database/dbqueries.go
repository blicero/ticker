// /home/krylon/go/src/ticker/database/dbqueries.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-18 15:10:29 krylon>

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
WHERE active = 1 AND refresh_timestamp + refresh_interval < ?
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
	query.FeedSetActive: "UPDATE feed SET active = ? WHERE id = ?",
	query.FeedSetTimestamp: `
UPDATE feed
SET refresh_timestamp = ?
WHERE id = ?
`,
	query.FeedDelete: "DELETE FROM feed WHERE id = ?",
	query.FeedModify: `
UPDATE feed SET 
    name		= ?, 
    url			= ?, 
    homepage		= ?, 
    refresh_interval	= ?
WHERE id = ?
`,
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
	query.ItemGetSearchExtended: `
WITH items1 AS (
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
)

SELECT DISTINCT
        i.id,
        i.feed_id,
        i.link,
        i.title,
        i.description,
        i.timestamp,
        i.read,
        i.rating
FROM tag_link l
INNER JOIN items1 i ON l.item_id = i.id
WHERE i.timestamp BETWEEN ? AND ?
ORDER BY i.timestamp DESC
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
ORDER BY i.timestamp DESC
`,
	query.ItemGetByTagRecursive: `
WITH RECURSIVE children(id, name, description, parent) AS (
    SELECT
        id,
        name,
        description,
        parent
    FROM tag WHERE id = ?
    UNION ALL
    SELECT
        tag.id,
        tag.name,
        tag.description,
        tag.parent
    FROM tag, children
    WHERE tag.parent = children.id
)

SELECT
    i.id,
    i.feed_id,
    i.link,
    i.title,
    i.description,
    i.timestamp,
    i.read,
    i.rating
FROM children c
INNER JOIN tag_link l ON c.id = l.tag_id
INNER JOIN item i ON l.item_id = i.id
INNER JOIN feed f ON i.feed_id = f.id
ORDER BY i.timestamp DESC
`,
	query.ItemGetPrefetch: `
SELECT id,
       feed_id,
       link,
       title,
       description,
       timestamp,
       read,
       rating
FROM item
WHERE prefetch <> 1
ORDER BY timestamp DESC
LIMIT ?
`,
	query.ItemGetTotalCnt: "SELECT COUNT(id) FROM item",
	query.ItemPrefetchSet: "UPDATE item SET description = ?, prefetch = 1 WHERE id = ?",
	query.ItemRatingSet:   "UPDATE item SET rating = ? WHERE id = ?",
	query.ItemRatingClear: "UPDATE item SET rating = NULL WHERE id = ?",
	query.ItemHasDuplicate: `
SELECT
    COUNT(id) AS cnt
FROM item
WHERE link = ?
   OR (feed_id = ? AND title = ?)
`,
	query.FTSClear:     "DELETE FROM item_index",
	query.TagCreate:    "INSERT INTO tag (name, description, parent) VALUES (?, ?, ?)",
	query.TagDelete:    "DELETE FROM tag WHERE id = ?",
	query.TagGetAll:    "SELECT id, name, description, parent FROM tag ORDER BY name",
	query.TagGetByID:   "SELECT name, description, parent FROM tag WHERE id = ?",
	query.TagGetByName: "SELECT id, description, parent FROM tag WHERE name = ?",
	query.TagGetByItem: `
SELECT
    t.id,
    t.name,
    t.description,
    t.parent
FROM tag_link l
INNER JOIN tag t ON l.tag_id = t.id
WHERE l.item_id = ?
`,
	query.TagGetChildren: `
WITH RECURSIVE children(id, name, description, parent) AS (
    SELECT
        id,
        name,
        description,
        parent
    FROM tag WHERE id = ?
    UNION ALL
    SELECT
        tag.id,
        tag.name,
        tag.description,
        tag.parent
    FROM tag, children
    WHERE tag.parent = children.id
)

SELECT
    id,
    name, 
    description,
    parent
FROM children
WHERE id <> ?
ORDER BY name
`,
	query.TagGetAllByHierarchy: `
WITH RECURSIVE children(id, name, description, lvl, root, parent, full_name) AS (
    SELECT
        id,
        name,
        description,
        0 AS lvl,
        id AS root,
        COALESCE(parent, 0) AS parent,
        name AS full_name
    FROM tag WHERE parent IS NULL
    UNION ALL
    SELECT
        tag.id,
        tag.name,
        tag.description,
        lvl + 1 AS lvl,
        children.root,
        tag.parent,
        full_name || '/' || tag.name AS full_name
    FROM tag, children
    WHERE tag.parent = children.id
)

SELECT
        id,
        name,
        description,
        parent,
        lvl,
        full_name
FROM children
ORDER BY full_name;
`,
	query.TagGetChildrenImmediate: `
SELECT
    id,
    name,
    description
FROM tag
WHERE parent = ?
ORDER BY name
`,
	query.TagGetRoots: `
SELECT
    id,
    name,
    description
FROM tag
WHERE COALESCE(parent, 0) = 0
ORDER BY name
`,
	query.TagNameUpdate:        "UPDATE tag SET name = ? WHERE id = ?",
	query.TagDescriptionUpdate: "UPDATE tag SET description = ? WHERE id = ?",
	query.TagParentSet:         "UPDATE tag SET parent = ? WHERE id = ?",
	query.TagParentClear:       "UPDATE tag SET parent = NULL WHERE id = ?",
	query.TagUpdate:            "UPDATE tag SET name = ?, parent = ?, description = ? WHERE id = ?",
	query.TagLinkCreate:        "INSERT INTO tag_link (tag_id, item_id) VALUES (?, ?)",
	query.TagLinkDelete:        "DELETE FROM tag_link WHERE tag_id = ? AND item_id = ?",
	query.TagLinkGetByItem:     "SELECT tag_id FROM tag_link WHERE item_id = ?",

	query.ReadLaterAdd: `
INSERT INTO read_later (item_id, note, timestamp, deadline)
                VALUES (      ?,    ?,         ?,        ?)
`,
	query.ReadLaterGetByItem: `
SELECT
    id,
    note,
    timestamp,
    deadline,
    read
FROM read_later
WHERE item_id = ?
`,
	query.ReadLaterGetAll: `
SELECT
    l.id,
    l.item_id,
    l.note,
    l.timestamp,
    l.deadline,
    l.read,
    i.feed_id,
    i.link,
    i.title,
    i.description,
    i.timestamp,
    i.read,
    i.rating
FROM read_later l
INNER JOIN item i ON i.id = l.item_id
ORDER BY l.deadline DESC
`,
	query.ReadLaterGetUnread: `
SELECT
    l.id,
    l.item_id,
    l.note,
    l.timestamp,
    l.deadline,
    i.feed_id,
    i.link,
    i.title,
    i.description,
    i.timestamp,
    i.read,
    i.rating
FROM read_later l
INNER JOIN item i ON l.item_id = i.id
WHERE l.read <> 1
ORDER BY l.deadline DESC
`,
	query.ReadLaterMarkRead: `
UPDATE read_later
SET read = 1
WHERE item_id = ?
`,
	query.ReadLaterMarkUnread: `
UPDATE read_later
SET read = 0
WHERE item_id = ?
`,
	query.ReadLaterDelete:     "DELETE FROM read_later WHERE item_id = ?",
	query.ReadLaterDeleteRead: "DELETE FROM read_later WHERE COALESCE(read, 0) <> 0",
	query.ReadLaterSetDeadine: "UPDATE read_later SET deadline = ? WHERE item_id = ?",
	query.ReadLaterSetNote:    "UPDATE read_later SET note = ? WHERE item_id = ?",

	query.ClusterCreate: `
INSERT INTO cluster (name, description, timestamp) 
             VALUES (?,    ?,           CAST(strftime('%s', 'now') AS INTEGER))`,
	query.ClusterDelete:            "DELETE FROM cluster WHERE id = ?",
	query.ClusterNameUpdate:        "UPDATE cluster SET name = ? WHERE id = ?",
	query.ClusterDescriptionUpdate: "UPDATE cluster SET description = ? WHERE id = ?",
	query.ClusterGetByID:           "SELECT name, description, timestamp FROM cluster WHERE id = ?",
	query.ClusterGetByName:         "SELECT id, description, timestamp FROM cluster WHERE name = ?",
	query.ClusterGetByItem: `
SELECT
    l.cluster_id,
    c.name,
    c.description,
    c.timestamp
FROM cluster_link l
INNER JOIN cluster c ON l.cluster_id = c.id
WHERE l.item_id = ?
`,
	query.ClusterGetItems: `
SELECT
    i.id,
    i.feed_id,
    i.link,
    i.title,
    i.description,
    i.timestamp,
    i.read,
    i.rating
FROM cluster c
INNER JOIN cluster_link l ON c.id = l.cluster_id
INNER JOIN item i ON l.item_id = i.id
WHERE c.id = ?
`,
	query.ClusterGetAll:           "SELECT id, name, description, timestamp FROM cluster",
	query.ClusterLinkAdd:          "INSERT INTO cluster_link (cluster_id, item_id) VALUES (?, ?)",
	query.ClusterLinkDel:          "DELETE FROM cluster_link WHERE cluster_id = ? AND item_id = ?",
	query.ClusterLinkGetByItem:    "SELECT id, cluster_id FROM cluster_link WHERE item_id = ?",
	query.ClusterLinkGetByCluster: "SELECT id, item_id FROM cluster_link WHERE cluster_id = ?",
}
