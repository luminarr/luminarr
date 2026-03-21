-- +goose Up
-- Remove movie_files rows whose movie_id no longer exists in the movies table.
-- These orphans accumulate when movies are deleted while foreign_keys was
-- temporarily disabled (e.g., during Radarr import migrations).
DELETE FROM movie_files
WHERE movie_id NOT IN (SELECT id FROM movies);

-- Also clean up orphaned grab_history rows.
DELETE FROM grab_history
WHERE movie_id NOT IN (SELECT id FROM movies);

-- +goose Down
-- Cannot restore deleted orphans.
