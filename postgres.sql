CREATE TABLE days (
  domain text NOT NULL,
  day date NOT NULL,
  sessions jsonb NOT NULL DEFAULT '[]',

  PRIMARY KEY (domain, day)
);

CREATE TABLE months (
  domain text NOT NULL,
  month date NOT NULL,
  referrer_summaries jsonb NOT NULL DEFAULT '{}',

  PRIMARY KEY (domain, month)
);

CREATE TABLE temp_migration (
  domain text,
  code text
);

drop table payments;
drop table sites;
drop table users;
