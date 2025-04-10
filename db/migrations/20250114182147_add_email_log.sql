-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd

CREATE TABLE email_log (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    email text NOT NULL,
    user_id text NOT NULL,
    address_id text NOT NULL,
    transaction_id text NOT NULL,
    error text
);

CREATE INDEX idx_email_log_user_created_at ON email_log (user_id, address_id, created_at);

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
DROP TABLE IF EXISTS email_log;
