-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
CREATE UNIQUE INDEX users_username_unique
ON users (username)
WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
DROP INDEX users_username_unique;
-- +goose StatementEnd
