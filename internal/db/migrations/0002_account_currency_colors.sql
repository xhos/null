-- +goose Up
ALTER TABLE accounts
ADD COLUMN main_currency TEXT NOT NULL DEFAULT 'CAD'
CHECK (main_currency ~ '^[A-Z]{3}$');

ALTER TABLE accounts
ADD COLUMN colors TEXT[] DEFAULT ARRAY['#1f2937', '#3b82f6', '#10b981']
CHECK (
  array_length(colors, 1) = 3 AND
  colors[1] ~ '^#[0-9a-fA-F]{6}$' AND
  colors[2] ~ '^#[0-9a-fA-F]{6}$' AND
  colors[3] ~ '^#[0-9a-fA-F]{6}$'
);

CREATE INDEX idx_accounts_main_currency ON accounts(main_currency);

-- +goose Down
ALTER TABLE accounts DROP COLUMN IF EXISTS colors;
ALTER TABLE accounts DROP COLUMN IF EXISTS main_currency;
