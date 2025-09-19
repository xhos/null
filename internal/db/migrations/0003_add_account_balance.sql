-- +goose Up

ALTER TABLE accounts 
ADD COLUMN balance JSONB NOT NULL DEFAULT '{"currency_code":"CAD","units":0,"nanos":0}'
    CHECK (
        jsonb_typeof(balance) = 'object' AND
        balance ? 'currency_code' AND
        balance ? 'units' AND
        balance ? 'nanos'
    );

CREATE INDEX idx_accounts_balance ON accounts USING GIN (balance);

-- for accounts with transactions: use the latest transaction's balance_after
-- for accounts without transactions: use the anchor_balance
UPDATE accounts SET balance = (
    SELECT COALESCE(
        (SELECT t.balance_after 
         FROM transactions t 
         WHERE t.account_id = accounts.id 
         ORDER BY t.tx_date DESC, t.id DESC 
         LIMIT 1),
        accounts.anchor_balance
    )
);

-- automatically update account balance when transactions change
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_account_balance_from_transaction()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    latest_balance JSONB;
BEGIN
    -- get the latest transaction balance for this account
    SELECT t.balance_after INTO latest_balance
    FROM transactions t
    WHERE t.account_id = COALESCE(NEW.account_id, OLD.account_id)
    ORDER BY t.tx_date DESC, t.id DESC
    LIMIT 1;
    
    -- if no transactions found, use anchor balance
    IF latest_balance IS NULL THEN
        SELECT a.anchor_balance INTO latest_balance
        FROM accounts a
        WHERE a.id = COALESCE(NEW.account_id, OLD.account_id);
    END IF;
    
    -- update the account balance
    UPDATE accounts 
    SET balance = latest_balance
    WHERE id = COALESCE(NEW.account_id, OLD.account_id);
    
    RETURN COALESCE(NEW, OLD);
END$$;
-- +goose StatementEnd

-- update account balance when transactions are modified
CREATE TRIGGER trg_update_account_balance_on_transaction
    AFTER INSERT OR UPDATE OR DELETE ON transactions
    FOR EACH ROW EXECUTE FUNCTION update_account_balance_from_transaction();

-- update account balance when anchor changes
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_account_balance_from_anchor()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    latest_transaction_balance JSONB;
BEGIN
    -- check if there are any transactions for this account
    SELECT t.balance_after INTO latest_transaction_balance
    FROM transactions t
    WHERE t.account_id = NEW.id
    ORDER BY t.tx_date DESC, t.id DESC
    LIMIT 1;
    
    -- if no transactions exist, use the anchor balance as the account balance
    IF latest_transaction_balance IS NULL THEN
        NEW.balance := NEW.anchor_balance;
    END IF;
    
    RETURN NEW;
END$$;
-- +goose StatementEnd

-- create trigger to update account balance when anchor changes
CREATE TRIGGER trg_update_account_balance_on_anchor
    BEFORE UPDATE OF anchor_balance ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_account_balance_from_anchor();

-- +goose Down

DROP TRIGGER IF EXISTS trg_update_account_balance_on_anchor ON accounts;
DROP TRIGGER IF EXISTS trg_update_account_balance_on_transaction ON transactions;

-- +goose StatementBegin
DROP FUNCTION IF EXISTS update_account_balance_from_anchor();
-- +goose StatementEnd

-- +goose StatementBegin
DROP FUNCTION IF EXISTS update_account_balance_from_transaction();
-- +goose StatementEnd

DROP INDEX IF EXISTS idx_accounts_balance;
ALTER TABLE accounts DROP COLUMN IF EXISTS balance;