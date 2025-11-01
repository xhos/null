-- name: GetDashboardTrends :many
select
  to_char(t.tx_date::date, 'YYYY-MM-DD') as date,
  SUM(case when t.tx_direction = 1 then amount_cents(t.tx_amount) else 0 end)::bigint as income_cents,
  SUM(case when t.tx_direction = 2 then amount_cents(t.tx_amount) else 0 end)::bigint as expense_cents
from transactions t
join accounts a on t.account_id = a.id
left join account_users au on a.id = au.account_id and au.user_id = @user_id::uuid
where (a.owner_id = @user_id::uuid or au.user_id is not null)
  and (sqlc.narg('start')::timestamptz is null or t.tx_date >= sqlc.narg('start')::timestamptz)
  and (sqlc.narg('end')::timestamptz is null or t.tx_date <= sqlc.narg('end')::timestamptz)
group by date
order by date;

-- name: GetDashboardSummary :one
select
  COUNT(distinct a.id)::bigint as total_accounts,
  COUNT(t.id)::bigint as total_transactions,
  COALESCE(SUM(case when t.tx_direction = 1 then amount_cents(t.tx_amount) else 0 end), 0)::bigint as total_income_cents,
  COALESCE(SUM(case when t.tx_direction = 2 then amount_cents(t.tx_amount) else 0 end), 0)::bigint as total_expense_cents,
  COUNT(distinct case when t.tx_date >= CURRENT_DATE - interval '30 days' then t.id end)::bigint as transactions_last_30_days,
  COUNT(distinct case when t.category_id is null then t.id end)::bigint as uncategorized_transactions
from accounts a
left join account_users au on a.id = au.account_id and au.user_id = @user_id::uuid
left join transactions t on a.id = t.account_id
where (a.owner_id = @user_id::uuid or au.user_id is not null)
  and (sqlc.narg('start')::timestamptz is null or t.tx_date >= sqlc.narg('start')::timestamptz)
  and (sqlc.narg('end')::timestamptz is null or t.tx_date <= sqlc.narg('end')::timestamptz);

-- name: GetTopCategories :many
select
  c.slug,
  c.color,
  COUNT(t.id)::bigint as transaction_count,
  SUM(amount_cents(t.tx_amount))::bigint as total_amount_cents
from transactions t
join categories c on t.category_id = c.id
join accounts a on t.account_id = a.id
left join account_users au on a.id = au.account_id and au.user_id = @user_id::uuid
where (a.owner_id = @user_id::uuid or au.user_id is not null)
  and t.tx_direction = 2
  and (sqlc.narg('start')::timestamptz is null or t.tx_date >= sqlc.narg('start')::timestamptz)
  and (sqlc.narg('end')::timestamptz is null or t.tx_date <= sqlc.narg('end')::timestamptz)
group by c.id, c.slug, c.color
order by total_amount_cents desc
limit COALESCE(sqlc.narg('limit')::int, 10);

-- name: GetTopMerchants :many
select
  t.merchant,
  COUNT(t.id)::bigint as transaction_count,
  SUM(amount_cents(t.tx_amount))::bigint as total_amount_cents,
  AVG(amount_cents(t.tx_amount))::bigint as avg_amount_cents
from transactions t
join accounts a on t.account_id = a.id
left join account_users au on a.id = au.account_id and au.user_id = @user_id::uuid
where (a.owner_id = @user_id::uuid or au.user_id is not null)
  and t.merchant is not null
  and t.tx_direction = 2
  and (sqlc.narg('start')::timestamptz is null or t.tx_date >= sqlc.narg('start')::timestamptz)
  and (sqlc.narg('end')::timestamptz is null or t.tx_date <= sqlc.narg('end')::timestamptz)
group by t.merchant
order by total_amount_cents desc
limit COALESCE(sqlc.narg('limit')::int, 10);

-- name: GetMonthlyComparison :many
select
  to_char(t.tx_date, 'YYYY-MM') as month,
  SUM(case when t.tx_direction = 1 then amount_cents(t.tx_amount) else 0 end)::bigint as income_cents,
  SUM(case when t.tx_direction = 2 then amount_cents(t.tx_amount) else 0 end)::bigint as expense_cents,
  SUM(case when t.tx_direction = 1 then amount_cents(t.tx_amount) else -amount_cents(t.tx_amount) end)::bigint as net_cents
from transactions t
join accounts a on t.account_id = a.id
left join account_users au on a.id = au.account_id and au.user_id = @user_id::uuid
where (a.owner_id = @user_id::uuid or au.user_id is not null)
  and t.tx_date >= COALESCE(sqlc.narg('start')::timestamptz, CURRENT_DATE - interval '12 months')
  and t.tx_date <= COALESCE(sqlc.narg('end')::timestamptz, CURRENT_DATE)
group by month
order by month;

-- name: GetAccountBalances :many
with balance_deltas as (
  select
    t.account_id,
    SUM(case when t.tx_direction = 1 then amount_cents(t.tx_amount) else -amount_cents(t.tx_amount) end) as delta_cents
  from transactions t
  join accounts a on t.account_id = a.id
  where t.tx_date > a.anchor_date
  group by t.account_id
)
select
  a.id,
  a.name,
  a.account_type,
  jsonb_build_object(
    'currency_code', a.anchor_balance->>'currency_code',
    'units', ((a.anchor_balance->>'units')::bigint + COALESCE(d.delta_cents, 0) / 100)::bigint,
    'nanos', (COALESCE(d.delta_cents, 0) % 100 * 10000000)::int
  ) as current_balance
from accounts a
left join account_users au on a.id = au.account_id and au.user_id = @user_id::uuid
left join balance_deltas d on a.id = d.account_id
where (a.owner_id = @user_id::uuid or au.user_id is not null)
order by (a.anchor_balance->>'units')::bigint + COALESCE(d.delta_cents, 0) / 100 desc;

-- name: GetDashboardSummaryForAccount :one
select
  COUNT(distinct a.id)::bigint as total_accounts,
  COUNT(t.id)::bigint as total_transactions,
  COALESCE(SUM(case when t.tx_direction = 1 then amount_cents(t.tx_amount) else 0 end), 0)::bigint as total_income_cents,
  COALESCE(SUM(case when t.tx_direction = 2 then amount_cents(t.tx_amount) else 0 end), 0)::bigint as total_expense_cents,
  COUNT(distinct case when t.tx_date >= CURRENT_DATE - interval '30 days' then t.id end)::bigint as transactions_last_30_days,
  COUNT(distinct case when t.category_id is null then t.id end)::bigint as uncategorized_transactions
from accounts a
left join account_users au on a.id = au.account_id and au.user_id = @user_id::uuid
left join transactions t on a.id = t.account_id
where (a.owner_id = @user_id::uuid or au.user_id is not null)
  and a.id = @account_id::bigint
  and (sqlc.narg('start')::timestamptz is null or t.tx_date >= sqlc.narg('start')::timestamptz)
  and (sqlc.narg('end')::timestamptz is null or t.tx_date <= sqlc.narg('end')::timestamptz);

-- name: GetDashboardTrendsForAccount :many
select
  to_char(t.tx_date::date, 'YYYY-MM-DD') as date,
  SUM(case when t.tx_direction = 1 then amount_cents(t.tx_amount) else 0 end)::bigint as income_cents,
  SUM(case when t.tx_direction = 2 then amount_cents(t.tx_amount) else 0 end)::bigint as expense_cents
from transactions t
join accounts a on t.account_id = a.id
left join account_users au on a.id = au.account_id and au.user_id = @user_id::uuid
where (a.owner_id = @user_id::uuid or au.user_id is not null)
  and a.id = @account_id::bigint
  and (sqlc.narg('start')::timestamptz is null or t.tx_date >= sqlc.narg('start')::timestamptz)
  and (sqlc.narg('end')::timestamptz is null or t.tx_date <= sqlc.narg('end')::timestamptz)
group by date
order by date;

-- name: GetCategorySpendingForPeriod :many
select
  t.category_id,
  c.id as category_db_id,
  c.slug as category_slug,
  c.color as category_color,
  COALESCE(SUM(amount_cents(t.tx_amount)), 0)::bigint as total_cents,
  COUNT(t.id)::bigint as transaction_count
from transactions t
join accounts a on t.account_id = a.id
left join account_users au on a.id = au.account_id and au.user_id = @user_id::uuid
left join categories c on t.category_id = c.id
where (a.owner_id = @user_id::uuid or au.user_id is not null)
  and t.tx_direction = 2
  and t.tx_date >= @start_date::timestamptz
  and t.tx_date <= @end_date::timestamptz
group by t.category_id, c.id, c.slug, c.color;
