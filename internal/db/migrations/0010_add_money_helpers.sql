-- +goose Up
-- SQL helper functions for money operations

-- +goose StatementBegin
CREATE FUNCTION amount(money_json jsonb)
RETURNS NUMERIC AS $$
BEGIN
    IF money_json IS NULL THEN
        RETURN 0;
    END IF;
    RETURN (money_json->>'units')::bigint + (money_json->>'nanos')::bigint / 1000000000.0;
END;
$$ LANGUAGE plpgsql IMMUTABLE;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION amount_cents(money_json jsonb)
RETURNS BIGINT AS $$
BEGIN
    IF money_json IS NULL THEN
        RETURN 0;
    END IF;
    RETURN (money_json->>'units')::bigint * 100 + (money_json->>'nanos')::bigint / 10000000;
END;
$$ LANGUAGE plpgsql IMMUTABLE;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS amount_cents(jsonb);
DROP FUNCTION IF EXISTS amount(jsonb);
