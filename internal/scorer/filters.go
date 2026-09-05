package scorer

import "strings"

// mustMatchTitle phrases — see docs/SCORING.md "Hard filters".
var mustMatchTitle = []string{
	"design engineer", "ux engineer", "design systems", "design technologist",
	"front-end", "frontend", "product designer", "ui engineer", "web engineer",
	"creative technologist",
}

// rejectTitleUnconditional phrases reject regardless of location.
// "staff" and "senior" are deliberately absent — see docs/SCORING.md.
var rejectTitleUnconditional = []string{
	"principal", "director", "head of", "vp", "manager", "intern",
}

var nycMetro = []string{"new york", "nyc", "brooklyn", "manhattan", "queens", "bronx"}
var philaMetro = []string{"philadelphia", "philly"}
var newJersey = []string{"new jersey", ", nj", "jersey city", "newark", "fort lee", "hoboken", "trenton"}

// nonUSRemoteMarkers catch "Remote" postings that are remote somewhere other
// than the US — an unqualified "Remote" is treated as US-eligible.
var nonUSRemoteMarkers = []string{
	"emea", "apac", "uk", "united kingdom", "canada", "europe", "india",
	"latam", "australia", "germany", "ireland", "poland", "worldwide", "international",
}

// FilterResult is the outcome of the cheap pre-model filter.
type FilterResult struct {
	Pass   bool
	Reason string
}

// PassesHardFilters applies docs/SCORING.md's SQL-tier hard filters in Go.
// These are approximate heuristics on free-text title/location strings, not
// a precise classifier — expect to tune the location buckets after reading
// a week of real output, same as the scoring rubric itself.
func PassesHardFilters(title, location string) FilterResult {
	titleLower := strings.ToLower(title)

	if !containsAny(titleLower, mustMatchTitle) {
		return FilterResult{false, "title does not match any required phrase"}
	}
	if reason, ok := matchesAny(titleLower, rejectTitleUnconditional); ok {
		return FilterResult{false, "title contains reject keyword: " + reason}
	}
	if strings.Contains(titleLower, "contract") && !isRemoteUS(location) {
		return FilterResult{false, "contract role, not remote"}
	}
	if containsNonASCII(location) {
		return FilterResult{false, "non-English location string"}
	}
	if !isAllowedLocation(location) {
		return FilterResult{false, "location not in allowed set (Remote US / NYC / Philly / NJ)"}
	}
	return FilterResult{true, ""}
}

func isAllowedLocation(location string) bool {
	if isRemoteUS(location) {
		return true
	}
	lower := strings.ToLower(location)
	return containsAny(lower, nycMetro) || containsAny(lower, philaMetro) || containsAny(lower, newJersey)
}

func isRemoteUS(location string) bool {
	lower := strings.ToLower(location)
	if !strings.Contains(lower, "remote") {
		return false
	}
	return !containsAny(lower, nonUSRemoteMarkers)
}

func containsAny(s string, phrases []string) bool {
	for _, p := range phrases {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func matchesAny(s string, phrases []string) (string, bool) {
	for _, p := range phrases {
		if strings.Contains(s, p) {
			return p, true
		}
	}
	return "", false
}

func containsNonASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}
