CREATE TABLE users (
  id text primary key, -- id or account from accountd.xyz
  sites_order text[],
  colours jsonb,
  domains text[] NOT NULL DEFAULT '{}',
  plan float NOT NULL DEFAULT 0
);

CREATE TABLE sites (
  code text primary key,
  owner text,
  name text,
  shared boolean DEFAULT false,
  created_at date DEFAULT now()
);
