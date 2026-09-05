package scorer

import "testing"

func TestPassesHardFilters(t *testing.T) {
	cases := []struct {
		name     string
		title    string
		location string
		want     bool
	}{
		{"design engineer remote", "Design Engineer", "Remote", true},
		{"design engineer NYC", "Design Engineer", "New York City, NY", true},
		{"design engineer NJ", "UX Engineer", "Fort Lee, NJ", true},
		{"design engineer philly", "Product Designer", "Philadelphia, PA", true},
		{"no title match", "Software Engineer", "Remote", false},
		{"director rejected", "Director of Design Engineering", "Remote", false},
		{"manager rejected", "Design Systems Manager", "Remote", false},
		{"intern rejected", "Design Engineer Intern", "Remote", false},
		{"contract not remote rejected", "Design Engineer (Contract)", "San Francisco, CA", false},
		{"contract remote allowed", "Design Engineer (Contract)", "Remote", true},
		{"onsite elsewhere rejected", "Design Engineer", "San Francisco, CA", false},
		{"remote non-US rejected", "Design Engineer", "Remote (UK)", false},
		{"remote EMEA rejected", "Design Engineer", "Remote - EMEA", false},
		{"non-ASCII location rejected", "Design Engineer", "Zürich, Switzerland", false},
		{"staff title not hard-filtered", "Staff Design Engineer", "Remote", true},
		{"senior title not hard-filtered", "Senior Product Designer", "NYC", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := PassesHardFilters(c.title, c.location)
			if got.Pass != c.want {
				t.Errorf("PassesHardFilters(%q, %q) = %v (reason: %q), want pass=%v",
					c.title, c.location, got.Pass, got.Reason, c.want)
			}
		})
	}
}
