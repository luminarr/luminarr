-- name: CreateCollection :one
INSERT INTO collections (id, name, person_id, person_type, created_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListCollections :many
SELECT * FROM collections ORDER BY name ASC;

-- name: GetCollection :one
SELECT * FROM collections WHERE id = ?;

-- name: GetCollectionByPerson :one
SELECT * FROM collections WHERE person_id = ? AND person_type = ?;

-- name: DeleteCollection :exec
DELETE FROM collections WHERE id = ?;
