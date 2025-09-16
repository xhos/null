-- name: ListReceipts :many
SELECT DISTINCT
  r.id, r.engine, r.parse_status, r.link_status, r.match_ids,
  r.merchant, r.purchase_date, r.total_amount, r.tax_amount,
  r.raw_payload, r.canonical_data, r.image_url, r.image_sha256,
  r.lat, r.lon, r.location_source, r.location_label,
  r.created_at, r.updated_at
FROM receipts r
LEFT JOIN transactions t ON r.id = t.receipt_id
LEFT JOIN accounts a ON t.account_id = a.id
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
WHERE a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL
ORDER BY r.created_at DESC;

-- name: GetReceipt :one
SELECT DISTINCT
  r.id, r.engine, r.parse_status, r.link_status, r.match_ids,
  r.merchant, r.purchase_date, r.total_amount, r.tax_amount,
  r.raw_payload, r.canonical_data, r.image_url, r.image_sha256,
  r.lat, r.lon, r.location_source, r.location_label,
  r.created_at, r.updated_at
FROM receipts r
LEFT JOIN transactions t ON r.id = t.receipt_id
LEFT JOIN accounts a ON t.account_id = a.id
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
WHERE r.id = @id::bigint
  AND (a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL);

-- name: CreateReceipt :one
INSERT INTO receipts (
  engine, parse_status, link_status, match_ids,
  merchant, purchase_date, total_amount, tax_amount,
  raw_payload, canonical_data, image_url, image_sha256,
  lat, lon, location_source, location_label
) VALUES (
  @engine::smallint,
  COALESCE(sqlc.narg('parse_status')::smallint, 1),
  COALESCE(sqlc.narg('link_status')::smallint, 1),
  sqlc.narg('match_ids')::bigint[],
  sqlc.narg('merchant')::text,
  sqlc.narg('purchase_date')::date,
  sqlc.narg('total_amount')::jsonb,
  sqlc.narg('tax_amount')::jsonb,
  sqlc.narg('raw_payload')::jsonb,
  sqlc.narg('canonical_data')::jsonb,
  sqlc.narg('image_url')::text,
  sqlc.narg('image_sha256')::bytea,
  sqlc.narg('lat')::double precision,
  sqlc.narg('lon')::double precision,
  sqlc.narg('location_source')::text,
  sqlc.narg('location_label')::text
)
RETURNING
  id, engine, parse_status, link_status, match_ids,
  merchant, purchase_date, total_amount, tax_amount,
  raw_payload, canonical_data, image_url, image_sha256,
  lat, lon, location_source, location_label,
  created_at, updated_at;

-- name: UpdateReceipt :execrows
UPDATE receipts
SET engine = COALESCE(sqlc.narg('engine')::smallint, engine),
    parse_status = COALESCE(sqlc.narg('parse_status')::smallint, parse_status),
    link_status = COALESCE(sqlc.narg('link_status')::smallint, link_status),
    match_ids = COALESCE(sqlc.narg('match_ids')::bigint[], match_ids),
    merchant = COALESCE(sqlc.narg('merchant')::text, merchant),
    purchase_date = COALESCE(sqlc.narg('purchase_date')::date, purchase_date),
    total_amount = COALESCE(sqlc.narg('total_amount')::jsonb, total_amount),
    tax_amount = COALESCE(sqlc.narg('tax_amount')::jsonb, tax_amount),
    raw_payload = COALESCE(sqlc.narg('raw_payload')::jsonb, raw_payload),
    canonical_data = COALESCE(sqlc.narg('canonical_data')::jsonb, canonical_data),
    image_url = COALESCE(sqlc.narg('image_url')::text, image_url),
    image_sha256 = COALESCE(sqlc.narg('image_sha256')::bytea, image_sha256),
    lat = COALESCE(sqlc.narg('lat')::double precision, lat),
    lon = COALESCE(sqlc.narg('lon')::double precision, lon),
    location_source = COALESCE(sqlc.narg('location_source')::text, location_source),
    location_label = COALESCE(sqlc.narg('location_label')::text, location_label)
WHERE id = @id::bigint;

-- name: DeleteReceipt :execrows
DELETE FROM receipts 
WHERE id = @id::bigint
  AND EXISTS (
    SELECT 1 FROM transactions t
    JOIN accounts a ON t.account_id = a.id
    LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
    WHERE t.receipt_id = receipts.id
      AND (a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL)
  );

-- Receipt Items CRUD
-- name: ListReceiptItemsForReceipt :many
SELECT
  id, receipt_id, line_no, name, qty, unit_price, line_total, sku, category_hint,
  created_at, updated_at
FROM receipt_items
WHERE receipt_id = @receipt_id::bigint
ORDER BY line_no NULLS LAST, id;

-- name: GetReceiptItem :one
SELECT
  id, receipt_id, line_no, name, qty, unit_price, line_total, sku, category_hint,
  created_at, updated_at
FROM receipt_items
WHERE id = @id::bigint;

-- name: CreateReceiptItem :one
INSERT INTO receipt_items (
  receipt_id, line_no, name, qty, unit_price, line_total, sku, category_hint
) VALUES (
  @receipt_id::bigint,
  sqlc.narg('line_no')::int,
  @name::text,
  COALESCE(sqlc.narg('qty')::numeric, 1),
  sqlc.narg('unit_price')::jsonb,
  sqlc.narg('line_total')::jsonb,
  sqlc.narg('sku')::text,
  sqlc.narg('category_hint')::text
)
RETURNING id, receipt_id, line_no, name, qty, unit_price, line_total, sku, category_hint,
          created_at, updated_at;

-- name: UpdateReceiptItem :one
UPDATE receipt_items
SET line_no = COALESCE(sqlc.narg('line_no')::int, line_no),
    name = COALESCE(sqlc.narg('name')::text, name),
    qty = COALESCE(sqlc.narg('qty')::numeric, qty),
    unit_price = COALESCE(sqlc.narg('unit_price')::jsonb, unit_price),
    line_total = COALESCE(sqlc.narg('line_total')::jsonb, line_total),
    sku = COALESCE(sqlc.narg('sku')::text, sku),
    category_hint = COALESCE(sqlc.narg('category_hint')::text, category_hint)
WHERE id = @id::bigint
RETURNING id, receipt_id, line_no, name, qty, unit_price, line_total, sku, category_hint,
          created_at, updated_at;

-- name: DeleteReceiptItem :execrows
DELETE FROM receipt_items WHERE id = @id::bigint;

-- name: BulkCreateReceiptItems :copyfrom
INSERT INTO receipt_items (
  receipt_id, line_no, name, qty, unit_price, line_total, sku, category_hint
) VALUES (
  @receipt_id, @line_no, @name, @qty, @unit_price, @line_total, @sku, @category_hint
);

-- name: DeleteReceiptItemsByReceipt :execrows
DELETE FROM receipt_items WHERE receipt_id = @receipt_id::bigint;

-- Utility queries
-- name: GetUnlinkedReceipts :many
SELECT id, merchant, purchase_date, total_amount, created_at
FROM receipts
WHERE link_status = 1  -- unlinked
ORDER BY created_at DESC
LIMIT COALESCE(sqlc.narg('limit')::int, 50);

-- name: GetReceiptMatchCandidates :many
SELECT r.id, r.merchant, r.purchase_date, r.total_amount,
       COUNT(t.id) AS potential_matches
FROM receipts r
LEFT JOIN transactions t ON t.id = ANY(r.match_ids)
WHERE r.link_status = 3  -- needs verification
GROUP BY r.id, r.merchant, r.purchase_date, r.total_amount
ORDER BY r.created_at DESC;

-- name: LinkTransactionToReceipt :exec
UPDATE transactions 
SET receipt_id = @receipt_id::bigint
WHERE id = @transaction_id::bigint
  AND receipt_id IS NULL;
