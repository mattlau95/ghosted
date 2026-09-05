// scorer applies the Stage 3 hard filters, then scores every surviving
// unscored open posting once via Claude Haiku. Prints to stdout — per
// docs/PROJECT.md, read a week of output before automating delivery.
//
//	DATABASE_URL=postgres://... ANTHROPIC_API_KEY=... go run ./cmd/scorer
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/anthropics/anthropic-sdk-go"

	"ghosted/internal/db"
	"ghosted/internal/scorer"
)

func main() {
	limit := flag.Int("limit", 0, "max postings to score this run (0 = no limit)")
	flag.Parse()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		log.Fatal("ANTHROPIC_API_KEY is not set")
	}

	ctx := context.Background()
	conn, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer conn.Close()

	postings, err := db.UnscoredOpenPostings(ctx, conn)
	if err != nil {
		log.Fatalf("loading unscored postings: %v", err)
	}
	fmt.Printf("%d unscored open postings\n", len(postings))

	client := anthropic.NewClient()

	var filtered, scored, errored int
	for _, p := range postings {
		if *limit > 0 && scored+errored >= *limit {
			break
		}

		result := scorer.PassesHardFilters(p.Title, p.Location)
		if !result.Pass {
			filtered++
			continue
		}

		s, err := scorer.ScorePosting(ctx, client, scorer.Posting{
			CompanyName: p.CompanyName,
			CompanyTier: p.CompanyTier,
			Title:       p.Title,
			Location:    p.Location,
			Description: p.Description,
		})
		if err != nil {
			fmt.Printf("[ERROR] %s - %s: %v\n", p.CompanyName, p.Title, err)
			errored++
			continue
		}

		if err := db.InsertScore(ctx, conn, db.ScoreRow{
			PostingID: p.PostingID,
			Want:      s.Want,
			Fit:       s.Fit,
			Access:    s.Access,
			Composite: s.Composite,
			LeadWith:  s.LeadWith,
			Reasoning: s.Reasoning,
			Concerns:  s.Concerns,
			Model:     "claude-haiku-4-5",
		}); err != nil {
			log.Fatalf("saving score for posting %d: %v", p.PostingID, err)
		}

		fmt.Printf("[%3d] %-25s %-40s want=%d fit=%d access=%d lead_with=%s\n",
			s.Composite, p.CompanyName, p.Title, s.Want, s.Fit, s.Access, s.LeadWith)
		if s.Concerns != "" {
			fmt.Printf("      concerns: %s\n", s.Concerns)
		}
		scored++
	}

	fmt.Printf("\ndone: %d scored, %d filtered out, %d errored\n", scored, filtered, errored)
}
