-- name: ListAccounts :many
SELECT a.id, a.owner_id, a.name, a.bank, a.account_type, a.alias,
       a.anchor_date, a.anchor_balance,
       a.created_at, a.updated_at,
       (a.owner_id = @user_id::uuid) AS is_owner
FROM accounts a
LEFT JOIN account_users au ON au.account_id = a.id AND au.user_id = @user_id::uuid
WHERE a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL
ORDER BY is_owner DESC, a.created_at;

-- name: GetAccount :one
SELECT a.id, a.owner_id, a.name, a.bank, a.account_type, a.alias,
       a.anchor_date, a.anchor_balance,
       a.created_at, a.updated_at,
       (a.owner_id = @user_id::uuid) AS is_owner
FROM accounts a
LEFT JOIN account_users au ON au.account_id = a.id AND au.user_id = @user_id::uuid
WHERE a.id = @id::bigint
  AND (a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL);

-- name: CreateAccount :one
INSERT INTO accounts (
  owner_id, name, bank, account_type, alias,
  anchor_balance
) VALUES (
  @owner_id::uuid,
  @name::text,
  @bank::text,
  @account_type::smallint,
  sqlc.narg('alias')::text,
  @anchor_balance::jsonb
)
RETURNING id, owner_id, name, bank, account_type, alias,
          anchor_date, anchor_balance,
          created_at, updated_at;

-- name: UpdateAccount :one
UPDATE accounts
SET name = COALESCE(sqlc.narg('name')::text, name),
    bank = COALESCE(sqlc.narg('bank')::text, bank),
    account_type = COALESCE(sqlc.narg('account_type')::smallint, account_type),
    alias = COALESCE(sqlc.narg('alias')::text, alias),
    anchor_date = COALESCE(sqlc.narg('anchor_date')::date, anchor_date),
    anchor_balance = COALESCE(sqlc.narg('anchor_balance')::jsonb, anchor_balance)
WHERE id = @id::bigint
RETURNING id, owner_id, name, bank, account_type, alias,
          anchor_date, anchor_balance,
          created_at, updated_at;

-- name: DeleteAccount :execrows
DELETE FROM accounts 
WHERE id = @id::bigint AND owner_id = @user_id::uuid;

-- name: SetAccountAnchor :execrows
UPDATE accounts
SET anchor_date = NOW()::date,
    anchor_balance = @anchor_balance::jsonb
WHERE id = @id::bigint;

-- name: GetAccountBalance :one
SELECT balance_after
FROM transactions
WHERE account_id = @account_id::bigint
ORDER BY tx_date DESC, id DESC
LIMIT 1;

-- name: GetAccountAnchorBalance :one
SELECT anchor_balance
FROM accounts
WHERE id = @id::bigint;

-- name: CheckUserAccountAccess :one
SELECT EXISTS(
  SELECT 1 FROM accounts a
  LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
  WHERE a.id = @account_id::bigint 
    AND (a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL)
) AS has_access;

-- name: GetUserAccountsCount :one
SELECT COUNT(*) AS account_count
FROM accounts a
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
WHERE a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL;
