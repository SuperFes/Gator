-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES ($1,
        NOW(),
        NOW(),
        $2,
        $3,
        $4)
RETURNING *;

-- name: GetFeeds :many
SELECT feeds.name, feeds.url, users.name as username
FROM feeds
         INNER JOIN users ON feeds.user_id = users.id;

-- name: CreateFeedFollow :one
INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
VALUES ($1,
        NOW(),
        NOW(),
        $2,
        $3)
RETURNING *;

-- name: GetFeedFollows :many
SELECT users.name as username, feeds.name as feed_name, feeds.url, feed_follows.feed_id
FROM feed_follows
         INNER JOIN users ON feed_follows.user_id = users.id
         INNER JOIN feeds ON feed_follows.feed_id = feeds.id
WHERE users.name = $1;

-- name: GetFeed :one
SELECT *
FROM feeds
WHERE feeds.url = $1;

-- name: DeleteFeeds :exec
DELETE
FROM feeds;
DELETE
FROM feed_follows;

-- name: DeleteFeedFollow :exec
DELETE
FROM feed_follows
WHERE feed_id = $1
  AND user_id = $2;

-- name: GetNextFeedToFetch :many
SELECT feeds.id, feeds.url
FROM feeds
         INNER JOIN feed_follows ON feeds.id = feed_follows.feed_id
WHERE feeds.last_fetched_at IS NULL
   OR feeds.last_fetched_at < NOW() - INTERVAL $1 AND feed_follows.user_id = $2
ORDER BY feeds.last_fetched_at ASC NULLS FIRST;

-- name: UpdateFeedLastFetchedAt :exec
UPDATE feeds
SET last_fetched_at = NOW()
WHERE id = $1;
