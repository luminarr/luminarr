-- name: InsertStorageSnapshot :exec
INSERT INTO storage_snapshots (id, captured_at, total_bytes, file_count)
VALUES (?, ?, ?, ?);

-- name: ListStorageSnapshots :many
SELECT * FROM storage_snapshots
ORDER BY captured_at DESC
LIMIT ?;

-- name: PruneOldStorageSnapshots :exec
DELETE FROM storage_snapshots
WHERE captured_at < ?;

-- name: GetCollectionStats :one
SELECT
    COUNT(*)                                                                    AS total_movies,
    SUM(CASE WHEN monitored = 1 THEN 1 ELSE 0 END)                             AS monitored,
    SUM(CASE WHEN path IS NOT NULL AND path != '' THEN 1 ELSE 0 END)           AS with_file,
    SUM(CASE WHEN monitored = 1 AND (path IS NULL OR path = '') THEN 1 ELSE 0 END) AS missing,
    SUM(CASE WHEN added_at > datetime('now', '-30 days') THEN 1 ELSE 0 END)    AS recently_added
FROM movies;

-- name: GetStorageTotals :one
SELECT
    COALESCE(SUM(size_bytes), 0) AS total_bytes,
    COUNT(*)                      AS file_count
FROM movie_files;

-- name: ListMovieFileQualities :many
SELECT quality_json FROM movie_files;

-- name: GetGrabStats :one
SELECT
    COUNT(*)                                                                    AS total_grabs,
    SUM(CASE WHEN download_status = 'completed' THEN 1 ELSE 0 END)             AS successful,
    SUM(CASE WHEN download_status = 'failed' THEN 1 ELSE 0 END)                AS failed
FROM grab_history;

-- name: GetTopIndexers :many
SELECT
    gh.indexer_id,
    COALESCE(ic.name, gh.indexer_id)                                            AS indexer_name,
    COUNT(*)                                                                    AS grab_count,
    SUM(CASE WHEN gh.download_status = 'completed' THEN 1 ELSE 0 END)          AS success_count
FROM grab_history gh
LEFT JOIN indexer_configs ic ON ic.id = gh.indexer_id
WHERE gh.indexer_id IS NOT NULL AND gh.indexer_id != ''
GROUP BY gh.indexer_id
ORDER BY grab_count DESC
LIMIT 10;

-- name: GetMovieYearDistribution :many
SELECT year, COUNT(*) AS count
FROM movies
WHERE year > 0
GROUP BY year
ORDER BY year ASC;

-- name: GetMoviesAddedByMonth :many
SELECT
    strftime('%Y-%m', added_at) AS month,
    COUNT(*)                     AS count
FROM movies
WHERE added_at IS NOT NULL
GROUP BY month
ORDER BY month ASC;

-- name: ListMovieGenresJSON :many
SELECT genres_json FROM movies WHERE genres_json IS NOT NULL AND genres_json != '[]';
