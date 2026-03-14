-- name: ListCustomFormats :many
SELECT * FROM custom_formats ORDER BY name ASC;

-- name: GetCustomFormat :one
SELECT * FROM custom_formats WHERE id = ?;

-- name: CreateCustomFormat :one
INSERT INTO custom_formats (id, name, include_when_renaming, specifications_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateCustomFormat :one
UPDATE custom_formats
SET name = ?, include_when_renaming = ?, specifications_json = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteCustomFormat :exec
DELETE FROM custom_formats WHERE id = ?;

-- name: ListCustomFormatScores :many
SELECT * FROM custom_format_scores WHERE quality_profile_id = ?;

-- name: SetCustomFormatScore :exec
INSERT INTO custom_format_scores (quality_profile_id, custom_format_id, score)
VALUES (?, ?, ?)
ON CONFLICT (quality_profile_id, custom_format_id) DO UPDATE SET score = excluded.score;

-- name: DeleteCustomFormatScores :exec
DELETE FROM custom_format_scores WHERE quality_profile_id = ?;
