package resolver

// Company is one row of the Stage 0 watchlist input.
type Company struct {
	Name       string
	Tier       string
	Why        string
	Manual     bool
	ManualNote string
	KnownATS   string
	KnownToken string
}

// Result is one row of the Stage 1 resolver output.
type Result struct {
	Company         string
	ATS             string
	BoardToken      string
	Confidence      string // high | low | none
	CandidatesTried []string
	ResolvedName    string
	NeedsReview     bool
	Notes           string
}
