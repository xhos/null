-- +goose Up
alter table transaction_rules
add column merchant text;

-- make category_id nullable since rules can now set merchant OR category OR both
alter table transaction_rules
alter column category_id drop not null;

-- add constraint to ensure at least one action is specified
alter table transaction_rules
add constraint check_has_action check (
    category_id is not null or merchant is not null
);

-- +goose Down
alter table transaction_rules
drop constraint if exists check_has_action;

alter table transaction_rules
alter column category_id set not null;

alter table transaction_rules
drop column merchant;
