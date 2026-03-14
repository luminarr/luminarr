-- +goose Up

-- Preferred edition per movie (NULL = no preference = accept any edition).
ALTER TABLE movies ADD COLUMN preferred_edition TEXT;

-- Index the existing edition column on movie_files for mismatch queries.
CREATE INDEX movie_files_edition ON movie_files(edition);

-- Store detected edition on grab history for visibility.
ALTER TABLE grab_history ADD COLUMN release_edition TEXT;

-- +goose Down
DROP INDEX IF EXISTS movie_files_edition;
