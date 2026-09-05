package resolver

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

var inputHeader = []string{"company", "tier", "why", "manual", "manual_note", "known_ats", "known_token"}
var outputHeader = []string{"company", "ats", "board_token", "confidence", "candidates_tried", "resolved_name", "needs_review", "notes"}

// ReadCompanies loads the watchlist CSV. Header order must match
// inputHeader; extra trailing columns are ignored.
func ReadCompanies(path string) ([]Company, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}
	col := make(map[string]int, len(header))
	for i, h := range header {
		col[strings.TrimSpace(h)] = i
	}
	for _, want := range []string{"company", "manual", "known_ats", "known_token"} {
		if _, ok := col[want]; !ok {
			return nil, fmt.Errorf("input csv missing required column %q", want)
		}
	}

	get := func(rec []string, name string) string {
		i, ok := col[name]
		if !ok || i >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[i])
	}

	var companies []Company
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		name := get(rec, "company")
		if name == "" {
			continue
		}
		manual, _ := strconv.ParseBool(get(rec, "manual"))
		companies = append(companies, Company{
			Name:       name,
			Tier:       get(rec, "tier"),
			Why:        get(rec, "why"),
			Manual:     manual,
			ManualNote: get(rec, "manual_note"),
			KnownATS:   get(rec, "known_ats"),
			KnownToken: get(rec, "known_token"),
		})
	}
	return companies, nil
}

// WriteCompanies writes the watchlist CSV back out in its original schema —
// used to persist backfilled known_ats/known_token values.
func WriteCompanies(path string, companies []Company) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(inputHeader); err != nil {
		return err
	}
	for _, c := range companies {
		if err := w.Write([]string{
			c.Name,
			c.Tier,
			c.Why,
			strconv.FormatBool(c.Manual),
			c.ManualNote,
			c.KnownATS,
			c.KnownToken,
		}); err != nil {
			return err
		}
	}
	return w.Error()
}

// BackfillKnownValues fills in known_ats/known_token for any company that
// resolved at high confidence but doesn't already have a known value
// recorded — companies and results must be the same slice Resolve produced
// them from, in the same order. Returns the updated companies and how many
// rows changed; low-confidence and manual results are left untouched since
// those need a human look first.
func BackfillKnownValues(companies []Company, results []Result) ([]Company, int) {
	updated := make([]Company, len(companies))
	copy(updated, companies)

	changed := 0
	for i := range updated {
		if i >= len(results) {
			break
		}
		r := results[i]
		if updated[i].KnownATS != "" {
			continue
		}
		if r.Confidence != "high" || r.ATS == "" || r.ATS == "manual" {
			continue
		}
		updated[i].KnownATS = r.ATS
		updated[i].KnownToken = r.BoardToken
		changed++
	}
	return updated, changed
}

// WriteResults writes the resolver output CSV.
func WriteResults(path string, results []Result) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(outputHeader); err != nil {
		return err
	}
	for _, r := range results {
		if err := w.Write([]string{
			r.Company,
			r.ATS,
			r.BoardToken,
			r.Confidence,
			strings.Join(r.CandidatesTried, ";"),
			r.ResolvedName,
			strconv.FormatBool(r.NeedsReview),
			r.Notes,
		}); err != nil {
			return err
		}
	}
	return w.Error()
}

// WriteUnresolved writes the subset of results that missed entirely
// (confidence == "none"), for manual follow-up.
func WriteUnresolved(path string, results []Result) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"company", "candidates_tried", "notes"}); err != nil {
		return err
	}
	for _, r := range results {
		// Only genuine probe misses belong here — not companies
		// deliberately skipped via the manual flag.
		if r.Confidence != "none" || r.ATS == "manual" {
			continue
		}
		if err := w.Write([]string{r.Company, strings.Join(r.CandidatesTried, ";"), r.Notes}); err != nil {
			return err
		}
	}
	return w.Error()
}
