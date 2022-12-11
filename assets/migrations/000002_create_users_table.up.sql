create table if not exists users
(
    id              uuid        not null primary key,
    created_at      timestamptz not null,
    email           text        not null unique,
    hashed_password text        not null,
    name            text        not null,
    is_activated    boolean     not null default false
);
