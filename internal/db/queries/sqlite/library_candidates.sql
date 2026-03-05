-- name: UpsertLibraryFileCandidate :exec
INSERT INTO library_file_candidates
    (library_id, file_path, file_size, parsed_title, parsed_year, scanned_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(library_id, file_path) DO UPDATE SET
    file_size    = excluded.file_size,
    parsed_title = excluded.parsed_title,
    parsed_year  = excluded.parsed_year,
    scanned_at   = excluded.scanned_at;

-- name: SetLibraryFileCandidateMatch :exec
UPDATE library_file_candidates
SET tmdb_id             = ?,
    tmdb_title          = ?,
    tmdb_year           = ?,
    tmdb_original_title = ?,
    auto_matched        = 1,
    matched_at          = ?
WHERE library_id = ? AND file_path = ?;

-- name: ListLibraryFileCandidates :many
SELECT * FROM library_file_candidates WHERE library_id = ?;

-- name: ListUnmatchedLibraryFileCandidates :many
SELECT * FROM library_file_candidates
WHERE library_id = ? AND tmdb_id = 0 AND parsed_year > 0 AND parsed_title != '';

-- name: DeleteLibraryFileCandidate :exec
DELETE FROM library_file_candidates WHERE library_id = ? AND file_path = ?;

-- name: PruneStaleLibraryFileCandidates :exec
-- Removes candidates that were not seen in the current scan (scanned_at < cutoff).
DELETE FROM library_file_candidates WHERE library_id = ? AND scanned_at < ?;
