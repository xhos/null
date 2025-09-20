-- name: AddAccountCollaborator :one
insert into
  account_users (account_id, user_id)
select
  @account_id::bigint,
  @collaborator_user_id::uuid
from
  accounts a
where
  a.id = @account_id::bigint
  and a.owner_id = @owner_user_id::uuid -- only owners can add collaborators
  and @collaborator_user_id::uuid != @owner_user_id::uuid -- can't add yourself
  on CONFLICT do NOTHING
returning
  account_id,
  user_id,
  added_at;

-- name: RemoveAccountCollaborator :execrows
delete from
  account_users
where
  account_id = @account_id::bigint
  and user_id = @collaborator_user_id::uuid
  and exists (
    select
      1
    from
      accounts a
    where
      a.id = @account_id::bigint
      and a.owner_id = @owner_user_id::uuid
  );

-- name: ListAccountCollaborators :many
select
  u.id,
  u.email,
  u.display_name,
  au.added_at
from
  account_users au
  join users u on u.id = au.user_id
  join accounts a on a.id = au.account_id
where
  au.account_id = @account_id::bigint
  and (
    a.owner_id = @requesting_user_id::uuid
    or au.user_id = @requesting_user_id::uuid
  )
order by
  u.email;

-- name: ListUserCollaborations :many
select
  a.id as account_id,
  a.name as account_name,
  a.bank,
  au.added_at,
  u.email as owner_email,
  u.display_name as owner_name
from
  account_users au
  join accounts a on au.account_id = a.id
  join users u on a.owner_id = u.id
where
  au.user_id = @user_id::uuid
order by
  au.added_at desc;

-- name: CheckAccountCollaborator :one
select
  exists(
    select
      1
    from
      account_users au
      join accounts a on au.account_id = a.id
    where
      au.account_id = @account_id::bigint
      and au.user_id = @user_id::uuid
      or a.owner_id = @user_id::uuid
  ) as is_collaborator;

-- name: GetAccountCollaboratorCount :one
select
  COUNT(*) as collaborator_count
from
  account_users
where
  account_id = @account_id::bigint;

-- name: RemoveUserFromAllAccounts :execrows
delete from
  account_users
where
  user_id = @user_id::uuid;

-- name: TransferAccountOwnership :execrows
update
  accounts
set
  owner_id = @new_owner_id::uuid
where
  id = @account_id::bigint
  and owner_id = @current_owner_id::uuid
  and exists (
    select
      1
    from
      account_users
    where
      account_id = @account_id::bigint
      and user_id = @new_owner_id::uuid
  );

-- name: LeaveAccountCollaboration :execrows
delete from
  account_users
where
  account_id = @account_id::bigint
  and user_id = @user_id::uuid;
