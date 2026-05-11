CREATE TABLE IF NOT EXISTS websites (
    id   SERIAL PRIMARY KEY,
    url  TEXT   NOT NULL UNIQUE,
    name TEXT   NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS checks (
    id          BIGSERIAL   PRIMARY KEY,
    website_id  INTEGER     NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    checked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_up       BOOLEAN     NOT NULL,
    status_code INTEGER     NOT NULL,
    response_ms INTEGER     NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_checks_site_time
    ON checks (website_id, checked_at DESC);
