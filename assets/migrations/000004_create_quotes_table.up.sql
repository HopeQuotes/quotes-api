create table if not exists quote_states
(
    id         uuid        not null primary key,
    value      text        not null unique,
    is_default bool        not null default false,
    color      varchar(10) not null unique
);

insert into quote_states (id, value, is_default, color)
values (gen_random_uuid(), 'pending', true, '#F3D104'),
       (gen_random_uuid(), 'rejected', false, '#C94040'),
       (gen_random_uuid(), 'accepted', false, '#047A37');

create table if not exists photos
(
    id        uuid         not null primary key,
    color     varchar(8)   not null,
    blur_hash varchar(200) not null,
    author    varchar(100) not null,
    url       varchar(500) not null
);

create table if not exists quotes
(
    id         uuid        NOT NULL PRIMARY KEY,
    author     TEXT        NOT NULL,
    text       TEXT        NOT NULL,
    created_by uuid        NOT NULL REFERENCES users (id) ON DELETE NO ACTION,
    state      uuid        NOT NULL REFERENCES quote_states (id) ON DELETE NO ACTION,
    photo_id   uuid        NOT NULL REFERENCES photos (id) ON DELETE NO ACTION,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    version    INTEGER     NOT NULL DEFAULT 1
)