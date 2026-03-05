-- +goose Up
CREATE TABLE collections (
    id          TEXT    PRIMARY KEY,
    name        TEXT    NOT NULL,
    person_id   INTEGER NOT NULL,
    person_type TEXT    NOT NULL DEFAULT 'director',
    created_at  DATETIME NOT NULL
);
CREATE UNIQUE INDEX idx_collections_person ON collections(person_id, person_type);

-- +goose Down
DROP TABLE collections;
