-- Time-stamp: <2021-03-10 10:33:40 krylon>

WITH RECURSIVE children(id, name, description, parent) AS (
    SELECT
        id,
        name,
        description,
        parent
    FROM tag WHERE id = 1
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
        f.name,
        i.title
FROM children c
INNER JOIN tag_link l ON c.id = l.tag_id
INNER JOIN item i ON l.item_id = i.id
INNER JOIN feed f ON i.feed_id = f.id


-- SELECT
--     id,
--     name, 
--     description,
--     parent
-- FROM children
-- WHERE id <> ?
-- ORDER BY name
