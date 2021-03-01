-- Time-stamp: <2021-03-01 19:42:23 krylon>
--
-- Montag, 01. 03. 2021, 09:41
-- I am convinced there is a way to do this in pure SQL, but the problem
-- is not important enough to keep banging my head against the wall, so
-- for now I put this one aside.

WITH RECURSIVE children(id, name, description, lvl, root, parent) AS (
    SELECT
        id,
        name,
        description,
        0 AS lvl,
        id AS root,
        COALESCE(parent, 0) AS parent
    FROM tag WHERE parent IS NULL
    UNION ALL
    SELECT
        tag.id,
        tag.name,
        tag.description,
        lvl + 1 AS lvl,
        children.root,
        tag.parent
    FROM tag, children
    WHERE tag.parent = children.id
)

SELECT
    id,
    name,
    -- description,
    lvl,
    root,
    parent,
    RANK() OVER (
        PARTITION BY root
        ORDER BY lvl
    ) AS aux
FROM children
ORDER BY root, parent;
