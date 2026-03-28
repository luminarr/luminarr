-- +goose Up
CREATE TABLE watch_history (
    id         TEXT PRIMARY KEY,
    movie_id   TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    tmdb_id    INTEGER NOT NULL,
    watched_at TEXT NOT NULL,
    user_name  TEXT NOT NULL DEFAULT '',
    source     TEXT NOT NULL,
    UNIQUE(movie_id, watched_at, user_name)
);
CREATE INDEX idx_watch_history_movie   ON watch_history(movie_id);
CREATE INDEX idx_watch_history_watched ON watch_history(watched_at DESC);

-- Tracks the last successful sync time per media server config.
CREATE TABLE watch_sync_state (
    media_server_id TEXT PRIMARY KEY,
    last_sync_at    TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS watch_sync_state;
DROP TABLE IF EXISTS watch_history;
