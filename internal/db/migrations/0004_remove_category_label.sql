-- +goose Up
alter table categories 
drop column label;

-- +goose Down
alter table categories
add column label TEXT not null default '';
