BEGIN;

create table refresh_tokens (
    token      text primary key,
    user_id    integer not null references users(id),
    expires_at timestamptz not null,
    revoked    boolean not null default false,
    created_at timestamptz not null default now()
);

COMMIT;