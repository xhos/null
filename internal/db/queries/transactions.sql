-- name: ListTransactions :many
select
  t.*
from
  transactions t
  join accounts a on t.account_id = a.id
  left join account_users au on a.id = au.account_id
  and au.user_id = sqlc.arg(user_id)::uuid
  left join categories c on t.category_id = c.id
where
  (
    a.owner_id = sqlc.arg(user_id)::uuid
    or au.user_id is not null
  )
  and (
    sqlc.narg('cursor_date')::timestamptz is null
    or sqlc.narg('cursor_id')::bigint is null
    or (t.tx_date, t.id) < (
      sqlc.narg('cursor_date')::timestamptz,
      sqlc.narg('cursor_id')::bigint
    )
  )
  and (
    sqlc.narg('start')::timestamptz is null
    or t.tx_date >= sqlc.narg('start')::timestamptz
  )
  and (
    sqlc.narg('end')::timestamptz is null
    or t.tx_date <= sqlc.narg('end')::timestamptz
  )
  and (
    sqlc.narg('amount_min')::double precision is null
    or (t.tx_amount ->> 'units')::bigint >= sqlc.narg('amount_min')::double precision
  )
  and (
    sqlc.narg('amount_max')::double precision is null
    or (t.tx_amount ->> 'units')::bigint <= sqlc.narg('amount_max')::double precision
  )
  and (
    sqlc.narg('direction')::smallint is null
    or t.tx_direction = sqlc.narg('direction')::smallint
  )
  and (
    sqlc.narg('account_ids')::bigint [] is null
    or t.account_id = ANY(sqlc.narg('account_ids')::bigint [])
  )
  and (
    sqlc.narg('categories')::text [] is null
    or c.slug = ANY(sqlc.narg('categories')::text [])
  )
  and (
    sqlc.narg('merchant_q')::text is null
    or t.merchant ILIKE ('%' || sqlc.narg('merchant_q')::text || '%')
  )
  and (
    sqlc.narg('desc_q')::text is null
    or t.tx_desc ILIKE ('%' || sqlc.narg('desc_q')::text || '%')
  )
  and (
    sqlc.narg('currency')::char(3) is null
    or t.tx_amount ->> 'currency_code' = sqlc.narg('currency')::char(3)
  )
  and (
    sqlc.narg('tod_start')::time is null
    or t.tx_date::time >= sqlc.narg('tod_start')::time
  )
  and (
    sqlc.narg('tod_end')::time is null
    or t.tx_date::time <= sqlc.narg('tod_end')::time
  )
  and (
    sqlc.narg('uncategorized')::boolean is null
    or (
      sqlc.narg('uncategorized')::boolean = true
      and t.category_id is null
    )
  )
order by
  t.tx_date desc,
  t.id desc
limit
  COALESCE(sqlc.narg('limit')::int, 100);

-- name: GetTransaction :one
select
  t.*
from
  transactions t
  join accounts a on t.account_id = a.id
  left join account_users au on a.id = au.account_id
  and au.user_id = sqlc.arg(user_id)::uuid
where
  t.id = sqlc.arg(id)::bigint
  and (
    a.owner_id = sqlc.arg(user_id)::uuid
    or au.user_id is not null
  );

-- name: CreateTransaction :one
insert into
  transactions (
    email_id,
    account_id,
    tx_date,
    tx_amount,
    tx_direction,
    tx_desc,
    balance_after,
    category_id,
    category_manually_set,
    merchant,
    merchant_manually_set,
    user_notes,
    foreign_amount,
    exchange_rate,
    suggestions
  )
select
  sqlc.narg('email_id')::text,
  sqlc.arg(account_id)::bigint,
  sqlc.arg(tx_date)::timestamptz,
  sqlc.arg(tx_amount)::jsonb,
  sqlc.arg(tx_direction)::smallint,
  sqlc.narg('tx_desc')::text,
  sqlc.narg('balance_after')::jsonb,
  sqlc.narg('category_id')::bigint,
  sqlc.narg('category_manually_set')::boolean,
  sqlc.narg('merchant')::text,
  sqlc.narg('merchant_manually_set')::boolean,
  sqlc.narg('user_notes')::text,
  sqlc.narg('foreign_amount')::jsonb,
  sqlc.narg('exchange_rate')::double precision,
  sqlc.narg('suggestions')::text []
from
  accounts a
  left join account_users au on a.id = au.account_id
  and au.user_id = sqlc.arg(user_id)::uuid
where
  a.id = sqlc.arg(account_id)::bigint
  and (
    a.owner_id = sqlc.arg(user_id)::uuid
    or au.user_id is not null
  )
returning
  id;

-- name: UpdateTransaction :one
update
  transactions
set
  email_id = coalesce(sqlc.narg('email_id')::text, email_id),
  tx_date = coalesce(sqlc.narg('tx_date')::timestamptz, tx_date),
  tx_amount = coalesce(sqlc.narg('tx_amount')::jsonb, tx_amount),
  tx_direction = coalesce(sqlc.narg('tx_direction')::smallint, tx_direction),
  tx_desc = coalesce(sqlc.narg('tx_desc')::text, tx_desc),
  category_id = coalesce(sqlc.narg('category_id')::bigint, category_id),
  merchant = coalesce(sqlc.narg('merchant')::text, merchant),
  user_notes = coalesce(sqlc.narg('user_notes')::text, user_notes),
  foreign_amount = coalesce(sqlc.narg('foreign_amount')::jsonb, foreign_amount),
  exchange_rate = coalesce(sqlc.narg('exchange_rate')::double precision, exchange_rate),
  suggestions = coalesce(sqlc.narg('suggestions')::text[], suggestions),
  category_manually_set = coalesce(sqlc.narg('category_manually_set')::boolean, category_manually_set),
  merchant_manually_set = coalesce(sqlc.narg('merchant_manually_set')::boolean, merchant_manually_set)
where
  id = sqlc.arg(id)::bigint
  and account_id in (
    select
      a.id
    from
      accounts a
      left join account_users au on a.id = au.account_id
      and au.user_id = sqlc.arg(user_id)::uuid
    where
      a.owner_id = sqlc.arg(user_id)::uuid
      or au.user_id is not null
  )
returning
  account_id;

-- name: DeleteTransaction :one
delete from
  transactions
where
  id = sqlc.arg(id)::bigint
  and account_id in (
    select
      a.id
    from
      accounts a
      left join account_users au on a.id = au.account_id
      and au.user_id = sqlc.arg(user_id)::uuid
    where
      a.owner_id = sqlc.arg(user_id)::uuid
      or au.user_id is not null
  )
returning
  account_id;

-- name: CategorizeTransactionAtomic :one
update
  transactions
set
  category_id = sqlc.narg('category_id')::bigint,
  category_manually_set = sqlc.arg(category_manually_set)::boolean,
  suggestions = sqlc.arg(suggestions)::text []
where
  id = sqlc.arg(id)::bigint
  and category_manually_set = false -- Only update if not manually set
  and account_id in (
    select
      a.id
    from
      accounts a
      left join account_users au on a.id = au.account_id
      and au.user_id = sqlc.arg(user_id)::uuid
    where
      a.owner_id = sqlc.arg(user_id)::uuid
      or au.user_id is not null
  )
returning
  id,
  category_manually_set;

-- name: BulkCategorizeTransactions :execrows
update
  transactions
set
  category_id = sqlc.arg(category_id)::bigint,
  category_manually_set = true -- manual categorization
where
  id = ANY(sqlc.arg(transaction_ids)::bigint [])
  and account_id in (
    select
      a.id
    from
      accounts a
      left join account_users au on a.id = au.account_id
      and au.user_id = sqlc.arg(user_id)::uuid
    where
      a.owner_id = sqlc.arg(user_id)::uuid
      or au.user_id is not null
  );

-- name: BulkDeleteTransactions :execrows
delete from
  transactions
where
  id = ANY(sqlc.arg(transaction_ids)::bigint [])
  and account_id in (
    select
      a.id
    from
      accounts a
      left join account_users au on a.id = au.account_id
      and au.user_id = sqlc.arg(user_id)::uuid
    where
      a.owner_id = sqlc.arg(user_id)::uuid
      or au.user_id is not null
  );

-- name: GetTransactionCountByAccount :many
select
  a.id,
  a.name,
  COUNT(t.id) as transaction_count
from
  accounts a
  left join account_users au on a.id = au.account_id
  and au.user_id = sqlc.arg(user_id)::uuid
  left join transactions t on a.id = t.account_id
where
  a.owner_id = sqlc.arg(user_id)::uuid
  or au.user_id is not null
group by
  a.id,
  a.name
order by
  transaction_count desc;

-- name: RecalculateBalancesAfterTransaction :exec
-- Recalculate balance_after for all transactions after a given date/id
with transaction_deltas as (
  select
    id,
    SUM(
      case
        when tx_direction = 1 then (tx_amount ->> 'units')::bigint + (tx_amount ->> 'nanos')::bigint / 1000000000.0
        else -(
          (tx_amount ->> 'units')::bigint + (tx_amount ->> 'nanos')::bigint / 1000000000.0
        )
      end
    ) OVER (
      partition BY account_id
      order by
        tx_date,
        id
    ) as running_delta
  from
    transactions
  where
    account_id = @account_id::bigint
    and (
      tx_date > @from_date::timestamptz
      or (
        tx_date = @from_date::timestamptz
        and id >= @from_id::bigint
      )
    )
),
anchor_point as (
  select
    a.anchor_balance,
    COALESCE(
      SUM(
        case
          when t.tx_direction = 1 then (t.tx_amount ->> 'units')::bigint + (t.tx_amount ->> 'nanos')::bigint / 1000000000.0
          else -(
            (t.tx_amount ->> 'units')::bigint + (t.tx_amount ->> 'nanos')::bigint / 1000000000.0
          )
        end
      ),
      0.0
    ) as delta_at_anchor
  from
    accounts a
    left join transactions t on t.account_id = a.id
    and t.tx_date < a.anchor_date
  where
    a.id = @account_id::bigint
  group by
    a.id,
    a.anchor_balance
)
update
  transactions
set
  balance_after = jsonb_build_object(
    'currency_code',
    tx_amount ->> 'currency_code',
    'units',
    (
      (ap.anchor_balance ->> 'units')::bigint + td.running_delta - ap.delta_at_anchor
    )::bigint,
    'nanos',
    0
  )
from
  transaction_deltas td,
  anchor_point ap
where
  transactions.id = td.id
  and transactions.account_id = @account_id::bigint;

-- name: SyncAccountBalances :exec
with transaction_deltas as (
  select
    id,
    SUM(
      case
        when tx_direction = 1 then (tx_amount ->> 'units')::bigint + (tx_amount ->> 'nanos')::bigint / 1000000000.0
        else -(
          (tx_amount ->> 'units')::bigint + (tx_amount ->> 'nanos')::bigint / 1000000000.0
        )
      end
    ) OVER (
      partition BY account_id
      order by
        tx_date,
        id
    ) as running_delta
  from
    transactions
  where
    account_id = sqlc.arg(account_id)::bigint
),
anchor_point as (
  select
    a.anchor_balance,
    COALESCE(
      SUM(
        case
          when t.tx_direction = 1 then (t.tx_amount ->> 'units')::bigint + (t.tx_amount ->> 'nanos')::bigint / 1000000000.0
          else -(
            (t.tx_amount ->> 'units')::bigint + (t.tx_amount ->> 'nanos')::bigint / 1000000000.0
          )
        end
      ),
      0.0
    ) as delta_at_anchor
  from
    accounts a
    left join transactions t on t.account_id = a.id
    and t.tx_date < a.anchor_date
  where
    a.id = sqlc.arg(account_id)::bigint
  group by
    a.id,
    a.anchor_balance
)
update
  transactions
set
  balance_after = jsonb_build_object(
    'currency_code',
    tx_amount ->> 'currency_code',
    'units',
    (
      (ap.anchor_balance ->> 'units')::bigint + td.running_delta - ap.delta_at_anchor
    )::bigint,
    'nanos',
    0
  )
from
  transaction_deltas td,
  anchor_point ap
where
  transactions.id = td.id
  and transactions.account_id = sqlc.arg(account_id)::bigint;

-- name: FindCandidateTransactions :many
select
  t.*,
  similarity(t.tx_desc::text, sqlc.arg(merchant)::text) as merchant_score
from
  transactions t
  join accounts a on t.account_id = a.id
  left join account_users au on a.id = au.account_id
  and au.user_id = sqlc.arg(user_id)::uuid
where
  (
    a.owner_id = sqlc.arg(user_id)::uuid
    or au.user_id is not null
  )
  and t.tx_direction = 2
  and t.tx_date >= (sqlc.arg(date)::date - interval '60 days')
  and (t.tx_amount ->> 'units')::bigint + (t.tx_amount ->> 'nanos')::bigint / 1000000000.0 between sqlc.arg(total)::double precision and (sqlc.arg(total)::double precision * 1.20)
  and similarity(t.tx_desc::text, sqlc.arg(merchant)::text) > 0.3
order by
  merchant_score desc
limit
  10;

-- name: GetAccountIDsFromTransactionIDs :many
select
  distinct account_id
from
  transactions
where
  id = ANY(@ids::bigint []);

-- name: ListAllTransactions :many
select
  t.*
from
  transactions t
  join accounts a on t.account_id = a.id
  left join account_users au on a.id = au.account_id
  and au.user_id = sqlc.arg(user_id)::uuid
where
  (
    a.owner_id = sqlc.arg(user_id)::uuid
    or au.user_id is not null
  )
order by
  t.tx_date desc,
  t.id desc;
