-- name: GetCityQuestions :many
SELECT meaning_id, text, options FROM city_questions
WHERE city_label = $1
ORDER BY position;

-- name: AddCityQuestion :exec
INSERT INTO city_questions (city_label, position, meaning_id, text, options)
VALUES ($1, $2, $3, $4, $5);

-- name: GetMatchQuestions :many
SELECT meaning_id, text, options FROM match_questions
ORDER BY position;

-- name: GetResultByCombination :one
SELECT * FROM results
WHERE combination = $1;

-- name: SaveResult :one
INSERT INTO results (combination, mode, city, score, title, summary,
                     green_flags, red_flags, alternatives, closing)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
-- the no-op update makes RETURNING yield the existing row when two
-- identical submissions race, instead of failing the second one
ON CONFLICT (combination) DO UPDATE SET combination = EXCLUDED.combination
RETURNING *;
