create table if not exists otps
(
    id      uuid                        not null primary key,
    hash    text                        not null,
    user_id uuid                        not null references users (id) on delete cascade,
    expiry  timestamp(0) with time zone not null,
    created timestamptz                 not null,
    "scope" text                        not null
)
