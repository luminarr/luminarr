-- name: InsertWatchEvent :exec
INSERT OR IGNORE INTO watch_history (id, movie_id, tmdb_id, watched_at, user_name, source)
VALUES (?, ?, ?, ?, ?, ?);

-- name: WatchStatusForMovie :one
SELECT
    COUNT(*) AS play_count,
    MAX(watched_at) AS last_watched_at,
    MIN(watched_at) AS first_watched_at
FROM watch_history
WHERE movie_id = ?;

-- name: WatchStatusBatch :many
SELECT
    movie_id,
    COUNT(*) AS play_count,
    MAX(watched_at) AS last_watched_at
FROM watch_history
GROUP BY movie_id;

-- name: WatchStats :one
SELECT
    (SELECT COUNT(DISTINCT movie_id) FROM watch_history) AS watched_count,
    (SELECT COUNT(*) FROM movies) AS total_count;

-- name: GetSyncState :one
SELECT last_sync_at FROM watch_sync_state WHERE media_server_id = ?;

-- name: UpsertSyncState :exec
INSERT INTO watch_sync_state (media_server_id, last_sync_at)
VALUES (?, ?)
ON CONFLICT(media_server_id) DO UPDATE SET last_sync_at = excluded.last_sync_at;

-- name: CleanupWatchedOnce :many
SELECT m.id, m.title, m.year,
       MAX(w.watched_at) AS last_watched,
       COALESCE(SUM(mf.size_bytes), 0) AS total_bytes
FROM movies m
JOIN watch_history w ON w.movie_id = m.id
LEFT JOIN movie_files mf ON mf.movie_id = m.id
GROUP BY m.id
HAVING COUNT(w.id) = 1 AND MAX(w.watched_at) < ?;

-- name: CleanupNeverWatched :many
SELECT m.id, m.title, m.year, m.added_at,
       COALESCE(SUM(mf.size_bytes), 0) AS total_bytes
FROM movies m
LEFT JOIN watch_history w ON w.movie_id = m.id
LEFT JOIN movie_files mf ON mf.movie_id = m.id
WHERE w.id IS NULL
  AND m.added_at < ?
GROUP BY m.id
HAVING total_bytes > 0;
