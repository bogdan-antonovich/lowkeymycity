-- name: GetStoredResult :one
SELECT id, mode, city, score, title, summary, green_flags, red_flags,
       alternatives, closing
FROM results
WHERE id = $1;
