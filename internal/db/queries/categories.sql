-- name: ListCategories :many
select
  id,
  user_id,
  slug,
  color,
  created_at,
  updated_at
from
  categories
where
  user_id = @user_id::uuid
order by
  slug;

-- name: GetCategory :one
select
  id,
  user_id,
  slug,
  color,
  created_at,
  updated_at
from
  categories
where
  id = @id::bigint
  and user_id = @user_id::uuid;

-- name: GetCategoryBySlug :one
select
  id,
  user_id,
  slug,
  color,
  created_at,
  updated_at
from
  categories
where
  slug = @slug::text
  and user_id = @user_id::uuid;

-- name: CreateCategory :one
insert into
  categories (user_id, slug, color)
values
  (@user_id::uuid, @slug::text, @color::text)
returning
  id,
  user_id,
  slug,
  color,
  created_at,
  updated_at;

-- name: UpdateCategory :one
update
  categories
set
  slug = sqlc.narg('slug')::text,
  color = sqlc.narg('color')::text
where
  id = @id::bigint
  and user_id = @user_id::uuid
returning
  id,
  user_id,
  slug,
  color,
  created_at,
  updated_at;

-- name: CreateCategoryIfNotExists :one
insert into
  categories (user_id, slug, color)
values
  (@user_id::uuid, @slug::text, @color::text) on CONFLICT (user_id, slug) do NOTHING
returning
  id,
  user_id,
  slug,
  color,
  created_at,
  updated_at;

-- name: DeleteCategoriesBySlugPrefix :execrows
delete from
  categories
where
  user_id = @user_id::uuid
  and (
    slug = @slug::text
    or slug like @slug::text || '.%'
  );

-- name: UpdateChildCategorySlugs :execrows
update
  categories
set
  slug = @new_slug_prefix::text || substring(slug from length(@old_slug_prefix::text) + 1)
where
  user_id = @user_id::uuid
  and slug like @old_slug_prefix::text || '.%';
