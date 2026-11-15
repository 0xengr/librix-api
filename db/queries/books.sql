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


-- GetBookForUpdate locks a book for the borrow flow. Soft-deleted books cannot
-- be borrowed, so they are excluded (and therefore not locked) here.
-- name: GetBookForUpdate :one
SELECT * FROM books
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE;

-- GetBookForUpdateIncludingDeleted locks a book for the return flow. A loan must
-- stay returnable even if the book was soft-deleted after it was borrowed, so
-- this query deliberately does not filter on deleted_at.
-- name: GetBookForUpdateIncludingDeleted :one
SELECT * FROM books
WHERE id = $1
FOR UPDATE;
