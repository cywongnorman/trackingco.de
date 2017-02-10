CREATE EXTENSION chkpass;

CREATE TABLE users (
  id serial primary key,
  name text unique,
  email text,
  pass chkpass
);

CREATE TABLE settings (
  user_id int primary key REFERENCES users(id),
  sites_order text[]
);

CREATE TABLE sites (
  code text primary key,
  user_id int REFERENCES users(id),
  name text,
  created_at date
);
