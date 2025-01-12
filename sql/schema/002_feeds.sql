-- +goose Up
CREATE TABLE feeds
(
    id         UUID                 NOT NULL PRIMARY KEY,
    created_at TIMESTAMP            NOT NULL,
    updated_at TIMESTAMP            NOT NULL,
    name       VARCHAR(255) UNIQUE  NOT NULL,
    url        VARCHAR(2048) UNIQUE NOT NULL,
    user_id    UUID                 NOT NULL REFERENCES users (id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE feeds;
