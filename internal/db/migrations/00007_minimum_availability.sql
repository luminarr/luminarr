-- +goose Up

ALTER TABLE movies
    ADD COLUMN minimum_availability TEXT NOT NULL DEFAULT 'released';

-- +goose Down

ALTER TABLE movies DROP COLUMN minimum_availability;
