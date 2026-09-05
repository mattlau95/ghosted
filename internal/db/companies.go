package db

import (
	"context"
	"database/sql"
)

type Company struct {
	ID         int
	Name       string
	NameNorm   string
	Tier       string
	ATS        string
	BoardToken string
	Active     bool
	Notes      string
}

// UpsertCompany inserts or updates a company keyed on the unique NameNorm.
func UpsertCompany(ctx context.Context, conn *sql.DB, c Company) error {
	_, err := conn.ExecContext(ctx, `
		INSERT INTO companies (name, name_norm, tier, ats, board_token, active, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (name_norm) DO UPDATE SET
			name = EXCLUDED.name,
			tier = EXCLUDED.tier,
			ats = EXCLUDED.ats,
			board_token = EXCLUDED.board_token,
			active = EXCLUDED.active,
			notes = EXCLUDED.notes
	`, c.Name, c.NameNorm, c.Tier, c.ATS, c.BoardToken, c.Active, c.Notes)
	return err
}

// ActiveCompanies returns every company marked active, for the fetcher to pull.
func ActiveCompanies(ctx context.Context, conn *sql.DB) ([]Company, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT id, name, name_norm, tier, ats, board_token, active, notes
		FROM companies
		WHERE active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var companies []Company
	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.ID, &c.Name, &c.NameNorm, &c.Tier, &c.ATS, &c.BoardToken, &c.Active, &c.Notes); err != nil {
			return nil, err
		}
		companies = append(companies, c)
	}
	return companies, rows.Err()
}
