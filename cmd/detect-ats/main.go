// detect-ats identifies which ATS a job posting URL belongs to, for
// hand-resolving companies in docs/unresolved.csv. Usage:
//
//	go run ./cmd/detect-ats <url>
package main

import (
	"fmt"
	"log"
	"os"

	"ghosted/internal/resolver"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: detect-ats <job-or-embed-url>")
	}
	target := os.Args[1]

	if token, ok := resolver.ExtractGreenhouseEmbedToken(target); ok {
		fmt.Printf("greenhouse / %s (from embed URL)\n", token)
		return
	}

	ats, manual := resolver.DetectATS(target)
	if ats == "" {
		fmt.Println("no known ATS signature found in this URL")
		return
	}

	fmt.Printf("ats: %s\n", ats)
	if manual {
		fmt.Println("mark manual — no public JSON API for this ATS")
	}
	if ats == "greenhouse" {
		fmt.Println("gh_jid is the job ID, not the board token.")
		fmt.Println("find the board token via the page's embed iframe src:")
		fmt.Println("  https://job-boards.greenhouse.io/embed/job_board?for={token}")
		fmt.Println("or view page source and search for job-boards.greenhouse.io")
	}
	if ats == "personio" {
		fmt.Println("token is the subdomain segment: https://{token}.jobs.personio.com/...")
		fmt.Println("public XML feed: https://{token}.jobs.personio.com/xml")
	}
	if ats == "none (YC listing)" {
		fmt.Println("YC company page — usually email application, no ATS API")
	}
	if ats == "none (Vaia marketplace)" {
		fmt.Println("third-party recruitment marketplace, not the company's own ATS — check the company's own careers page too")
	}
	if ats == "gem" {
		fmt.Println("no confirmed public JSON API for Gem — posting IDs look opaque/GraphQL-style")
	}
	if ats == "workable" {
		fmt.Println("token is the path segment: https://apply.workable.com/{token}/")
		fmt.Println("public widget feed: https://apply.workable.com/api/v1/widget/accounts/{token}")
	}
	if ats == "icims" {
		fmt.Println("no reliably public feed — /xmlfeed exists on some tenants but is opt-in per client")
		fmt.Println("check https://{tenant}.icims.com/xmlfeed by hand; if it 200s with real XML (not the app shell), it's usable")
	}
}
