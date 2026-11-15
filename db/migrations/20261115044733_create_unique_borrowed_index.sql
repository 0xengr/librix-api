-- +goose Up
CREATE UNIQUE INDEX idx_one_active_borrow_per_book
    ON borrowed_books(book_id)
    WHERE returned_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_one_active_borrow_per_book;
