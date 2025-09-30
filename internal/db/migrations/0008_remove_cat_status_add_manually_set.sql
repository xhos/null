-- +goose Up
-- remove cat_status and replace with simpler boolean flags
alter table transactions
add column category_manually_set boolean not null default false;

alter table transactions
add column merchant_manually_set boolean not null default false;

-- migrate existing data: cat_status=3 means manually set
update transactions
set category_manually_set = true
where cat_status = 3 and category_id is not null;

-- drop old cat_status column
alter table transactions
drop column cat_status;

-- +goose Down
alter table transactions
add column cat_status smallint not null default 0;

-- migrate back: manually_set=true means cat_status=3
update transactions
set cat_status = 3
where category_manually_set = true;

alter table transactions
drop column merchant_manually_set;

alter table transactions
drop column category_manually_set;
