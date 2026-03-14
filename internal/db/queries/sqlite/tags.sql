-- name: CreateTag :one
INSERT INTO tags (id, name) VALUES (?, ?) RETURNING *;

-- name: GetTag :one
SELECT * FROM tags WHERE id = ?;

-- name: GetTagByName :one
SELECT * FROM tags WHERE name = ?;

-- name: ListTags :many
SELECT * FROM tags ORDER BY name ASC;

-- name: UpdateTag :one
UPDATE tags SET name = ? WHERE id = ? RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = ?;

-- Tag counts per entity type (for usage display).

-- name: CountMoviesForTag :one
SELECT COUNT(*) FROM movie_tags WHERE tag_id = ?;

-- name: CountIndexersForTag :one
SELECT COUNT(*) FROM indexer_tags WHERE tag_id = ?;

-- name: CountDownloadClientsForTag :one
SELECT COUNT(*) FROM download_client_tags WHERE tag_id = ?;

-- name: CountNotificationsForTag :one
SELECT COUNT(*) FROM notification_tags WHERE tag_id = ?;

-- Movie tag operations.

-- name: SetMovieTags :exec
DELETE FROM movie_tags WHERE movie_id = ?;

-- name: AddMovieTag :exec
INSERT OR IGNORE INTO movie_tags (movie_id, tag_id) VALUES (?, ?);

-- name: ListMovieTagIDs :many
SELECT tag_id FROM movie_tags WHERE movie_id = ?;

-- Indexer tag operations.

-- name: SetIndexerTags :exec
DELETE FROM indexer_tags WHERE indexer_id = ?;

-- name: AddIndexerTag :exec
INSERT OR IGNORE INTO indexer_tags (indexer_id, tag_id) VALUES (?, ?);

-- name: ListIndexerTagIDs :many
SELECT tag_id FROM indexer_tags WHERE indexer_id = ?;

-- Download client tag operations.

-- name: SetDownloadClientTags :exec
DELETE FROM download_client_tags WHERE download_client_id = ?;

-- name: AddDownloadClientTag :exec
INSERT OR IGNORE INTO download_client_tags (download_client_id, tag_id) VALUES (?, ?);

-- name: ListDownloadClientTagIDs :many
SELECT tag_id FROM download_client_tags WHERE download_client_id = ?;

-- Notification tag operations.

-- name: SetNotificationTags :exec
DELETE FROM notification_tags WHERE notification_id = ?;

-- name: AddNotificationTag :exec
INSERT OR IGNORE INTO notification_tags (notification_id, tag_id) VALUES (?, ?);

-- name: ListNotificationTagIDs :many
SELECT tag_id FROM notification_tags WHERE notification_id = ?;
