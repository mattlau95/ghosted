// sync-companies loads docs/companies.csv into the companies table, keyed
// on the normalized company name so re-running it is idempotent. Run this
// whenever companies.csv changes (new watchlist entries, newly resolved
// tokens) before running the fetcher. Usage:
//
//	DATABASE_URL=postgres://... go run ./cmd/sync-companies
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"ghosted/internal/db"
	"ghosted/internal/resolver"
)

func main() {
	input := flag.String("input", "docs/companies.csv", "watchlist CSV to sync")
	flag.Parse()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	companies, err := resolver.ReadCompanies(*input)
	if err != nil {
		log.Fatalf("reading %s: %v", *input, err)
	}

	ctx := context.Background()
	conn, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer conn.Close()

	var synced int
	for _, c := range companies {
		ats, token := c.KnownATS, c.KnownToken
		notes := c.Why
		if c.Manual {
			ats = "manual"
			token = ""
			if c.ManualNote != "" {
				notes = c.Why + " — " + c.ManualNote
			}
		}

		row := db.Company{
			Name:       c.Name,
			NameNorm:   resolver.NormalizeCompanyName(c.Name),
			Tier:       c.Tier,
			ATS:        ats,
			BoardToken: token,
			Active:     !c.Manual,
			Notes:      notes,
		}
		if err := db.UpsertCompany(ctx, conn, row); err != nil {
			log.Fatalf("upserting %s: %v", c.Name, err)
		}
		synced++
	}

	fmt.Printf("synced %d companies from %s\n", synced, *input)
}
