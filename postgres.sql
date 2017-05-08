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

CREATE TABLE payments (
  user_email text REFERENCES users(email),
  bitpay_invoice text,
  paid boolean
);
