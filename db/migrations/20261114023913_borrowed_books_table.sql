-- +goose Up
CREATE TABLE borrowed_books (
    id SERIAL PRIMARY KEY,
    book_id INTEGER NOT NULL REFERENCES books (id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    borrowed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    due_date TIMESTAMPTZ NOT NULL,
    returned_at TIMESTAMPTZ
);

CREATE INDEX idx_borrowed_books_user_id ON borrowed_books (user_id);
CREATE INDEX idx_borrowed_books_book_id ON borrowed_books (book_id);

-- +goose Down
DROP TABLE IF EXISTS borrowed_books;
