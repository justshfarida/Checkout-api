create table sessions (
  token      text primary key,
  user_id    integer not null references users(id),
  created_at timestamptz not null default now()
);