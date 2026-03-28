-- +goose Up
CREATE TABLE activity_log (
    id         TEXT PRIMARY KEY,
    type       TEXT NOT NULL,
    category   TEXT NOT NULL,
    movie_id   TEXT,
    title      TEXT NOT NULL,
    detail     TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_activity_log_created  ON activity_log(created_at DESC);
CREATE INDEX idx_activity_log_category ON activity_log(category);
CREATE INDEX idx_activity_log_movie    ON activity_log(movie_id);

-- +goose Down
DROP TABLE IF EXISTS activity_log;
