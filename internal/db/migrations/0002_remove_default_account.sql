-- +goose Up
-- +goose StatementBegin
-- Drop the foreign key constraint first
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_default_account;

-- Drop the index
DROP INDEX IF EXISTS idx_users_default_account;

-- Drop the column
ALTER TABLE users DROP COLUMN IF EXISTS default_account_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Re-add the column
ALTER TABLE users ADD COLUMN default_account_id BIGINT;

-- Re-create the index
CREATE INDEX idx_users_default_account ON users(default_account_id);

-- Re-add the foreign key constraint
ALTER TABLE users
  ADD CONSTRAINT fk_users_default_account
  FOREIGN KEY (default_account_id)
  REFERENCES accounts(id)
  ON DELETE SET NULL;
-- +goose StatementEnd
