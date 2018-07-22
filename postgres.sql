CREATE TABLE users (
  id text primary key, -- id or account from accountd.xyz
  sites_order text[],
  colours jsonb,
  domains text[] NOT NULL DEFAULT '{}',
  months_using text[] NOT NULL DEFAULT '{}',
  months_free int NOT NULL DEFAULT 0
);

CREATE TABLE sites (
  code text primary key,
  owner text,
  name text,
  shared boolean DEFAULT false,
  created_at date DEFAULT now()
);

CREATE TABLE payments (
  id text PRIMARY KEY,
  user_id text NOT NULL,
  created_at timestamp NOT NULL DEFAULT 'now',
  paid_at timestamp,
  amount int NOT NULL -- amount in satoshis
);
