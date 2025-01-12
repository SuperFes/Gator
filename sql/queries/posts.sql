-- name: AddPost :one
INSERT INTO posts (id, created_at, updated_at, title, description, url, feed_id, user_id)
VALUES (
       $1,
       $2,
       $3,
       $4,
       $5,
       $6,
       $7,
       $8
)
RETURNING *;

-- name: GetPosts :many
SELECT * FROM posts WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: GetPost :one
SELECT * FROM posts WHERE id = $1;

-- name: ReadPost :exec
UPDATE posts SET read_at = NOW() WHERE id = $1;
