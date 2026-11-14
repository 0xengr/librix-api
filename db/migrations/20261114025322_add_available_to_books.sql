-- +goose Up
ALTER TABLE books
ADD COLUMN available BOOLEAN NOT NULL DEFAULT TRUE;

-- +goose Down
ALTER TABLE books
DROP COLUMN IF EXISTS available;
