-- name: ListTransactions :many
SELECT
  t.id, t.email_id, t.account_id, t.tx_date, t.tx_amount,
  t.tx_direction, t.tx_desc, t.balance_after, t.category_id, t.cat_status,
  t.merchant, t.user_notes, t.suggestions, t.receipt_id,
  t.foreign_amount, t.exchange_rate,
  t.created_at, t.updated_at,
  c.slug AS category_slug,
  c.label AS category_label,
  c.color AS category_color,
  a.name AS account_name
FROM transactions t
LEFT JOIN categories c ON t.category_id = c.id
JOIN accounts a ON t.account_id = a.id
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = sqlc.arg(user_id)::uuid
WHERE (a.owner_id = sqlc.arg(user_id)::uuid OR au.user_id IS NOT NULL)
  AND (
        sqlc.narg('cursor_date')::timestamptz IS NULL
        OR sqlc.narg('cursor_id')::bigint IS NULL
        OR (t.tx_date, t.id) < (sqlc.narg('cursor_date')::timestamptz, sqlc.narg('cursor_id')::bigint)
      )
  AND (sqlc.narg('start')::timestamptz IS NULL OR t.tx_date >= sqlc.narg('start')::timestamptz)
  AND (sqlc.narg('end')::timestamptz IS NULL OR t.tx_date <= sqlc.narg('end')::timestamptz)
  AND (sqlc.narg('amount_min')::numeric IS NULL OR (t.tx_amount->>'units')::bigint >= sqlc.narg('amount_min')::numeric)
  AND (sqlc.narg('amount_max')::numeric IS NULL OR (t.tx_amount->>'units')::bigint <= sqlc.narg('amount_max')::numeric)
  AND (sqlc.narg('direction')::smallint IS NULL OR t.tx_direction = sqlc.narg('direction')::smallint)
  AND (sqlc.narg('account_ids')::bigint[] IS NULL OR t.account_id = ANY(sqlc.narg('account_ids')::bigint[]))
  AND (sqlc.narg('categories')::text[] IS NULL OR c.slug = ANY(sqlc.narg('categories')::text[]))
  AND (sqlc.narg('merchant_q')::text IS NULL OR t.merchant ILIKE ('%' || sqlc.narg('merchant_q')::text || '%'))
  AND (sqlc.narg('desc_q')::text IS NULL OR t.tx_desc ILIKE ('%' || sqlc.narg('desc_q')::text || '%'))
  AND (sqlc.narg('currency')::char(3) IS NULL OR t.tx_amount->>'currency_code' = sqlc.narg('currency')::char(3))
  AND (sqlc.narg('tod_start')::time IS NULL OR t.tx_date::time >= sqlc.narg('tod_start')::time)
  AND (sqlc.narg('tod_end')::time IS NULL OR t.tx_date::time <= sqlc.narg('tod_end')::time)
  AND (sqlc.narg('uncategorized')::boolean IS NULL OR (sqlc.narg('uncategorized')::boolean = true AND t.category_id IS NULL))
ORDER BY t.tx_date DESC, t.id DESC
LIMIT COALESCE(sqlc.narg('limit')::int, 100);

-- name: GetTransaction :one
SELECT
  t.id, t.email_id, t.account_id, t.tx_date, t.tx_amount,
  t.tx_direction, t.tx_desc, t.balance_after, t.category_id, t.cat_status,
  t.merchant, t.user_notes, t.suggestions, t.receipt_id,
  t.foreign_amount, t.exchange_rate,
  t.created_at, t.updated_at,
  c.slug AS category_slug,
  c.label AS category_label,
  c.color AS category_color,
  a.name AS account_name
FROM transactions t
LEFT JOIN categories c ON t.category_id = c.id
JOIN accounts a ON t.account_id = a.id
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = sqlc.arg(user_id)::uuid
WHERE t.id = sqlc.arg(id)::bigint
  AND (a.owner_id = sqlc.arg(user_id)::uuid OR au.user_id IS NOT NULL);

-- name: CreateTransaction :one
INSERT INTO transactions (
  email_id, account_id, tx_date, tx_amount, tx_direction,
  tx_desc, balance_after, category_id, merchant, user_notes,
  foreign_amount, exchange_rate, suggestions, receipt_id
)
SELECT
  sqlc.narg('email_id')::text,
  sqlc.arg(account_id)::bigint,
  sqlc.arg(tx_date)::timestamptz,
  sqlc.arg(tx_amount)::jsonb,
  sqlc.arg(tx_direction)::smallint,
  sqlc.narg('tx_desc')::text,
  sqlc.narg('balance_after')::jsonb,
  sqlc.narg('category_id')::bigint,
  sqlc.narg('merchant')::text,
  sqlc.narg('user_notes')::text,
  sqlc.narg('foreign_amount')::jsonb,
  sqlc.narg('exchange_rate')::numeric,
  sqlc.narg('suggestions')::text[],
  sqlc.narg('receipt_id')::bigint
FROM accounts a
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = sqlc.arg(user_id)::uuid
WHERE a.id = sqlc.arg(account_id)::bigint
  AND (a.owner_id = sqlc.arg(user_id)::uuid OR au.user_id IS NOT NULL)
RETURNING id;

-- name: UpdateTransaction :one
UPDATE transactions
SET email_id = COALESCE(sqlc.narg('email_id')::text, email_id),
    tx_date = COALESCE(sqlc.narg('tx_date')::timestamptz, tx_date),
    tx_amount = COALESCE(sqlc.narg('tx_amount')::jsonb, tx_amount),
    tx_direction = COALESCE(sqlc.narg('tx_direction')::smallint, tx_direction),
    tx_desc = COALESCE(sqlc.narg('tx_desc')::text, tx_desc),
    category_id = COALESCE(sqlc.narg('category_id')::bigint, category_id),
    merchant = COALESCE(sqlc.narg('merchant')::text, merchant),
    user_notes = COALESCE(sqlc.narg('user_notes')::text, user_notes),
    foreign_amount = COALESCE(sqlc.narg('foreign_amount')::jsonb, foreign_amount),
    exchange_rate = COALESCE(sqlc.narg('exchange_rate')::numeric, exchange_rate),
    suggestions = COALESCE(sqlc.narg('suggestions')::text[], suggestions),
    receipt_id = COALESCE(sqlc.narg('receipt_id')::bigint, receipt_id),
    cat_status = COALESCE(sqlc.narg('cat_status')::smallint, cat_status)
WHERE id = sqlc.arg(id)::bigint
  AND account_id IN (
    SELECT a.id FROM accounts a
    LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = sqlc.arg(user_id)::uuid
    WHERE a.owner_id = sqlc.arg(user_id)::uuid OR au.user_id IS NOT NULL
  )
RETURNING account_id;

-- name: DeleteTransaction :one
DELETE FROM transactions
WHERE id = sqlc.arg(id)::bigint
  AND account_id IN (
    SELECT a.id FROM accounts a
    LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = sqlc.arg(user_id)::uuid
    WHERE a.owner_id = sqlc.arg(user_id)::uuid OR au.user_id IS NOT NULL
  )
RETURNING account_id;

-- name: SetTransactionReceipt :execrows
UPDATE transactions
SET receipt_id = sqlc.arg(receipt_id)::bigint
WHERE id = sqlc.arg(id)::bigint AND receipt_id IS NULL;

-- name: CategorizeTransactionAtomic :one
UPDATE transactions
SET category_id = sqlc.narg('category_id')::bigint,
    cat_status = sqlc.arg(cat_status)::smallint,
    suggestions = sqlc.arg(suggestions)::text[]
WHERE id = sqlc.arg(id)::bigint
  AND cat_status = 0  -- Only update if still uncategorized
  AND account_id IN (
    SELECT a.id FROM accounts a
    LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = sqlc.arg(user_id)::uuid
    WHERE a.owner_id = sqlc.arg(user_id)::uuid OR au.user_id IS NOT NULL
  )
RETURNING id, cat_status;

-- name: BulkCategorizeTransactions :execrows
UPDATE transactions
SET category_id = sqlc.arg(category_id)::bigint,
    cat_status = 3  -- manual categorization
WHERE id = ANY(sqlc.arg(transaction_ids)::bigint[])
  AND account_id IN (
    SELECT a.id FROM accounts a
    LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = sqlc.arg(user_id)::uuid
    WHERE a.owner_id = sqlc.arg(user_id)::uuid OR au.user_id IS NOT NULL
  );

-- name: BulkDeleteTransactions :execrows
DELETE FROM transactions
WHERE id = ANY(sqlc.arg(transaction_ids)::bigint[])
  AND account_id IN (
    SELECT a.id FROM accounts a
    LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = sqlc.arg(user_id)::uuid
    WHERE a.owner_id = sqlc.arg(user_id)::uuid OR au.user_id IS NOT NULL
  );

-- name: GetTransactionCountByAccount :many
SELECT a.id, a.name, COUNT(t.id) AS transaction_count
FROM accounts a
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = sqlc.arg(user_id)::uuid
LEFT JOIN transactions t ON a.id = t.account_id
WHERE a.owner_id = sqlc.arg(user_id)::uuid OR au.user_id IS NOT NULL
GROUP BY a.id, a.name
ORDER BY transaction_count DESC;


-- name: RecalculateBalancesAfterTransaction :exec
-- Recalculate balance_after for all transactions after a given date/id
WITH transaction_deltas AS (
  SELECT id,
         SUM(CASE WHEN tx_direction = 1 THEN (tx_amount->>'units')::bigint + (tx_amount->>'nanos')::bigint/1000000000.0 
                  ELSE -((tx_amount->>'units')::bigint + (tx_amount->>'nanos')::bigint/1000000000.0) END)
           OVER (PARTITION BY account_id ORDER BY tx_date, id) AS running_delta
  FROM transactions
  WHERE account_id = @account_id::bigint
    AND (tx_date > @from_date::timestamptz OR (tx_date = @from_date::timestamptz AND id >= @from_id::bigint))
),
anchor_point AS (
  SELECT a.anchor_balance,
         COALESCE(SUM(CASE WHEN t.tx_direction = 1 THEN (t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0 
                           ELSE -((t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0) END), 0.0) AS delta_at_anchor
  FROM accounts a
  LEFT JOIN transactions t ON t.account_id = a.id AND t.tx_date < a.anchor_date
  WHERE a.id = @account_id::bigint
  GROUP BY a.id, a.anchor_balance
)
UPDATE transactions
SET balance_after = jsonb_build_object(
  'currency_code', tx_amount->>'currency_code',
  'units', ((ap.anchor_balance->>'units')::bigint + td.running_delta - ap.delta_at_anchor)::bigint,
  'nanos', 0
)
FROM transaction_deltas td, anchor_point ap
WHERE transactions.id = td.id
  AND transactions.account_id = @account_id::bigint;

-- name: SyncAccountBalances :exec
WITH transaction_deltas AS (
  SELECT id,
         SUM(CASE WHEN tx_direction = 1 THEN (tx_amount->>'units')::bigint + (tx_amount->>'nanos')::bigint/1000000000.0 
                  ELSE -((tx_amount->>'units')::bigint + (tx_amount->>'nanos')::bigint/1000000000.0) END)
           OVER (PARTITION BY account_id ORDER BY tx_date, id) AS running_delta
  FROM transactions
  WHERE account_id = sqlc.arg(account_id)::bigint
),
anchor_point AS (
  SELECT a.anchor_balance,
         COALESCE(SUM(CASE WHEN t.tx_direction = 1 THEN (t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0 
                           ELSE -((t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0) END), 0.0) AS delta_at_anchor
  FROM accounts a
  LEFT JOIN transactions t ON t.account_id = a.id AND t.tx_date < a.anchor_date
  WHERE a.id = sqlc.arg(account_id)::bigint
  GROUP BY a.id, a.anchor_balance
)
UPDATE transactions
SET balance_after = jsonb_build_object(
  'currency_code', tx_amount->>'currency_code',
  'units', ((ap.anchor_balance->>'units')::bigint + td.running_delta - ap.delta_at_anchor)::bigint,
  'nanos', 0
)
FROM transaction_deltas td, anchor_point ap
WHERE transactions.id = td.id
  AND transactions.account_id = sqlc.arg(account_id)::bigint;

-- name: FindCandidateTransactions :many
SELECT
  t.id, t.email_id, t.account_id, t.tx_date, t.tx_amount,
  t.tx_direction, t.tx_desc, t.balance_after, t.category_id, t.cat_status,
  t.merchant, t.user_notes, t.suggestions, t.receipt_id,
  t.foreign_amount, t.exchange_rate,
  t.created_at, t.updated_at,
  c.slug AS category_slug,
  c.label AS category_label,
  c.color AS category_color,
  similarity(t.tx_desc::text, sqlc.arg(merchant)::text) AS merchant_score
FROM transactions t
LEFT JOIN categories c ON t.category_id = c.id
JOIN accounts a ON t.account_id = a.id
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = sqlc.arg(user_id)::uuid
WHERE (a.owner_id = sqlc.arg(user_id)::uuid OR au.user_id IS NOT NULL)
  AND t.receipt_id IS NULL
  AND t.tx_direction = 2
  AND t.tx_date >= (sqlc.arg(date)::date - interval '60 days')
  AND (t.tx_amount->>'units')::bigint + (t.tx_amount->>'nanos')::bigint/1000000000.0 BETWEEN sqlc.arg(total)::numeric AND (sqlc.arg(total)::numeric * 1.20)
  AND similarity(t.tx_desc::text, sqlc.arg(merchant)::text) > 0.3
ORDER BY merchant_score DESC
LIMIT 10;

-- name: GetAccountIDsFromTransactionIDs :many
SELECT DISTINCT account_id
FROM transactions
WHERE id = ANY(@ids::bigint[]);
