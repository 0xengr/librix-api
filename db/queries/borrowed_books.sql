-- name: BorrowBook :one
INSERT INTO borrowed_books (book_id, user_id, due_date)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ReturnBook :one
UPDATE borrowed_books
SET returned_at = CURRENT_TIMESTAMP
WHERE book_id = $1
  AND user_id = $2
  AND returned_at IS NULL
RETURNING *;

-- name: GetUserBorrowedBooks :many
SELECT bb.*, b.title, b.author
FROM borrowed_books bb
JOIN books b ON bb.book_id = b.id
WHERE bb.user_id = $1
  AND bb.returned_at IS NULL
ORDER BY bb.borrowed_at DESC;

-- name: GetOverDueBooks :many
SELECT bb.*, b.title, b.author, u.name, u.email
FROM borrowed_books bb
JOIN books b ON bb.book_id = b.id
JOIN users u ON bb.user_id = u.id
WHERE bb.returned_at IS NULL
  AND bb.due_date < CURRENT_TIMESTAMP
ORDER BY bb.due_date ASC;

-- name: UpdateBookAvailability :exec
UPDATE books
SET available = $2
WHERE id = $1;
