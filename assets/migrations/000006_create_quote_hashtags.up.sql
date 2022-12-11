create table if not exists hashtags
(
    id    uuid not null primary key,
    value text not null unique
);

create table if not exists quote_hashtags
(
    quote_id   uuid not null references quotes on delete cascade,
    hashtag_id uuid not null references hashtags on delete cascade,
    primary key (quote_id, hashtag_id)
);

