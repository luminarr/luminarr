-- name: CreateIndexerConfig :one
INSERT INTO indexer_configs (id, name, kind, enabled, priority, settings, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetIndexerConfig :one
SELECT * FROM indexer_configs WHERE id = ?;

-- name: ListIndexerConfigs :many
SELECT * FROM indexer_configs ORDER BY priority ASC, name ASC;

-- name: ListEnabledIndexers :many
SELECT * FROM indexer_configs WHERE enabled = 1 ORDER BY priority ASC, name ASC;

-- name: UpdateIndexerConfig :one
UPDATE indexer_configs SET
    name       = ?,
    kind       = ?,
    enabled    = ?,
    priority   = ?,
    settings   = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteIndexerConfig :exec
DELETE FROM indexer_configs WHERE id = ?;

-- name: CreateGrabHistory :one
INSERT INTO grab_history (
    id, movie_id, indexer_id, release_guid, release_title,
    release_source, release_resolution, release_codec, release_hdr,
    protocol, size, download_client_id, client_item_id, grabbed_at,
    download_status, downloaded_bytes, score_breakdown, release_edition
) VALUES (
    ?, ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?, ?, ?
)
RETURNING *;

-- name: ListGrabHistoryByMovie :many
SELECT * FROM grab_history WHERE movie_id = ? ORDER BY grabbed_at DESC;

-- name: ListGrabHistory :many
SELECT * FROM grab_history ORDER BY grabbed_at DESC LIMIT ?;

-- name: UpdateGrabDownloadClient :exec
UPDATE grab_history
SET download_client_id = ?, client_item_id = ?, download_status = 'queued'
WHERE id = ?;

-- name: UpdateGrabStatus :exec
UPDATE grab_history
SET download_status = ?, downloaded_bytes = ?
WHERE id = ?;

-- name: ListActiveGrabs :many
SELECT * FROM grab_history
WHERE client_item_id IS NOT NULL
  AND download_status NOT IN ('completed', 'failed', 'removed')
ORDER BY grabbed_at DESC;

-- name: GetGrabByClientItemID :one
SELECT * FROM grab_history
WHERE download_client_id = ? AND client_item_id = ?
LIMIT 1;

-- name: MarkGrabRemoved :exec
UPDATE grab_history SET download_status = 'removed' WHERE id = ?;

-- name: ListGrabHistoryByStatus :many
SELECT * FROM grab_history WHERE download_status = ? ORDER BY grabbed_at DESC LIMIT ?;

-- name: ListGrabHistoryByProtocol :many
SELECT * FROM grab_history WHERE protocol = ? ORDER BY grabbed_at DESC LIMIT ?;

-- name: ListGrabHistoryByStatusAndProtocol :many
SELECT * FROM grab_history WHERE download_status = ? AND protocol = ? ORDER BY grabbed_at DESC LIMIT ?;

-- name: GetGrabByID :one
SELECT * FROM grab_history WHERE id = ?;
