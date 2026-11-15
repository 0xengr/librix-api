-- +goose Up
ALTER TABLE users DROP CONSTRAINT users_email_key;

CREATE UNIQUE INDEX idx_unique_active_user_email
    ON users(email)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_unique_active_user_email;

-- Restores the original global constraint. This fails (with the duplicate key
-- in the error detail) if an email was reused after its user was soft-deleted;
-- this provider supports neither DO blocks nor functions, so there is no way
-- to pre-check that here.
ALTER TABLE users
    ADD CONSTRAINT users_email_key UNIQUE (email);
