package db

import (
	"context"
	"database/sql"
)

type Posting struct {
	CompanyID   int
	ATS         string
	ATSJobID    string
	Title       string
	Location    string
	URL         string
	Description string
}

// CloseAllOpenPostings marks every currently-open posting for a company as
// closed. Call this before upserting a fresh fetch for that company —
// UpsertPosting reopens (clears closed_at on) anything still present, so
// whatever's left closed afterward is genuinely gone from the board.
func CloseAllOpenPostings(ctx context.Context, conn *sql.DB, companyID int) error {
	_, err := conn.ExecContext(ctx, `
		UPDATE postings SET closed_at = now()
		WHERE company_id = $1 AND closed_at IS NULL
	`, companyID)
	return err
}

// UpsertPosting inserts a new posting or, on conflict with the (ats,
// ats_job_id) dedupe key, refreshes its fields, bumps last_seen, and clears
// closed_at — first_seen is left untouched on update.
func UpsertPosting(ctx context.Context, conn *sql.DB, p Posting) error {
	_, err := conn.ExecContext(ctx, `
		INSERT INTO postings (company_id, ats, ats_job_id, title, location, url, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (ats, ats_job_id) DO UPDATE SET
			title = EXCLUDED.title,
			location = EXCLUDED.location,
			url = EXCLUDED.url,
			description = EXCLUDED.description,
			last_seen = now(),
			closed_at = NULL
	`, p.CompanyID, p.ATS, p.ATSJobID, p.Title, p.Location, p.URL, p.Description)
	return err
}
