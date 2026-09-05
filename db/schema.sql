-- ghosted schema — see docs/PROJECT.md for the full data model rationale.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS companies (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    name_norm   TEXT NOT NULL UNIQUE,
    tier        TEXT NOT NULL DEFAULT '',  -- Yes | OK | Prefer Not
    ats         TEXT NOT NULL,
    board_token TEXT NOT NULL DEFAULT '',
    active      BOOLEAN NOT NULL DEFAULT true,
    notes       TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS postings (
    id          SERIAL PRIMARY KEY,
    company_id  INTEGER NOT NULL REFERENCES companies(id),
    ats         TEXT NOT NULL,
    ats_job_id  TEXT NOT NULL,
    title       TEXT NOT NULL,
    location    TEXT NOT NULL DEFAULT '',
    url         TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    first_seen  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen   TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at   TIMESTAMPTZ,

    -- application tracking
    status        TEXT,   -- null | interested | applied | screening | onsite | offer | rejected
    applied_at    TIMESTAMPTZ,
    last_contact  TIMESTAMPTZ,
    next_action   TEXT,
    contact_name  TEXT,
    notes         TEXT,
    source        TEXT,   -- board | referral | direct | recruiter

    UNIQUE (ats, ats_job_id)
);

CREATE TABLE IF NOT EXISTS scores (
    posting_id  INTEGER PRIMARY KEY REFERENCES postings(id),
    want        INTEGER NOT NULL,
    fit         INTEGER NOT NULL,
    access      INTEGER NOT NULL,
    composite   INTEGER NOT NULL,
    lead_with   TEXT NOT NULL,
    reasoning   TEXT NOT NULL,
    concerns    TEXT,
    model       TEXT NOT NULL,
    scored_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS connections (
    id            SERIAL PRIMARY KEY,
    name          TEXT NOT NULL,
    company       TEXT NOT NULL,
    company_norm  TEXT NOT NULL,
    title         TEXT NOT NULL DEFAULT '',
    tier          INTEGER NOT NULL DEFAULT 4,
    last_verified TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_postings_company_id ON postings(company_id);
CREATE INDEX IF NOT EXISTS idx_postings_closed_at ON postings(closed_at) WHERE closed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_connections_company_norm_trgm ON connections USING gin (company_norm gin_trgm_ops);
