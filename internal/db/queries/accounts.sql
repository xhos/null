-- name: ListAccounts :many
select
  a.*
from
  accounts a
  left join account_users au on au.account_id = a.id
  and au.user_id = @user_id::uuid
where
  a.owner_id = @user_id::uuid
  or au.user_id is not null
order by
  (a.owner_id = @user_id::uuid) desc,
  a.created_at;

-- name: GetAccount :one
select
  a.*
from
  accounts a
  left join account_users au on au.account_id = a.id
  and au.user_id = @user_id::uuid
where
  a.id = @id::bigint
  and (
    a.owner_id = @user_id::uuid
    or au.user_id is not null
  );

-- name: CreateAccount :one
insert into
  accounts (
    owner_id,
    name,
    bank,
    account_type,
    alias,
    anchor_balance,
    balance,
    main_currency,
    colors
  )
values
  (
    @owner_id::uuid,
    @name::text,
    @bank::text,
    @account_type::smallint,
    sqlc.narg('alias')::text,
    @anchor_balance::jsonb,
    @anchor_balance::jsonb,
    @main_currency::text,
    @colors::text []
  )
returning
  *;

-- name: UpdateAccount :one
update
  accounts
set
  name = COALESCE(sqlc.narg('name')::text, name),
  bank = COALESCE(sqlc.narg('bank')::text, bank),
  account_type = COALESCE(
    sqlc.narg('account_type')::smallint,
    account_type
  ),
  alias = COALESCE(sqlc.narg('alias')::text, alias),
  anchor_date = COALESCE(sqlc.narg('anchor_date')::date, anchor_date),
  anchor_balance = COALESCE(
    sqlc.narg('anchor_balance')::jsonb,
    anchor_balance
  ),
  balance = COALESCE(sqlc.narg('balance')::jsonb, balance),
  main_currency = COALESCE(sqlc.narg('main_currency')::text, main_currency),
  colors = COALESCE(sqlc.narg('colors')::text [], colors)
where
  id = @id::bigint
returning
  *;

-- name: DeleteAccount :execrows
delete from
  accounts
where
  id = @id::bigint
  and owner_id = @user_id::uuid;

-- name: SetAccountAnchor :execrows
update
  accounts
set
  anchor_date = now()::date,
  anchor_balance = @anchor_balance::jsonb
where
  id = @id::bigint;

-- name: GetAccountBalance :one
select
  balance_after
from
  transactions
where
  account_id = @account_id::bigint
order by
  tx_date desc,
  id desc
limit
  1;

-- name: GetAccountBalanceSimple :one
select
  balance
from
  accounts
where
  id = @account_id::bigint;

-- name: GetAccountAnchorBalance :one
select
  anchor_balance
from
  accounts
where
  id = @id::bigint;

-- name: CheckUserAccountAccess :one
select
  exists(
    select
      1
    from
      accounts a
      left join account_users au on a.id = au.account_id
      and au.user_id = @user_id::uuid
    where
      a.id = @account_id::bigint
      and (
        a.owner_id = @user_id::uuid
        or au.user_id is not null
      )
  ) as has_access;

-- name: GetUserAccountsCount :one
select
  COUNT(*) as account_count
from
  accounts a
  left join account_users au on a.id = au.account_id
  and au.user_id = @user_id::uuid
where
  a.owner_id = @user_id::uuid
  or au.user_id is not null;
