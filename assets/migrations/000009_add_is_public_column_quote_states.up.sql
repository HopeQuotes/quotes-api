alter table if exists quote_states
    add is_public boolean not null
        default (false);

update quote_states
set is_public = true
where value = 'accepted';