
-- name: GetDashboardTrends :many
SELECT
  to_char(t.tx_date::date, 'YYYY-MM-DD') AS date,
  SUM(CASE WHEN t.tx_direction = 1 THEN (t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0 ELSE 0 END) AS income,
  SUM(CASE WHEN t.tx_direction = 2 THEN (t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0 ELSE 0 END) AS expenses
FROM transactions t
JOIN accounts a ON t.account_id = a.id
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
WHERE (a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL)
  AND (sqlc.narg('start')::timestamptz IS NULL OR t.tx_date >= sqlc.narg('start')::timestamptz)
  AND (sqlc.narg('end')::timestamptz IS NULL OR t.tx_date <= sqlc.narg('end')::timestamptz)
GROUP BY date
ORDER BY date;

-- name: GetDashboardSummary :one
SELECT
  COUNT(DISTINCT a.id) AS total_accounts,
  COUNT(t.id) AS total_transactions,
  COALESCE(SUM(CASE WHEN t.tx_direction = 1 THEN (t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0 ELSE 0 END), 0) AS total_income,
  COALESCE(SUM(CASE WHEN t.tx_direction = 2 THEN (t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0 ELSE 0 END), 0) AS total_expenses,
  COUNT(DISTINCT CASE WHEN t.tx_date >= CURRENT_DATE - INTERVAL '30 days' THEN t.id END) AS transactions_last_30_days,
  COUNT(DISTINCT CASE WHEN t.category_id IS NULL THEN t.id END) AS uncategorized_transactions
FROM accounts a
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
LEFT JOIN transactions t ON a.id = t.account_id
WHERE (a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL)
  AND (sqlc.narg('start')::timestamptz IS NULL OR t.tx_date >= sqlc.narg('start')::timestamptz)
  AND (sqlc.narg('end')::timestamptz IS NULL OR t.tx_date <= sqlc.narg('end')::timestamptz);

-- name: GetTopCategories :many
SELECT
  c.slug,
  c.label,
  c.color,
  COUNT(t.id) AS transaction_count,
  SUM((t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0) AS total_amount
FROM transactions t
JOIN categories c ON t.category_id = c.id
JOIN accounts a ON t.account_id = a.id
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
WHERE (a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL)
  AND t.tx_direction = 2  -- expenses only
  AND (sqlc.narg('start')::timestamptz IS NULL OR t.tx_date >= sqlc.narg('start')::timestamptz)
  AND (sqlc.narg('end')::timestamptz IS NULL OR t.tx_date <= sqlc.narg('end')::timestamptz)
GROUP BY c.id, c.slug, c.label, c.color
ORDER BY total_amount DESC
LIMIT COALESCE(sqlc.narg('limit')::int, 10);

-- name: GetTopMerchants :many
SELECT
  t.merchant,
  COUNT(t.id) AS transaction_count,
  SUM((t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0) AS total_amount,
  AVG((t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0) AS avg_amount
FROM transactions t
JOIN accounts a ON t.account_id = a.id
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
WHERE (a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL)
  AND t.merchant IS NOT NULL
  AND t.tx_direction = 2  -- expenses only
  AND (sqlc.narg('start')::timestamptz IS NULL OR t.tx_date >= sqlc.narg('start')::timestamptz)
  AND (sqlc.narg('end')::timestamptz IS NULL OR t.tx_date <= sqlc.narg('end')::timestamptz)
GROUP BY t.merchant
ORDER BY total_amount DESC
LIMIT COALESCE(sqlc.narg('limit')::int, 10);

-- name: GetMonthlyComparison :many
SELECT
  to_char(t.tx_date, 'YYYY-MM') AS month,
  SUM(CASE WHEN t.tx_direction = 1 THEN (t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0 ELSE 0 END) AS income,
  SUM(CASE WHEN t.tx_direction = 2 THEN (t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0 ELSE 0 END) AS expenses,
  SUM(CASE WHEN t.tx_direction = 1 THEN (t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0 ELSE -((t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0) END) AS net
FROM transactions t
JOIN accounts a ON t.account_id = a.id
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
WHERE (a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL)
  AND t.tx_date >= COALESCE(sqlc.narg('start')::timestamptz, CURRENT_DATE - INTERVAL '12 months')
  AND t.tx_date <= COALESCE(sqlc.narg('end')::timestamptz, CURRENT_DATE)
GROUP BY month
ORDER BY month;

-- name: GetAccountBalances :many
SELECT
  a.id,
  a.name,
  a.account_type,
  jsonb_build_object(
    'currency_code', a.anchor_balance->>'currency_code',
    'units', ((a.anchor_balance->>'units')::bigint + COALESCE(d.delta, 0))::bigint,
    'nanos', 0
  ) AS current_balance
FROM accounts a
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
LEFT JOIN LATERAL (
  SELECT SUM(
    CASE
      WHEN t.tx_direction = 1 THEN (t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0
      WHEN t.tx_direction = 2 THEN -((t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0)
    END
  ) AS delta
  FROM transactions t
  WHERE t.account_id = a.id
    AND t.tx_date > a.anchor_date
) d ON TRUE
WHERE (a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL)
ORDER BY current_balance DESC;

-- name: GetDashboardSummaryForAccount :one
SELECT
  COUNT(DISTINCT a.id) AS total_accounts,
  COUNT(t.id) AS total_transactions,
  COALESCE(SUM(CASE WHEN t.tx_direction = 1 THEN (t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0 ELSE 0 END), 0) AS total_income,
  COALESCE(SUM(CASE WHEN t.tx_direction = 2 THEN (t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0 ELSE 0 END), 0) AS total_expenses,
  COUNT(DISTINCT CASE WHEN t.tx_date >= CURRENT_DATE - INTERVAL '30 days' THEN t.id END) AS transactions_last_30_days,
  COUNT(DISTINCT CASE WHEN t.category_id IS NULL THEN t.id END) AS uncategorized_transactions
FROM accounts a
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
LEFT JOIN transactions t ON a.id = t.account_id
WHERE (a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL)
  AND a.id = @account_id::bigint
  AND (sqlc.narg('start')::timestamptz IS NULL OR t.tx_date >= sqlc.narg('start')::timestamptz)
  AND (sqlc.narg('end')::timestamptz IS NULL OR t.tx_date <= sqlc.narg('end')::timestamptz);

-- name: GetDashboardTrendsForAccount :many
SELECT
  to_char(t.tx_date::date, 'YYYY-MM-DD') AS date,
  SUM(CASE WHEN t.tx_direction = 1 THEN (t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0 ELSE 0 END) AS income,
  SUM(CASE WHEN t.tx_direction = 2 THEN (t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0 ELSE 0 END) AS expenses
FROM transactions t
JOIN accounts a ON t.account_id = a.id
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
WHERE (a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL)
  AND a.id = @account_id::bigint
  AND (sqlc.narg('start')::timestamptz IS NULL OR t.tx_date >= sqlc.narg('start')::timestamptz)
  AND (sqlc.narg('end')::timestamptz IS NULL OR t.tx_date <= sqlc.narg('end')::timestamptz)
GROUP BY date
ORDER BY date;
