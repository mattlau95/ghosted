// fetcher pulls every active company's board, normalizes it, and upserts
// into postings — bumping last_seen on postings still open and setting
// closed_at on anything that dropped off the board since the last run.
// No scoring, no delivery, no HTTP server. Usage:
//
//	DATABASE_URL=postgres://... go run ./cmd/fetcher
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"time"

	"ghosted/internal/db"
	"ghosted/internal/fetcher"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx := context.Background()
	conn, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer conn.Close()

	companies, err := db.ActiveCompanies(ctx, conn)
	if err != nil {
		log.Fatalf("loading active companies: %v", err)
	}
	fmt.Printf("fetching %d active companies...\n", len(companies))

	client := &http.Client{Timeout: 20 * time.Second}

	var fetched, skipped, errored, postingsSeen int
	for i, c := range companies {
		var postings []fetcher.Posting
		var fetchErr error

		switch c.ATS {
		case "greenhouse":
			postings, fetchErr = fetcher.FetchGreenhouse(client, c.BoardToken)
		case "lever":
			postings, fetchErr = fetcher.FetchLever(client, c.BoardToken)
		case "ashby":
			postings, fetchErr = fetcher.FetchAshby(client, c.BoardToken)
		default:
			fmt.Printf("[%d/%d] %-30s skip (unsupported ats %q)\n", i+1, len(companies), c.Name, c.ATS)
			skipped++
			continue
		}

		if fetchErr != nil {
			fmt.Printf("[%d/%d] %-30s ERROR: %v\n", i+1, len(companies), c.Name, fetchErr)
			errored++
			continue
		}

		if err := db.CloseAllOpenPostings(ctx, conn, c.ID); err != nil {
			log.Fatalf("closing stale postings for %s: %v", c.Name, err)
		}
		for _, p := range postings {
			if err := db.UpsertPosting(ctx, conn, db.Posting{
				CompanyID:   c.ID,
				ATS:         p.ATS,
				ATSJobID:    p.ATSJobID,
				Title:       p.Title,
				Location:    p.Location,
				URL:         p.URL,
				Description: p.Description,
			}); err != nil {
				log.Fatalf("upserting posting %s/%s: %v", c.Name, p.ATSJobID, err)
			}
		}

		fmt.Printf("[%d/%d] %-30s %d postings\n", i+1, len(companies), c.Name, len(postings))
		fetched++
		postingsSeen += len(postings)

		politeSleep()
	}

	fmt.Printf("\ndone: %d companies fetched (%d postings), %d skipped (unsupported ats), %d errored\n", fetched, postingsSeen, skipped, errored)
}

// politeSleep waits 300-500ms between requests, matching the resolver's rate.
func politeSleep() {
	time.Sleep(300*time.Millisecond + time.Duration(rand.IntN(201))*time.Millisecond)
}
