CREATE TABLE IF NOT EXISTS url_mappings (
  code        VARCHAR(10) PRIMARY KEY,
  original    TEXT        NOT NULL UNIQUE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
