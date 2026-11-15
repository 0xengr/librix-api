-- +goose Up
ALTER TABLE books ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;

-- Add index for performance when filtering out deleted
CREATE INDEX idx_books_deleted_at ON books(deleted_at);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- +goose Down
DROP INDEX IF EXISTS idx_books_deleted_at;
DROP INDEX IF EXISTS idx_users_deleted_at;

ALTER TABLE books DROP COLUMN deleted_at;
ALTER TABLE users DROP COLUMN deleted_at;
