-- +goose Up
create table transaction_rules (
    rule_id         uuid primary key default gen_random_uuid(),
    user_id         uuid not null references users(id),
    rule_name       varchar(255) not null,
    category_id     bigint not null references categories(id),

    conditions      jsonb not null,

    is_active       boolean default true,
    priority_order  integer not null default 0,
    rule_source     varchar(20) not null default 'user_created', -- 'user_created', 'ai_suggested', 'ai_approved'

    created_at      timestamptz not null default now(),
    updated_at      timestamptz not null default now(),
    last_applied_at timestamptz,

    times_applied   integer default 0,

    unique(user_id, rule_name)
);

create index idx_rules_user_active_priority
on transaction_rules(user_id, is_active, priority_order);

create index idx_rules_source
on transaction_rules(user_id, rule_source, created_at);

-- +goose Down
drop index if exists idx_rules_source;
drop index if exists idx_rules_user_active_priority;
drop table if exists transaction_rules;