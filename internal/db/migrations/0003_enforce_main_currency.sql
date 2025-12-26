-- +goose Up
-- Fix existing accounts with empty/null currency fields
UPDATE accounts
SET main_currency = 'CAD'
WHERE main_currency IS NULL OR main_currency = '';

UPDATE accounts
SET anchor_currency = 'CAD'
WHERE anchor_currency IS NULL OR anchor_currency = '';

-- Add constraints to enforce non-empty currencies
ALTER TABLE accounts
ADD CONSTRAINT check_main_currency_not_empty
  CHECK (main_currency IS NOT NULL AND main_currency != '');

ALTER TABLE accounts
ADD CONSTRAINT check_anchor_currency_not_empty
  CHECK (anchor_currency IS NOT NULL AND anchor_currency != '');

-- +goose Down
-- Remove constraints
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS check_main_currency_not_empty;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS check_anchor_currency_not_empty;
