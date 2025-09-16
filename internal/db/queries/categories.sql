-- name: ListCategoriesWithUsage :many
SELECT 
  c.id, c.user_id, c.slug, c.label, c.color, c.created_at, c.updated_at,
  COUNT(t.id) AS usage_count,
  COALESCE(SUM(t.tx_amount), 0) AS total_amount
FROM categories c
LEFT JOIN transactions t ON c.id = t.category_id
LEFT JOIN accounts a ON t.account_id = a.id
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
WHERE c.user_id = @user_id::uuid
  AND (@user_id::uuid IS NULL OR (a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL))
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR t.tx_date >= sqlc.narg('start_date')::timestamptz)
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR t.tx_date <= sqlc.narg('end_date')::timestamptz)
GROUP BY c.id, c.user_id, c.slug, c.label, c.color, c.created_at, c.updated_at
ORDER BY usage_count DESC, c.slug
LIMIT COALESCE(sqlc.narg('limit')::int, 100);

-- name: ListCategories :many
SELECT id, user_id, slug, label, color, created_at, updated_at
FROM categories
WHERE user_id = @user_id::uuid
ORDER BY slug;

-- name: GetCategory :one
SELECT id, user_id, slug, label, color, created_at, updated_at
FROM categories
WHERE id = @id::bigint AND user_id = @user_id::uuid;

-- name: GetCategoryBySlug :one
SELECT id, user_id, slug, label, color, created_at, updated_at
FROM categories
WHERE slug = @slug::text AND user_id = @user_id::uuid;

-- name: GetCategoryWithStats :one
SELECT 
  c.id, c.user_id, c.slug, c.label, c.color, c.created_at, c.updated_at,
  COUNT(t.id) AS usage_count,
  COALESCE(SUM(t.tx_amount), 0) AS total_amount,
  COALESCE(AVG(t.tx_amount), 0) AS avg_amount,
  MIN(t.tx_date) AS first_used,
  MAX(t.tx_date) AS last_used
FROM categories c
LEFT JOIN transactions t ON c.id = t.category_id
LEFT JOIN accounts a ON t.account_id = a.id
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
WHERE c.id = @id::bigint AND c.user_id = @user_id::uuid
  AND (@user_id::uuid IS NULL OR (a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL))
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR t.tx_date >= sqlc.narg('start_date')::timestamptz)
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR t.tx_date <= sqlc.narg('end_date')::timestamptz)
GROUP BY c.id, c.user_id, c.slug, c.label, c.color, c.created_at, c.updated_at;

-- name: ListCategorySlugs :many
SELECT slug
FROM categories
WHERE user_id = @user_id::uuid
ORDER BY slug;

-- name: CreateCategory :one
INSERT INTO categories (user_id, slug, label, color)
VALUES (@user_id::uuid, @slug::text, @label::text, @color::text)
RETURNING id, user_id, slug, label, color, created_at, updated_at;

-- name: UpdateCategory :one
UPDATE categories
SET slug = COALESCE(sqlc.narg('slug')::text, slug),
    label = COALESCE(sqlc.narg('label')::text, label),
    color = COALESCE(sqlc.narg('color')::text, color)
WHERE id = @id::bigint AND user_id = @user_id::uuid
RETURNING id, user_id, slug, label, color, created_at, updated_at;

-- name: DeleteCategory :execrows
DELETE FROM categories
WHERE id = @id::bigint AND user_id = @user_id::uuid;

-- name: DeleteUnusedCategories :execrows
DELETE FROM categories
WHERE user_id = @user_id::uuid
  AND id NOT IN (
    SELECT DISTINCT category_id 
    FROM transactions 
    WHERE category_id IS NOT NULL
  );

-- name: SearchCategories :many
SELECT id, user_id, slug, label, color, created_at, updated_at
FROM categories
WHERE user_id = @user_id::uuid
  AND (slug ILIKE ('%' || @query::text || '%') 
       OR label ILIKE ('%' || @query::text || '%'))
ORDER BY 
  CASE WHEN slug ILIKE (@query::text || '%') THEN 1 ELSE 2 END,
  slug;

-- name: GetMostUsedCategories :many
SELECT 
  c.id, c.user_id, c.slug, c.label, c.color, c.created_at, c.updated_at,
  COUNT(t.id) AS usage_count,
  SUM(t.tx_amount) AS total_amount
FROM categories c
JOIN transactions t ON c.id = t.category_id
JOIN accounts a ON t.account_id = a.id
LEFT JOIN account_users au ON a.id = au.account_id AND au.user_id = @user_id::uuid
WHERE c.user_id = @user_id::uuid
  AND (a.owner_id = @user_id::uuid OR au.user_id IS NOT NULL)
  AND (sqlc.narg('start')::timestamptz IS NULL OR t.tx_date >= sqlc.narg('start')::timestamptz)
  AND (sqlc.narg('end')::timestamptz IS NULL OR t.tx_date <= sqlc.narg('end')::timestamptz)
GROUP BY c.id, c.user_id, c.slug, c.label, c.color, c.created_at, c.updated_at
ORDER BY usage_count DESC
LIMIT COALESCE(sqlc.narg('limit')::int, 10);

-- name: GetUnusedCategories :many
SELECT c.id, c.user_id, c.slug, c.label, c.color, c.created_at, c.updated_at
FROM categories c
LEFT JOIN transactions t ON c.id = t.category_id
WHERE c.user_id = @user_id::uuid AND t.id IS NULL
ORDER BY c.created_at DESC;

-- name: BulkCreateCategories :copyfrom
INSERT INTO categories (user_id, slug, label, color) 
VALUES (@user_id, @slug, @label, @color);