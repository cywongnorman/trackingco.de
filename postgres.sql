CREATE TABLE days (
  domain text NOT NULL,
  day text NOT NULL, -- 20060102
  sessions jsonb NOT NULL DEFAULT '[]',

  PRIMARY KEY (domain, day)
);

CREATE TABLE months (
  domain text NOT NULL,
  month text NOT NULL, -- 200601
  nbounces int NOT NULL,
  nsessions int NOT NULL,
  npageviews int NOT NULL,
  score int NOT NULL,
  top_referrers jsonb NOT NULL,
  top_referrers_scores jsonb NOT NULL,
  top_pages jsonb NOT NULL,

  PRIMARY KEY (domain, month)
);

CREATE TABLE temp_migration (
  domain text,
  code text,

  UNIQUE (domain, code)
);

drop table payments;
drop table sites;
drop table users;
