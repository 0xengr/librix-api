-- name: CreateBook :one
INSERT INTO books (title, author, description, year_of_publication)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetBook :one
SELECT * FROM books
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListBooks :many
SELECT * FROM books
WHERE deleted_at IS NULL
ORDER BY id;

-- name: UpdateBook :one
UPDATE books
SET title = $2,
    author = $3,
    description = $4,
    year_of_publication = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteBook :one
UPDATE books
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING id;


-- name: GetBookForUpdate :one
SELECT * FROM books
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE;
