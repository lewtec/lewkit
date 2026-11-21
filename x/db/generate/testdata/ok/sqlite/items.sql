-- name: ListItems :many
SELECT id, name FROM items;

-- name: GetItem :one
SELECT id, name FROM items WHERE id = ?;
