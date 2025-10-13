-- name: ListReceipts :many
select
  distinct r.id,
  r.engine,
  r.parse_status,
  r.link_status,
  r.match_ids,
  r.merchant,
  r.purchase_date,
  r.total_amount,
  r.tax_amount,
  r.raw_payload,
  r.canonical_data,
  r.image_url,
  r.image_sha256,
  r.lat,
  r.lon,
  r.location_source,
  r.location_label,
  r.created_at,
  r.updated_at
from
  receipts r
  left join transactions t on r.id = t.receipt_id
  left join accounts a on t.account_id = a.id
  left join account_users au on a.id = au.account_id
  and au.user_id = @user_id::uuid
where
  a.owner_id = @user_id::uuid
  or au.user_id is not null
order by
  r.created_at desc;

-- name: GetReceipt :one
select
  distinct r.id,
  r.engine,
  r.parse_status,
  r.link_status,
  r.match_ids,
  r.merchant,
  r.purchase_date,
  r.total_amount,
  r.tax_amount,
  r.raw_payload,
  r.canonical_data,
  r.image_url,
  r.image_sha256,
  r.lat,
  r.lon,
  r.location_source,
  r.location_label,
  r.created_at,
  r.updated_at
from
  receipts r
  left join transactions t on r.id = t.receipt_id
  left join accounts a on t.account_id = a.id
  left join account_users au on a.id = au.account_id
  and au.user_id = @user_id::uuid
where
  r.id = @id::bigint
  and (
    a.owner_id = @user_id::uuid
    or au.user_id is not null
  );

-- name: CreateReceipt :one
insert into
  receipts (
    engine,
    parse_status,
    link_status,
    match_ids,
    merchant,
    purchase_date,
    total_amount,
    tax_amount,
    raw_payload,
    canonical_data,
    image_url,
    image_sha256,
    lat,
    lon,
    location_source,
    location_label
  )
values
  (
    @engine::smallint,
    COALESCE(sqlc.narg('parse_status')::smallint, 1),
    COALESCE(sqlc.narg('link_status')::smallint, 1),
    sqlc.narg('match_ids')::bigint [],
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
returning
  id,
  engine,
  parse_status,
  link_status,
  match_ids,
  merchant,
  purchase_date,
  total_amount,
  tax_amount,
  raw_payload,
  canonical_data,
  image_url,
  image_sha256,
  lat,
  lon,
  location_source,
  location_label,
  created_at,
  updated_at;

-- name: UpdateReceipt :execrows
update
  receipts
set
  engine = sqlc.narg('engine')::smallint,
  parse_status = sqlc.narg('parse_status')::smallint,
  link_status = sqlc.narg('link_status')::smallint,
  match_ids = sqlc.narg('match_ids')::bigint [],
  merchant = sqlc.narg('merchant')::text,
  purchase_date = sqlc.narg('purchase_date')::date,
  total_amount = sqlc.narg('total_amount')::jsonb,
  tax_amount = sqlc.narg('tax_amount')::jsonb,
  raw_payload = sqlc.narg('raw_payload')::jsonb,
  canonical_data = sqlc.narg('canonical_data')::jsonb,
  image_url = sqlc.narg('image_url')::text,
  image_sha256 = sqlc.narg('image_sha256')::bytea,
  lat = sqlc.narg('lat')::double precision,
  lon = sqlc.narg('lon')::double precision,
  location_source = sqlc.narg('location_source')::text,
  location_label = sqlc.narg('location_label')::text
where
  id = @id::bigint;

-- name: DeleteReceipt :execrows
delete from
  receipts
where
  id = @id::bigint
  and exists (
    select
      1
    from
      transactions t
      join accounts a on t.account_id = a.id
      left join account_users au on a.id = au.account_id
      and au.user_id = @user_id::uuid
    where
      t.receipt_id = receipts.id
      and (
        a.owner_id = @user_id::uuid
        or au.user_id is not null
      )
  );

-- Receipt Items CRUD
-- name: ListReceiptItemsForReceipt :many
select
  id,
  receipt_id,
  line_no,
  name,
  qty,
  unit_price,
  line_total,
  sku,
  category_hint,
  created_at,
  updated_at
from
  receipt_items
where
  receipt_id = @receipt_id::bigint
order by
  line_no NULLS LAST,
  id;

-- name: GetReceiptItem :one
select
  id,
  receipt_id,
  line_no,
  name,
  qty,
  unit_price,
  line_total,
  sku,
  category_hint,
  created_at,
  updated_at
from
  receipt_items
where
  id = @id::bigint;

-- name: CreateReceiptItem :one
insert into
  receipt_items (
    receipt_id,
    line_no,
    name,
    qty,
    unit_price,
    line_total,
    sku,
    category_hint
  )
values
  (
    @receipt_id::bigint,
    sqlc.narg('line_no')::int,
    @name::text,
    COALESCE(sqlc.narg('qty')::int, 1),
    sqlc.narg('unit_price')::jsonb,
    sqlc.narg('line_total')::jsonb,
    sqlc.narg('sku')::text,
    sqlc.narg('category_hint')::text
  )
returning
  id,
  receipt_id,
  line_no,
  name,
  qty,
  unit_price,
  line_total,
  sku,
  category_hint,
  created_at,
  updated_at;

-- name: UpdateReceiptItem :one
update
  receipt_items
set
  line_no = sqlc.narg('line_no')::int,
  name = sqlc.narg('name')::text,
  qty = sqlc.narg('qty')::int,
  unit_price = sqlc.narg('unit_price')::jsonb,
  line_total = sqlc.narg('line_total')::jsonb,
  sku = sqlc.narg('sku')::text,
  category_hint = sqlc.narg('category_hint')::text
where
  id = @id::bigint
returning
  id,
  receipt_id,
  line_no,
  name,
  qty,
  unit_price,
  line_total,
  sku,
  category_hint,
  created_at,
  updated_at;

-- name: DeleteReceiptItem :execrows
delete from
  receipt_items
where
  id = @id::bigint;

-- name: BulkCreateReceiptItems :copyfrom
insert into
  receipt_items (
    receipt_id,
    line_no,
    name,
    qty,
    unit_price,
    line_total,
    sku,
    category_hint
  )
values
  (
    @receipt_id,
    @line_no,
    @name,
    @qty,
    @unit_price,
    @line_total,
    @sku,
    @category_hint
  );

-- name: DeleteReceiptItemsByReceipt :execrows
delete from
  receipt_items
where
  receipt_id = @receipt_id::bigint;

-- Utility queries
-- name: GetUnlinkedReceipts :many
select
  *
from
  receipts
where
  link_status = 1 -- unlinked
order by
  created_at desc
limit
  COALESCE(sqlc.narg('limit')::int, 50);

-- name: GetReceiptMatchCandidates :many
select
  r.id,
  r.merchant,
  r.purchase_date,
  r.total_amount,
  COUNT(t.id) as potential_matches
from
  receipts r
  left join transactions t on t.id = ANY(r.match_ids)
where
  r.link_status = 3 -- needs verification
group by
  r.id,
  r.merchant,
  r.purchase_date,
  r.total_amount
order by
  r.created_at desc;

-- name: LinkTransactionToReceipt :exec
update
  transactions
set
  receipt_id = @receipt_id::bigint
where
  id = @transaction_id::bigint
  and receipt_id is null;
