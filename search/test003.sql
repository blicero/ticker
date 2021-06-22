-- /home/krylon/go/src/ticker/search/test003.sql
-- created on 22. 06. 2021 by Benjamin Walkenhorst
-- (c) 2021 Benjamin Walkenhorst
-- Use at your own risk!

WITH mysearch AS (
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
WHERE item_index MATCH 'ibm OR linux OR openbsd'
ORDER BY i.timestamp DESC, i.title ASC
)

SELECT
        m.id,
        t.name,
        m.title
FROM mysearch m
INNER JOIN tag_link l ON m.id = l.item_id
INNER JOIN tag t ON l.tag_id = t.id
;
