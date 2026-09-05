package resolver

import (
	"fmt"
	"net/http"
	"time"
)

// Progress is called after each company is resolved, for CLI status output.
type Progress func(i, total int, c Company, r Result)

// Resolve runs Stage 1 resolution over the full watchlist: skips companies
// flagged manual, accepts pre-verified known_ats/known_token pairs as-is,
// and probes everything else against Greenhouse, Lever, and Ashby.
func Resolve(companies []Company, progress Progress) []Result {
	client := &http.Client{Timeout: 15 * time.Second}

	results := make([]Result, 0, len(companies))
	for i, c := range companies {
		r := resolveOne(client, c)
		results = append(results, r)
		if progress != nil {
			progress(i+1, len(companies), c, r)
		}
	}
	return results
}

func resolveOne(client *http.Client, c Company) Result {
	if c.Manual {
		return Result{
			Company:     c.Name,
			ATS:         "manual",
			Confidence:  "none",
			NeedsReview: true,
			Notes:       c.ManualNote,
		}
	}

	if c.KnownATS != "" && c.KnownToken != "" {
		return Result{
			Company:     c.Name,
			ATS:         c.KnownATS,
			BoardToken:  c.KnownToken,
			Confidence:  "high",
			NeedsReview: false,
			Notes:       "known (skip probing)",
		}
	}

	primary := GenerateCandidates(c.Name)
	tried := make([]string, 0, len(primary))
	seen := make(map[string]bool, len(primary))

	if r, ok := probeCandidates(client, c, primary, &tried, seen); ok {
		return r
	}

	// Second pass: the real token is sometimes a suffixed or acronymed
	// variant (noomgrowth, ww) rather than a literal rendering of the
	// name. Only worth the extra requests once the primary pass misses.
	fallback := GenerateFallbackCandidates(c.Name)
	if r, ok := probeCandidates(client, c, fallback, &tried, seen); ok {
		return r
	}

	return Result{
		Company:         c.Name,
		Confidence:      "none",
		CandidatesTried: tried,
		NeedsReview:     true,
		Notes:           fmt.Sprintf("no match across %d candidate(s) on any ATS", len(tried)),
	}
}

// probeCandidates tries each candidate (skipping any already tried) against
// Greenhouse, Lever, then Ashby, stopping at the first hit.
func probeCandidates(client *http.Client, c Company, candidates []string, tried *[]string, seen map[string]bool) (Result, bool) {
	for _, cand := range candidates {
		if seen[cand] {
			continue
		}
		seen[cand] = true
		*tried = append(*tried, cand)

		outcome, err := probeGreenhouse(client, cand, c.Name)
		politeSleep()
		if err != nil || outcome == nil {
			outcome, err = probeLever(client, cand)
			politeSleep()
		}
		if err != nil || outcome == nil {
			outcome, err = probeAshby(client, cand)
			politeSleep()
		}
		if err != nil || outcome == nil {
			continue
		}

		return Result{
			Company:         c.Name,
			ATS:             outcome.ATS,
			BoardToken:      outcome.Token,
			Confidence:      outcome.Confidence,
			CandidatesTried: *tried,
			ResolvedName:    outcome.ResolvedName,
			NeedsReview:     outcome.Confidence != "high",
			Notes:           outcome.Notes,
		}, true
	}
	return Result{}, false
}
