-- +goose Up
-- Replace NUMERIC types with native Go-friendly types

-- Change exchange_rate from NUMERIC to DOUBLE PRECISION
ALTER TABLE transactions
  ALTER COLUMN exchange_rate TYPE DOUBLE PRECISION;

-- Change qty from NUMERIC to INT
ALTER TABLE receipt_items
  ALTER COLUMN qty TYPE INT USING COALESCE(qty::int, 1);

-- +goose Down
-- Revert to NUMERIC types

ALTER TABLE receipt_items
  ALTER COLUMN qty TYPE NUMERIC;

ALTER TABLE transactions
  ALTER COLUMN exchange_rate TYPE NUMERIC;
