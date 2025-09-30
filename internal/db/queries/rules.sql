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