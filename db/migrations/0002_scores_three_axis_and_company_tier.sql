-- Stage 3: scores table redesigned to match docs/SCORING.md's three-axis
-- contract (want/fit/access/composite/lead_with), and companies gains the
-- tier column SCORING.md's Want axis needs as a modifier. Safe to run
-- against a database with an empty scores table (no data migration needed).

ALTER TABLE companies ADD COLUMN IF NOT EXISTS tier TEXT NOT NULL DEFAULT '';

DROP TABLE IF EXISTS scores;

CREATE TABLE scores (
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
