CREATE TABLE users (
  id text primary key,
  sites_order text[]
);

CREATE TABLE sites (
  code text primary key,
  user_id text REFERENCES users(id),
  name text,
  shared boolean DEFAULT false,
  created_at date DEFAULT now()
);
