-- +goose Up
-- +goose StatementBegin
CREATE TABLE config (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    smtp_host text,
    smtp_port integer,
    smtp_user text,
    smtp_password text
);

-- Insert initial row
INSERT INTO config (id) VALUES (gen_random_uuid());
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS config;
-- +goose StatementEnd 