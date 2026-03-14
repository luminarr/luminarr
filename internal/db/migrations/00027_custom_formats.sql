-- +goose Up

CREATE TABLE custom_formats (
    id                      TEXT PRIMARY KEY,
    name                    TEXT NOT NULL UNIQUE,
    include_when_renaming   INTEGER NOT NULL DEFAULT 0,
    specifications_json     TEXT NOT NULL DEFAULT '[]',   -- JSON array of condition specs
    created_at              TEXT NOT NULL,
    updated_at              TEXT NOT NULL
);

-- Per-quality-profile scoring for each custom format.
CREATE TABLE custom_format_scores (
    quality_profile_id TEXT NOT NULL REFERENCES quality_profiles(id) ON DELETE CASCADE,
    custom_format_id   TEXT NOT NULL REFERENCES custom_formats(id) ON DELETE CASCADE,
    score              INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (quality_profile_id, custom_format_id)
);

-- Add CF score thresholds to quality profiles.
ALTER TABLE quality_profiles ADD COLUMN min_custom_format_score  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE quality_profiles ADD COLUMN upgrade_until_cf_score   INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE quality_profiles DROP COLUMN upgrade_until_cf_score;
ALTER TABLE quality_profiles DROP COLUMN min_custom_format_score;
DROP TABLE IF EXISTS custom_format_scores;
DROP TABLE IF EXISTS custom_formats;
