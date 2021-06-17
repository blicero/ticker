// /home/krylon/go/src/ticker/database/initqueries.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 02. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-06-16 12:07:25 krylon>

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
    prefetch            INTEGER NOT NULL DEFAULT 0,
    
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

	`
CREATE TABLE tag (
    id		INTEGER PRIMARY KEY,
    name	TEXT UNIQUE NOT NULL,
    parent      INTEGER,
    description TEXT,
    FOREIGN KEY (parent) REFERENCES tag (id)
         ON DELETE RESTRICT
         ON UPDATE RESTRICT
)
`,

	`
CREATE TABLE tag_link (
    id		INTEGER PRIMARY KEY,
    tag_id	INTEGER NOT NULL,
    item_id	INTEGER NOT NULL,
    CONSTRAINT tag_item_uniq UNIQUE (tag_id, item_id),
    FOREIGN KEY (tag_id) REFERENCES tag (id)
        ON DELETE CASCADE
        ON UPDATE RESTRICT,
    FOREIGN KEY (item_id) REFERENCES item (id)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
)
`,

	`
CREATE TABLE read_later (
    id		INTEGER PRIMARY KEY,
    item_id	INTEGER UNIQUE NOT NULL,
    note        TEXT,
    timestamp   INTEGER NOT NULL,
    deadline	INTEGER,
    read	INTEGER,
    FOREIGN KEY (item_id) REFERENCES item (id)
        ON DELETE CASCADE
	ON UPDATE RESTRICT
)
`,

	`
CREATE TABLE cluster (
    id			INTEGER PRIMARY KEY,
    name		TEXT UNIQUE NOT NULL,
    description		TEXT NOT NULL DEFAULT '',
    timestamp		INTEGER NOT NULL
)
`,

	`
CREATE TABLE cluster_link (
    id INTEGER PRIMARY KEY,
    cluster_id INTEGER NOT NULL,
    item_id INTEGER NOT NULL,
    CONSTRAINT cluster_item_link_uniq UNIQUE (cluster_id, item_id),
    FOREIGN KEY (cluster_id) REFERENCES cluster (id)
        ON DELETE CASCADE
        ON UPDATE RESTRICT,
    FOREIGN KEY (item_id) REFERENCES item (id)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
)
`,

	`
CREATE TRIGGER tr_clu_lnk_add
AFTER INSERT ON cluster_link
BEGIN
    UPDATE cluster 
    SET timestamp = CAST(strftime('%s', 'now') AS INTEGER)
    WHERE id = new.cluster_id;
END;
`,

	`
CREATE TRIGGER tr_clu_lnk_del
AFTER DELETE ON cluster_link
BEGIN
    UPDATE cluster 
    SET timestamp = CAST(strftime('%s', 'now') AS INTEGER)
    WHERE id = old.cluster_id;
END;
`,
}
