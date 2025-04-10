-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE TABLE emails (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    email text NOT NULL,
    user_id uuid NOT NULL,
    description text NOT NULL,
    verification_code text DEFAULT gen_random_uuid(),
    verification_expires timestamp with time zone,
    is_verified boolean NOT NULL,
    verified_on timestamp with time zone,
    UNIQUE (user_id, email)
);
CREATE INDEX idx_emails_deleted_at ON emails USING btree (deleted_at);
CREATE TRIGGER update_modified_time BEFORE UPDATE ON emails FOR EACH ROW EXECUTE PROCEDURE update_modified_column();
ALTER TABLE ONLY emails ADD CONSTRAINT fk_emails_user FOREIGN KEY (user_id) REFERENCES users(id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
DROP TABLE IF EXISTS emails;
-- +goose StatementEnd
