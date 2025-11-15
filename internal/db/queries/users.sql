-- name: ListUsers :many
select * from users
order by created_at desc;

-- name: GetUser :one
select * from users
where id = @id::uuid;

-- name: GetUserByEmail :one
select * from users
where lower(email) = lower(@email::text);

-- name: CreateUser :one
insert into users (id, email, display_name)
values (@id::uuid, @email::text, sqlc.narg('display_name')::text)
returning *;

-- name: UpdateUser :one
update users
set
  email = sqlc.narg('email')::text,
  display_name = sqlc.narg('display_name')::text,
  default_account_id = sqlc.narg('default_account_id')::bigint,
  primary_currency = coalesce(sqlc.narg('primary_currency')::varchar(3), primary_currency),
  timezone = coalesce(sqlc.narg('timezone')::varchar(50), timezone)
where id = @id::uuid
returning *;

-- name: UpdateUserDisplayName :one
update users
set display_name = @display_name::text
where id = @id::uuid
returning *;

-- name: SetUserDefaultAccount :one
update users
set default_account_id = @default_account_id::bigint
where id = @id::uuid
returning *;

-- name: DeleteUser :execrows
delete from
  users
where
  id = @id::uuid;

-- name: DeleteUserWithCascade :execrows
with removed_from_accounts as (
  delete from
    account_users
  where
    user_id = @id::uuid
  returning
    user_id
)
delete from
  users
where
  id = @id::uuid;

-- name: CheckUserExists :one
select
  exists(
    select
      1
    from
      users
    where
      id = @id::uuid
  ) as exists;

-- name: GetUserFirstAccount :one
select
  id
from
  accounts
where
  owner_id = @user_id::uuid
order by
  created_at asc
limit
  1;

-- name: GetUserPrimaryCurrency :one
select
  primary_currency
from
  users
where
  id = @id::uuid;

-- name: GetUserTimezone :one
select
  timezone
from
  users
where
  id = @id::uuid;
