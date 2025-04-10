-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
ALTER TABLE addresses DROP CONSTRAINT addresses_user_id_address_key;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
ALTER TABLE addresses
ADD CONSTRAINT addresses_user_id_address_key
UNIQUE (user_id, address);
-- +goose StatementEnd
