-- name: ListUsers :many
SELECT id, email, display_name, default_account_id, created_at, updated_at
FROM users
ORDER BY created_at DESC;

-- name: GetUser :one
SELECT id, email, display_name, default_account_id, created_at, updated_at
FROM users
WHERE id = @id::uuid;

-- name: GetUserByEmail :one
SELECT id, email, display_name, default_account_id, created_at, updated_at
FROM users
WHERE lower(email) = lower(@email::text);

-- name: CreateUser :one
INSERT INTO users (id, email, display_name)
VALUES (@id::uuid, @email::text, sqlc.narg('display_name')::text)
RETURNING id, email, display_name, default_account_id, created_at, updated_at;

-- name: UpdateUser :one
UPDATE users
SET email = COALESCE(sqlc.narg('email')::text, email),
    display_name = COALESCE(sqlc.narg('display_name')::text, display_name),
    default_account_id = COALESCE(sqlc.narg('default_account_id')::bigint, default_account_id)
WHERE id = @id::uuid
RETURNING id, email, display_name, default_account_id, created_at, updated_at;

-- name: UpdateUserDisplayName :one
UPDATE users
SET display_name = @display_name::text
WHERE id = @id::uuid
RETURNING id, email, display_name, default_account_id, created_at, updated_at;

-- name: SetUserDefaultAccount :one
UPDATE users
SET default_account_id = @default_account_id::bigint
WHERE id = @id::uuid
RETURNING id, email, display_name, default_account_id, created_at, updated_at;

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = @id::uuid;

-- name: DeleteUserWithCascade :execrows
WITH removed_from_accounts AS (
    DELETE FROM account_users
    WHERE user_id = @id::uuid
    RETURNING user_id
)
DELETE FROM users
WHERE id = @id::uuid;

-- name: CheckUserExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE id = @id::uuid) AS exists;

-- name: GetUserFirstAccount :one
SELECT id FROM accounts 
WHERE owner_id = @user_id::uuid 
ORDER BY created_at ASC 
LIMIT 1;