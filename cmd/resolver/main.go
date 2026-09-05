package main

import (
	"flag"
	"fmt"
	"log"

	"ghosted/internal/resolver"
)

func main() {
	input := flag.String("input", "docs/companies.csv", "input watchlist CSV")
	output := flag.String("output", "docs/companies.resolved.csv", "output CSV with resolved ATS + token")
	unresolved := flag.String("unresolved", "docs/unresolved.csv", "output CSV of companies that missed on every ATS")
	flag.Parse()

	companies, err := resolver.ReadCompanies(*input)
	if err != nil {
		log.Fatalf("reading %s: %v", *input, err)
	}
	fmt.Printf("resolving %d companies...\n", len(companies))

	results := resolver.Resolve(companies, func(i, total int, c resolver.Company, r resolver.Result) {
		switch {
		case r.ATS == "manual":
			fmt.Printf("[%d/%d] %-30s manual - skipping\n", i, total, c.Name)
		case r.Confidence == "high" && r.Notes == "known (skip probing)":
			fmt.Printf("[%d/%d] %-30s known: %s/%s\n", i, total, c.Name, r.ATS, r.BoardToken)
		case r.Confidence == "none":
			fmt.Printf("[%d/%d] %-30s MISS (tried %d candidates)\n", i, total, c.Name, len(r.CandidatesTried))
		default:
			fmt.Printf("[%d/%d] %-30s %s/%s (%s confidence)\n", i, total, c.Name, r.ATS, r.BoardToken, r.Confidence)
		}
	})

	if err := resolver.WriteResults(*output, results); err != nil {
		log.Fatalf("writing %s: %v", *output, err)
	}
	if err := resolver.WriteUnresolved(*unresolved, results); err != nil {
		log.Fatalf("writing %s: %v", *unresolved, err)
	}

	updatedCompanies, backfilled := resolver.BackfillKnownValues(companies, results)
	if backfilled > 0 {
		if err := resolver.WriteCompanies(*input, updatedCompanies); err != nil {
			log.Fatalf("backfilling known values into %s: %v", *input, err)
		}
		fmt.Printf("backfilled known_ats/known_token for %d high-confidence companies into %s\n", backfilled, *input)
	}

	var high, low, none, manual int
	for _, r := range results {
		switch {
		case r.ATS == "manual":
			manual++
		case r.Confidence == "high":
			high++
		case r.Confidence == "low":
			low++
		default:
			none++
		}
	}
	fmt.Printf("\ndone: %d high, %d low (needs review), %d missed, %d manual\n", high, low, none, manual)
	fmt.Printf("results: %s\nunresolved: %s\n", *output, *unresolved)
}
