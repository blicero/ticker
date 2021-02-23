// /home/krylon/go/src/ticker/database/initqueries.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-02-23 21:05:40 krylon>

package database

var initQueries = []string{
	`
CREATE TABLE feed (
    id			INTEGER PRIMARY KEY,
    name		TEXT NOT NULL,
    url  		TEXT UNIQUE NOT NULL,
    homepage            TEXT NOT NULL,
    refresh_interval    INTEGER NOT NULL,
    refresh_timestamp   INTEGER NOT NULL DEFAULT 0,
    active              INTEGER NOT NULL DEFAULT 1,

    CONSTRAINT interval_positive CHECK (refresh_interval > 0)
)
`,

	`
CREATE TABLE item (
    id			INTEGER PRIMARY KEY,
    feed_id		INTEGER NOT NULL,
    link		TEXT NOT NULL,
    title               TEXT NOT NULL,
    description         TEXT NOT NULL,
    timestamp           INTEGER NOT NULL,
    read                INTEGER NOT NULL DEFAULT 0,
    rating              REAL,
    
    CHECK (rating IS NULL OR (rating BETWEEN 0.0 AND 1.0)),
    CONSTRAINT feed_link_uniq UNIQUE (feed_id, link),
    FOREIGN KEY (feed_id) REFERENCES feed (id)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
)
`,

	"CREATE VIRTUAL TABLE item_index USING fts4(link, body)",

	`
CREATE TRIGGER tr_item_fts_insert
AFTER INSERT ON item
BEGIN
    INSERT INTO item_index (link, body) VALUES (new.link, new.title || ' ' || new.description);
END;
`,
	`
CREATE TRIGGER tr_item_fts_delete
AFTER DELETE ON item
BEGIN
    DELETE FROM item_index
    WHERE item_index.link = old.link;
END;
`,
}
