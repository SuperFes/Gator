-- +goose Up
ALTER TABLE posts DROP COLUMN content;

-- +goose Down
ALTER TABLE posts ADD COLUMN content TEXT;
