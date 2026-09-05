package resolver

import (
	"net/url"
	"strings"
)

// DetectATS looks for known ATS signatures in a job posting URL, for
// hand-resolving companies that missed automated probing. It only
// identifies the ATS — for Greenhouse, the gh_jid query param is the job
// ID, not the board token; use ExtractGreenhouseEmbedToken or check the
// page's embed iframe for that.
func DetectATS(jobURL string) (ats string, manual bool) {
	switch {
	case strings.Contains(jobURL, "greenhouse.io") || strings.Contains(jobURL, "gh_jid"):
		return "greenhouse", false
	case strings.Contains(jobURL, "lever.co") || strings.Contains(jobURL, "lever-origin"):
		return "lever", false
	case strings.Contains(jobURL, "ashbyhq.com"):
		return "ashby", false
	case strings.Contains(jobURL, "recruitee.com"):
		return "recruitee", false
	case strings.Contains(jobURL, "smartrecruiters.com"):
		return "smartrecruiters", false
	case strings.Contains(jobURL, "jobs.personio.com") || strings.Contains(jobURL, "jobs.personio.de"):
		return "personio", false
	case strings.Contains(jobURL, "jobs.gem.com"):
		return "gem", true
	case strings.Contains(jobURL, "apply.workable.com"):
		return "workable", false
	case strings.Contains(jobURL, "icims.com"):
		return "icims", true
	case strings.Contains(jobURL, "myworkdayjobs.com"):
		return "workday", true
	case strings.Contains(jobURL, "ats.rippling.com"):
		return "rippling", true
	case strings.Contains(jobURL, "ycombinator.com/companies/") || strings.Contains(jobURL, "workatastartup.com"):
		return "none (YC listing)", true
	case strings.Contains(jobURL, "talents.vaia.com"):
		return "none (Vaia marketplace)", true
	default:
		return "", false
	}
}

// ExtractGreenhouseEmbedToken pulls the board token out of a Greenhouse
// iframe embed URL: https://job-boards.greenhouse.io/embed/job_board?for={token}
// A company's own careers page carries only ?gh_jid={job_id} in its job
// links, which does not contain the token — that has to come from the
// embed iframe's src (or the page source) instead.
func ExtractGreenhouseEmbedToken(embedURL string) (token string, ok bool) {
	u, err := url.Parse(embedURL)
	if err != nil {
		return "", false
	}
	token = u.Query().Get("for")
	return token, token != ""
}
