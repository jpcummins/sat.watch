-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
ALTER TABLE emails ADD pgp_pubkey text;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
ALTER TABLE emails DROP COLUMN pgp_pubkey;
-- +goose StatementEnd
