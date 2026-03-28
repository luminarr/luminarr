-- name: InsertActivity :exec
INSERT INTO activity_log (id, type, category, movie_id, title, detail, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: ListActivities :many
SELECT * FROM activity_log
WHERE (sqlc.narg('category') IS NULL OR category = sqlc.narg('category'))
  AND (sqlc.narg('since') IS NULL OR created_at > sqlc.narg('since'))
ORDER BY created_at DESC
LIMIT sqlc.arg('limit');

-- name: CountActivities :one
SELECT COUNT(*) FROM activity_log
WHERE (sqlc.narg('category') IS NULL OR category = sqlc.narg('category'))
  AND (sqlc.narg('since') IS NULL OR created_at > sqlc.narg('since'));

-- name: PruneActivities :exec
DELETE FROM activity_log WHERE created_at < ?;
