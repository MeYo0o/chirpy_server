-- name: CreateChirpy :one
INSERT INTO chirps(id, body, user_id, created_at, updated_at)
VALUES($1, $2, $3, $4, $5)
RETURNING *;
-- name: GetChirps :many
SELECT *
FROM chirps
ORDER BY created_at ASC;
-- name: GetChirpy :one
SELECT *
From chirps
WHERE id = $1;
-- name: GetChirpsByUserID :many
SELECT *
FROM chirps
WHERE user_id = $1
ORDER BY created_at ASC;
-- name: GetChirpyByUserID :one
SELECT *
FROM chirps
WHERE id = $1
  AND user_id = $2;
-- name: DeleteChirp :exec
DELETE FROM chirps
WHERE id = $1
  AND user_id = $2;