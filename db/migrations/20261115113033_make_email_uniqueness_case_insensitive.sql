-- +goose Up
-- Email uniqueness must be case-insensitive, but this database does not
-- support expression indexes (e.g. UNIQUE (lower(email))) or generated columns.
-- citext gives the same guarantee: comparisons and unique indexes ignore case,
-- while the address is still stored as the user typed it.
-- citext ships with the database, so no CREATE EXTENSION is needed (and this
-- provider rejects it as an unhandled command for an already installed
-- extension).
ALTER TABLE users
    ALTER COLUMN email TYPE citext;

-- +goose Down
ALTER TABLE users
    ALTER COLUMN email TYPE varchar(255);
