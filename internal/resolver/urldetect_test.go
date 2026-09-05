package resolver

import "testing"

func TestDetectATS(t *testing.T) {
	cases := []struct {
		url        string
		wantATS    string
		wantManual bool
	}{
		{"https://boards.greenhouse.io/anthropic/jobs/123", "greenhouse", false},
		{"https://example.com/careers/role?gh_jid=987654", "greenhouse", false},
		{"https://jobs.lever.co/example/abc-123", "lever", false},
		{"https://jobs.ashbyhq.com/example/xyz", "ashby", false},
		{"https://example.recruitee.com/o/role", "recruitee", false},
		{"https://api.smartrecruiters.com/v1/companies/example/postings/1", "smartrecruiters", false},
		{"https://kaleidos.jobs.personio.com/job/12345", "personio", false},
		{"https://example.jobs.personio.de/job/12345", "personio", false},
		{"https://jobs.gem.com/retool/am9icG9zdDpdvI4F22-EVGeXat61zM8n", "gem", true},
		{"https://apply.workable.com/pitch-software/", "workable", false},
		{"https://careers-githubinc.icims.com/jobs/5759", "icims", true},
		{"https://example.myworkdayjobs.com/careers/job/role", "workday", true},
		{"https://ats.rippling.com/example/jobs/1", "rippling", true},
		{"https://www.ycombinator.com/companies/example/jobs/abc", "none (YC listing)", true},
		{"https://www.workatastartup.com/jobs/12345", "none (YC listing)", true},
		{"https://talents.vaia.com/companies/vimeo/some-role-12345/", "none (Vaia marketplace)", true},
		{"https://example.com/careers", "", false},
	}
	for _, c := range cases {
		ats, manual := DetectATS(c.url)
		if ats != c.wantATS || manual != c.wantManual {
			t.Errorf("DetectATS(%q) = (%q, %v), want (%q, %v)", c.url, ats, manual, c.wantATS, c.wantManual)
		}
	}
}

func TestExtractGreenhouseEmbedToken(t *testing.T) {
	token, ok := ExtractGreenhouseEmbedToken("https://job-boards.greenhouse.io/embed/job_board?for=noomgrowth")
	if !ok || token != "noomgrowth" {
		t.Errorf("ExtractGreenhouseEmbedToken() = (%q, %v), want (%q, true)", token, ok, "noomgrowth")
	}

	if _, ok := ExtractGreenhouseEmbedToken("https://example.com/careers"); ok {
		t.Error("ExtractGreenhouseEmbedToken() = ok=true for a URL with no for= param")
	}
}
