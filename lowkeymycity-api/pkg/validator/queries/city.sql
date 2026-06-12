-- name: GetCities :many
SELECT label FROM cities ORDER BY population DESC, label;
