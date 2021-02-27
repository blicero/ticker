-- Time-stamp: <2021-02-28 00:45:23 krylon>

SELECT
        id,
        [name],
        parent,
        ROW_NUMBER() OVER (
                     PARTITION BY parent
                     ORDER BY parent, id
                     ) AS rnk
FROM tag
ORDER BY COALESCE(parent, id);

