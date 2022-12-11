create table if not exists permissions
(
    id   uuid primary key,
    code varchar(100) not null
);

create table if not exists users_permissions
(
    user_id       uuid not null references users on delete cascade,
    permission_id uuid not null references permissions on delete cascade,
    primary key (user_id, permission_id)
);

insert into permissions (id, code)
values (gen_random_uuid(), 'quotes:read'),
       (gen_random_uuid(), 'quotes:write'),
       (gen_random_uuid(), 'quotes:state');
