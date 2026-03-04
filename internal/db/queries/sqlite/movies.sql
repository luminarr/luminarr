-- name: CreateMovie :one
INSERT INTO movies (
    id, tmdb_id, imdb_id, title, original_title,
    year, overview, runtime_minutes, genres_json,
    poster_url, fanart_url, status, monitored,
    library_id, quality_profile_id, path,
    added_at, updated_at, metadata_refreshed_at,
    minimum_availability, release_date
) VALUES (
    ?, ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?,
    ?, ?
)
RETURNING *;

-- name: GetMovie :one
SELECT * FROM movies WHERE id = ?;

-- name: GetMovieByTMDBID :one
SELECT * FROM movies WHERE tmdb_id = ?;

-- name: ListMovies :many
SELECT * FROM movies
ORDER BY title ASC
LIMIT ? OFFSET ?;

-- name: ListMoviesByLibrary :many
SELECT * FROM movies
WHERE library_id = ?
ORDER BY title ASC
LIMIT ? OFFSET ?;

-- name: ListMonitoredMovies :many
SELECT * FROM movies
WHERE monitored = 1
ORDER BY title ASC;

-- name: CountMovies :one
SELECT COUNT(*) FROM movies;

-- name: CountMoviesByLibrary :one
SELECT COUNT(*) FROM movies WHERE library_id = ?;

-- name: UpdateMovie :one
UPDATE movies SET
    title                = ?,
    original_title       = ?,
    year                 = ?,
    overview             = ?,
    runtime_minutes      = ?,
    genres_json          = ?,
    poster_url           = ?,
    fanart_url           = ?,
    status               = ?,
    monitored            = ?,
    library_id           = ?,
    quality_profile_id   = ?,
    minimum_availability = ?,
    release_date         = ?,
    updated_at           = ?
WHERE id = ?
RETURNING *;

-- name: UpdateMovieTMDBID :exec
UPDATE movies SET tmdb_id = ?, updated_at = ? WHERE id = ?;

-- name: UpdateMovieStatus :one
UPDATE movies SET status = ?, updated_at = ? WHERE id = ? RETURNING *;

-- name: UpdateMoviePath :one
UPDATE movies SET path = ?, updated_at = ? WHERE id = ? RETURNING *;

-- name: UpdateMovieMetadataRefreshed :exec
UPDATE movies SET metadata_refreshed_at = ?, updated_at = ? WHERE id = ?;

-- name: DeleteMovie :exec
DELETE FROM movies WHERE id = ?;

-- name: CreateMovieFile :one
INSERT INTO movie_files (
    id, movie_id, path, size_bytes, quality_json,
    edition, imported_at, indexed_at
) VALUES (
    ?, ?, ?, ?, ?,
    ?, ?, ?
)
RETURNING *;

-- name: GetMovieFile :one
SELECT * FROM movie_files WHERE id = ?;

-- name: ListMovieFiles :many
SELECT * FROM movie_files WHERE movie_id = ? ORDER BY imported_at DESC;

-- name: UpdateMovieFileIndexed :exec
UPDATE movie_files SET indexed_at = ? WHERE id = ?;

-- name: UpdateMovieFilePath :exec
UPDATE movie_files SET path = ? WHERE id = ?;

-- name: DeleteMovieFile :exec
DELETE FROM movie_files WHERE id = ?;

-- name: SumMovieFileSizesByLibrary :one
SELECT COALESCE(SUM(mf.size_bytes), 0)
FROM movie_files mf
JOIN movies m ON m.id = mf.movie_id
WHERE m.library_id = ?;

-- name: ListMovieFilesByLibrary :many
SELECT mf.*
FROM movie_files mf
JOIN movies m ON m.id = mf.movie_id
WHERE m.library_id = ?
ORDER BY mf.path ASC;

-- name: GetMovieFileByPath :one
SELECT * FROM movie_files WHERE path = ?;

-- name: ListMonitoredMoviesWithoutFile :many
SELECT m.*
FROM movies m
LEFT JOIN movie_files mf ON mf.movie_id = m.id
WHERE m.monitored = 1
  AND mf.id IS NULL
ORDER BY m.title ASC
LIMIT ? OFFSET ?;

-- name: CountMonitoredMoviesWithoutFile :one
SELECT COUNT(*)
FROM movies m
LEFT JOIN movie_files mf ON mf.movie_id = m.id
WHERE m.monitored = 1
  AND mf.id IS NULL;

-- name: ListMonitoredMoviesWithFiles :many
SELECT m.*, mf.quality_json, qp.cutoff_json
FROM movies m
JOIN movie_files mf ON mf.movie_id = m.id
JOIN quality_profiles qp ON qp.id = m.quality_profile_id
WHERE m.monitored = 1
ORDER BY m.title ASC;
