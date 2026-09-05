package resolver

import "testing"

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func TestGenerateCandidates(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"Chess.com", "chesscom"},
		{"Trust Wallet", "trust-wallet"},
		{"Vetcove", "Vetcove"},
		{"US Mobile", "USMobile"},
		{"Weights & Biases", "weights"},
	}
	for _, c := range cases {
		got := GenerateCandidates(c.name)
		if !contains(got, c.want) {
			t.Errorf("GenerateCandidates(%q) = %v, want to contain %q", c.name, got, c.want)
		}
		if len(got) > 5 {
			t.Errorf("GenerateCandidates(%q) returned %d candidates, cap is 5", c.name, len(got))
		}
	}
}

func TestGenerateFallbackCandidates(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"Noom", "noomgrowth"},
		{"Weight Watchers", "ww"},
	}
	for _, c := range cases {
		got := GenerateFallbackCandidates(c.name)
		if !contains(got, c.want) {
			t.Errorf("GenerateFallbackCandidates(%q) = %v, want to contain %q", c.name, got, c.want)
		}
	}
}

func TestAcronymOfSkipsPunctuationFields(t *testing.T) {
	got := acronymOf("Weights & Biases")
	if got != "wb" {
		t.Errorf("acronymOf(%q) = %q, want %q", "Weights & Biases", got, "wb")
	}
	if got := acronymOf("Noom"); got != "" {
		t.Errorf("acronymOf(single word) = %q, want empty", got)
	}
}
