-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
ALTER TABLE addresses ADD scripthash text NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
ALTER TABLE addresses DROP COLUMN scripthash;
-- +goose StatementEnd
