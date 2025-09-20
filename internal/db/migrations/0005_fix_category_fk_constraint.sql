-- +goose Up
alter table transactions
drop constraint if exists transactions_category_id_fkey;

alter table transactions
add constraint transactions_category_id_fkey foreign key (category_id) references categories(id) on delete
set null;

-- +goose Down
alter table transactions
drop constraint if exists transactions_category_id_fkey;

alter table transactions
add constraint transactions_category_id_fkey foreign key (category_id) references categories(id);
