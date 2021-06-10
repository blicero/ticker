-- Time-stamp: <2021-06-10 16:54:42 krylon>
--
-- Montag, 01. 03. 2021, 09:41
-- I am convinced there is a way to do this in pure SQL, but the problem
-- is not important enough to keep banging my head against the wall, so
-- for now I put this one aside.

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
    -- id,
    -- name,
    -- lvl,
    -- root,
    -- parent,
    -- full_name
FROM children
ORDER BY full_name;
