CREATE EXTENSION chkpass;

CREATE TABLE users (
  id serial primary key,
  name text unique,
  email text,
  pass chkpass,
  sites_order text[]
);

CREATE TABLE sites (
  code text primary key,
  user_id int REFERENCES users(id),
  name text,
  created_at date DEFAULT now()
);
