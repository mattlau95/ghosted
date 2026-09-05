package db

import (
	"context"
	"database/sql"
)

// UnscoredPosting is an open posting with no row in scores yet, joined with
// the company context the scorer needs.
type UnscoredPosting struct {
	PostingID   int
	CompanyName string
	CompanyTier string
	Title       string
	Location    string
	Description string
}

// UnscoredOpenPostings returns every open posting that hasn't been scored.
func UnscoredOpenPostings(ctx context.Context, conn *sql.DB) ([]UnscoredPosting, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT p.id, c.name, c.tier, p.title, p.location, p.description
		FROM postings p
		JOIN companies c ON c.id = p.company_id
		LEFT JOIN scores s ON s.posting_id = p.id
		WHERE p.closed_at IS NULL AND s.posting_id IS NULL
		ORDER BY p.first_seen
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var postings []UnscoredPosting
	for rows.Next() {
		var p UnscoredPosting
		if err := rows.Scan(&p.PostingID, &p.CompanyName, &p.CompanyTier, &p.Title, &p.Location, &p.Description); err != nil {
			return nil, err
		}
		postings = append(postings, p)
	}
	return postings, rows.Err()
}

type ScoreRow struct {
	PostingID int
	Want      int
	Fit       int
	Access    int
	Composite int
	LeadWith  string
	Reasoning string
	Concerns  string
	Model     string
}

// InsertScore records a posting's score. Scoring runs once per posting —
// callers should not call this for a posting that already has a score row.
func InsertScore(ctx context.Context, conn *sql.DB, s ScoreRow) error {
	var concerns any
	if s.Concerns != "" {
		concerns = s.Concerns
	}
	_, err := conn.ExecContext(ctx, `
		INSERT INTO scores (posting_id, want, fit, access, composite, lead_with, reasoning, concerns, model)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, s.PostingID, s.Want, s.Fit, s.Access, s.Composite, s.LeadWith, s.Reasoning, concerns, s.Model)
	return err
}
