-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
NEW.updated_at = now();
RETURN NEW;
END;
$$ language 'plpgsql'; 

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    username text,
    password text
);
CREATE INDEX idx_users_deleted_at ON users USING btree (deleted_at);
CREATE TRIGGER update_modified_time BEFORE UPDATE ON users FOR EACH ROW EXECUTE PROCEDURE update_modified_column();

CREATE TABLE webhooks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    user_id uuid NOT NULL,
    name text NOT NULL,
    url text NOT NULL,
    UNIQUE (user_id, url)
);
CREATE INDEX idx_webhooks_deleted_at ON webhooks USING btree (deleted_at);
CREATE TRIGGER update_modified_time BEFORE UPDATE ON webhooks FOR EACH ROW EXECUTE PROCEDURE update_modified_column();
ALTER TABLE ONLY webhooks ADD CONSTRAINT fk_webhooks_user FOREIGN KEY (user_id) REFERENCES users(id);

CREATE TABLE xpubs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    user_id uuid NOT NULL,
    pubkey text NOT NULL,
    name text,
    gap bigint NOT NULL,
    UNIQUE (user_id, pubkey)
);
CREATE INDEX idx_xpubs_deleted_at ON xpubs USING btree (deleted_at);
CREATE TRIGGER update_modified_time BEFORE UPDATE ON xpubs FOR EACH ROW EXECUTE PROCEDURE update_modified_column();
ALTER TABLE ONLY xpubs ADD CONSTRAINT fk_xpubs_user FOREIGN KEY (user_id) REFERENCES users(id);

CREATE TABLE addresses (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    user_id uuid NOT NULL,
    xpub_id uuid,
    address text NOT NULL,
    name text,
    is_external boolean NOT NULL,
    address_index bigint NOT NULL,
    UNIQUE (user_id, address)
);
CREATE INDEX idx_addresses_deleted_at ON addresses USING btree (deleted_at);
CREATE TRIGGER update_modified_time BEFORE UPDATE ON addresses FOR EACH ROW EXECUTE PROCEDURE update_modified_column();
ALTER TABLE ONLY addresses ADD CONSTRAINT fk_addresses_user FOREIGN KEY (user_id) REFERENCES users(id);
ALTER TABLE ONLY addresses ADD CONSTRAINT fk_addresses_xpub FOREIGN KEY (xpub_id) REFERENCES xpubs(id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS webhooks;
DROP TABLE IF EXISTS addresses;
DROP TABLE IF EXISTS xpubs;
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS update_modified_column;
-- +goose StatementEnd
