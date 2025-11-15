-- +goose Up

-- add user preference columns
ALTER TABLE users ADD COLUMN primary_currency VARCHAR(3) NOT NULL DEFAULT 'CAD';
ALTER TABLE users ADD COLUMN timezone VARCHAR(50) NOT NULL DEFAULT 'America/Toronto';

ALTER TABLE users ADD CONSTRAINT valid_currency_code
  CHECK (primary_currency ~ '^[A-Z]{3}$');

CREATE INDEX idx_users_timezone ON users(timezone);
CREATE INDEX idx_users_primary_currency ON users(primary_currency);

-- +goose Down

DROP INDEX IF EXISTS idx_users_primary_currency;
DROP INDEX IF EXISTS idx_users_timezone;
ALTER TABLE users DROP CONSTRAINT IF EXISTS valid_currency_code;
ALTER TABLE users DROP COLUMN IF EXISTS timezone;
ALTER TABLE users DROP COLUMN IF EXISTS primary_currency;
