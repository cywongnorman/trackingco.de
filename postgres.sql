CREATE TABLE users (
  email text primary key,
  sites_order text[],
  colours jsonb,
  domains text[] NOT NULL DEFAULT '{}',
  plan float NOT NULL DEFAULT 0
);

CREATE TABLE sites (
  code text primary key,
  user_email text REFERENCES users(email),
  name text,
  shared boolean DEFAULT false,
  created_at date DEFAULT now()
);

CREATE TABLE balances (
  id serial PRIMARY KEY,
  user_email text REFERENCES users(email),
  time date DEFAULT now(),
  delta integer, -- if positive, the user has paid something, if negative the user owes something
  due interval -- if this is an invoice, is it valid for a month? an year?
);
