-- +goose Up

-- Tags are the glue that links movies to specific indexers, download clients,
-- and notifications. A movie uses resources that share at least one tag — plus
-- all untagged resources (the default pool).

CREATE TABLE tags (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE movie_tags (
    movie_id TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    tag_id   TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (movie_id, tag_id)
);

CREATE TABLE indexer_tags (
    indexer_id TEXT NOT NULL REFERENCES indexer_configs(id) ON DELETE CASCADE,
    tag_id     TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (indexer_id, tag_id)
);

CREATE TABLE download_client_tags (
    download_client_id TEXT NOT NULL REFERENCES download_client_configs(id) ON DELETE CASCADE,
    tag_id             TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (download_client_id, tag_id)
);

CREATE TABLE notification_tags (
    notification_id TEXT NOT NULL REFERENCES notification_configs(id) ON DELETE CASCADE,
    tag_id          TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (notification_id, tag_id)
);

-- +goose Down

DROP TABLE IF EXISTS notification_tags;
DROP TABLE IF EXISTS download_client_tags;
DROP TABLE IF EXISTS indexer_tags;
DROP TABLE IF EXISTS movie_tags;
DROP TABLE IF EXISTS tags;
