-- Time-stamp: <2021-03-18 18:40:00 krylon>

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
WHERE item_index MATCH 'biden OR china'
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
INNER JOIN tag t ON t.id = l.tag_id
WHERE t.id IN (2, 8, 11)
ORDER BY i.feed_id, i.id, t.name
