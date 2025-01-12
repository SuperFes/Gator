-- +goose Up
CREATE TABLE posts
(
    id           UUID      NOT NULL PRIMARY KEY,
    created_at   TIMESTAMP NOT NULL,
    updated_at   TIMESTAMP NOT NULL,
    read_at      TIMESTAMP NULL,
    user_id      UUID      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    feed_id      UUID      NOT NULL REFERENCES feeds (id) ON DELETE CASCADE,
    title        TEXT,
    description  TEXT,
    url          TEXT      NOT NULL,
    published_at TIMESTAMP,
    content      TEXT      NOT NULL
);

-- +goose Down
DROP TABLE posts;
