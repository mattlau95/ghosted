package resolver

import (
	"regexp"
	"strings"
)

// corporateSuffixRe matches a trailing corporate suffix, optionally preceded
// by a comma, e.g. "Foo, Inc." or "Foo Technologies".
var corporateSuffixRe = regexp.MustCompile(`(?i),?\s*(inc|llc|corp|ltd|labs|technologies|group)\.?\s*$`)

var nonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)
var nonAlnumPreserveCaseRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)
var nonSlugRe = regexp.MustCompile(`[^a-z0-9-]+`)
var multiHyphenRe = regexp.MustCompile(`-+`)

func stripCorporateSuffix(name string) string {
	return strings.TrimSpace(corporateSuffixRe.ReplaceAllString(name, ""))
}

// alnumOnly lowercases and strips everything but letters/digits.
// "Chess.com" -> "chesscom"
func alnumOnly(name string) string {
	return nonAlnumRe.ReplaceAllString(strings.ToLower(name), "")
}

// hyphenate lowercases, turns spaces into hyphens, and strips other
// punctuation. "Trust Wallet" -> "trust-wallet"
func hyphenate(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = nonSlugRe.ReplaceAllString(s, "")
	s = multiHyphenRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// alnumOnlyPreserveCase strips spaces/punctuation but keeps original casing.
// "US Mobile" -> "USMobile", "Vetcove" -> "Vetcove"
func alnumOnlyPreserveCase(name string) string {
	return nonAlnumPreserveCaseRe.ReplaceAllString(name, "")
}

// GenerateCandidates produces up to 5 deduped slug candidates for a company
// name, in the priority order defined in docs/RESOLVER.md.
func GenerateCandidates(name string) []string {
	base := strings.TrimSpace(name)
	if base == "" {
		return nil
	}

	var raw []string
	add := func(s string) {
		if s != "" {
			raw = append(raw, s)
		}
	}

	// 1. Lowercase, strip all non-alphanumerics
	add(alnumOnly(base))

	// 2. Lowercase, spaces -> hyphens
	add(hyphenate(base))

	// 3. Strip corporate suffixes first, then apply 1 and 2
	if stripped := stripCorporateSuffix(base); stripped != base && stripped != "" {
		add(alnumOnly(stripped))
		add(hyphenate(stripped))
	}

	// 4. Original casing, no spaces
	add(alnumOnlyPreserveCase(base))

	// 5. First word only, lowercased
	if fields := strings.Fields(base); len(fields) > 0 {
		add(alnumOnly(fields[0]))
	}

	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, 5)
	for _, c := range raw {
		if seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
		if len(out) == 5 {
			break
		}
	}
	return out
}

// fallbackSuffixes are appended to a company's base slug when the primary
// candidates all miss. Real observed tokens carry these: noomgrowth, not
// noom.
var fallbackSuffixes = []string{"growth", "careers", "inc", "hq", "us"}

// GenerateFallbackCandidates produces a second-pass candidate list, tried
// only after GenerateCandidates misses on every ATS. Company-name-derived
// slugs miss when the real token has a suffix (noomgrowth) or is an
// acronym (ww for Weight Watchers) rather than a literal rendering of the
// name.
func GenerateFallbackCandidates(name string) []string {
	base := strings.TrimSpace(name)
	if base == "" {
		return nil
	}

	var raw []string
	add := func(s string) {
		if s != "" {
			raw = append(raw, s)
		}
	}

	root := alnumOnly(base)
	for _, suffix := range fallbackSuffixes {
		add(root + suffix)
	}

	if acronym := acronymOf(base); acronym != "" {
		add(acronym)
	}

	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))
	for _, c := range raw {
		if seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

// acronymOf builds a lowercase acronym from a multi-word name, ignoring
// fields that are pure punctuation (e.g. "&"). "Weight Watchers" -> "ww".
// Returns "" for single-word names, where an acronym is meaningless.
func acronymOf(name string) string {
	var letters []byte
	for _, field := range strings.Fields(name) {
		for _, r := range field {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				letters = append(letters, byte(strings.ToLower(string(r))[0]))
				break
			}
		}
	}
	if len(letters) < 2 {
		return ""
	}
	return string(letters)
}

// NormalizeCompanyName reduces a display name to letters/digits/spaces for
// loose comparison, stripping common corporate suffixes.
func NormalizeCompanyName(name string) string {
	s := strings.ToLower(stripCorporateSuffix(name))
	s = nonAlnumRe.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}
