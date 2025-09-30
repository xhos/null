-- name: ListRules :many
select *
from transaction_rules
where user_id = @user_id::uuid
order by priority_order, created_at;

-- name: GetRule :one
select *
from transaction_rules
where rule_id = @rule_id::uuid
  and user_id = @user_id::uuid;

-- name: CreateRule :one
insert into transaction_rules (user_id, rule_name, category_id, conditions, merchant)
values (@user_id::uuid, @rule_name::text, @category_id::bigint, @conditions::jsonb, @merchant::text)
returning *;

-- name: UpdateRule :one
update transaction_rules
set
  rule_name = sqlc.narg('rule_name'),
  category_id = sqlc.narg('category_id')::bigint,
  conditions = sqlc.narg('conditions'),
  is_active = sqlc.narg('is_active'),
  priority_order = sqlc.narg('priority_order'),
  merchant = sqlc.narg('merchant'),
  updated_at = now()
where rule_id = @rule_id::uuid
  and user_id = @user_id::uuid
returning *;

-- name: DeleteRule :execrows
delete from transaction_rules
where rule_id = @rule_id::uuid
  and user_id = @user_id::uuid;

-- name: GetActiveRules :many
select *
from transaction_rules
where user_id = @user_id::uuid
  and (is_active is null or is_active = true)
order by priority_order, created_at;

-- name: GetTransactionsForRuleApplication :many
select
  t.id,
  t.account_id,
  t.tx_date,
  t.tx_amount,
  t.tx_direction,
  t.tx_desc,
  t.merchant,
  t.category_id,
  t.category_manually_set,
  t.merchant_manually_set,
  a.account_type,
  a.bank,
  a.name as account_name
from transactions t
join accounts a on t.account_id = a.id
left join account_users au on a.id = au.account_id and au.user_id = @user_id::uuid
where (a.owner_id = @user_id::uuid or au.user_id is not null)
  and (sqlc.narg('transaction_ids')::bigint[] is null or t.id = ANY(sqlc.narg('transaction_ids')::bigint[]))
  and (sqlc.narg('include_manually_set')::boolean = true or (t.category_manually_set = false and t.merchant_manually_set = false));

-- name: BulkApplyRuleToTransactions :execrows
update transactions
set
  category_id = coalesce(@category_id::bigint, category_id),
  merchant = coalesce(@merchant::text, merchant)
where id = ANY(@transaction_ids::bigint[])
  and account_id in (
    select a.id
    from accounts a
    left join account_users au on a.id = au.account_id and au.user_id = @user_id::uuid
    where a.owner_id = @user_id::uuid or au.user_id is not null
  )
  and category_manually_set = false
  and merchant_manually_set = false;