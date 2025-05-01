-- +goose Up
-- +goose StatementBegin
ALTER TABLE config ADD COLUMN smtp_from TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE config DROP COLUMN IF EXISTS smtp_from;
-- +goose StatementEnd 