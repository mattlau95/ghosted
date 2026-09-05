package resolver

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

const userAgent = "ghosted-resolver/1.0 (+mailto:mattlau95@gmail.com)"

// Outcome describes the result of successfully resolving a company to an
// ATS + board token.
type Outcome struct {
	ATS          string
	Token        string
	Confidence   string // high | low
	ResolvedName string
	Notes        string
}

// httpGet performs a GET with a real User-Agent, retrying once on 5xx.
func httpGet(client *http.Client, url string) (status int, body []byte, err error) {
	for attempt := 0; attempt < 2; attempt++ {
		req, reqErr := http.NewRequest(http.MethodGet, url, nil)
		if reqErr != nil {
			return 0, nil, reqErr
		}
		req.Header.Set("User-Agent", userAgent)

		resp, doErr := client.Do(req)
		if doErr != nil {
			if attempt == 0 {
				time.Sleep(1 * time.Second)
				continue
			}
			return 0, nil, doErr
		}
		b, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return resp.StatusCode, nil, readErr
		}

		if resp.StatusCode >= 500 && attempt == 0 {
			time.Sleep(1 * time.Second)
			continue
		}
		return resp.StatusCode, b, nil
	}
	return 0, nil, fmt.Errorf("unreachable")
}

// probeGreenhouse checks the jobs endpoint, and on success verifies the
// board's real company name to guard against a wrong-company false positive.
func probeGreenhouse(client *http.Client, token, companyName string) (*Outcome, error) {
	status, body, err := httpGet(client, fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs", url.PathEscape(token)))
	if err != nil {
		return nil, err
	}
	if status == 404 {
		return nil, nil
	}
	if status != 200 {
		return nil, nil
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, nil
	}
	if _, ok := parsed["jobs"].([]interface{}); !ok {
		return nil, nil
	}

	politeSleep()

	// Verify against the board metadata endpoint.
	confidence := "low"
	resolvedName := ""
	notes := "no board name field returned"
	bStatus, bBody, bErr := httpGet(client, fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s", url.PathEscape(token)))
	if bErr == nil && bStatus == 200 {
		var boardMeta struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(bBody, &boardMeta) == nil && boardMeta.Name != "" {
			resolvedName = boardMeta.Name
			if NormalizeCompanyName(resolvedName) == NormalizeCompanyName(companyName) {
				confidence = "high"
				notes = ""
			} else {
				notes = fmt.Sprintf("board name %q did not match input company name", resolvedName)
			}
		}
	}

	return &Outcome{
		ATS:          "greenhouse",
		Token:        token,
		Confidence:   confidence,
		ResolvedName: resolvedName,
		Notes:        notes,
	}, nil
}

var leverHostedURLRe = regexp.MustCompile(`jobs\.lever\.co/([^/?]+)`)

func probeLever(client *http.Client, token string) (*Outcome, error) {
	status, body, err := httpGet(client, fmt.Sprintf("https://api.lever.co/v0/postings/%s?mode=json", url.PathEscape(token)))
	if err != nil {
		return nil, err
	}
	if status == 404 {
		return nil, nil
	}
	if status != 200 {
		return nil, nil
	}

	var jobs []map[string]interface{}
	if err := json.Unmarshal(body, &jobs); err != nil {
		return nil, nil
	}

	notes := "no company-name field on Lever board endpoint; token match unverified"
	if len(jobs) > 0 {
		if hostedURL, ok := jobs[0]["hostedUrl"].(string); ok {
			if m := leverHostedURLRe.FindStringSubmatch(hostedURL); m != nil {
				notes = fmt.Sprintf("first posting hostedUrl site segment: %q", m[1])
			}
		}
	} else {
		notes = "board has zero postings; token match unverified"
	}

	return &Outcome{
		ATS:        "lever",
		Token:      token,
		Confidence: "low",
		Notes:      notes,
	}, nil
}

var ashbyJobURLRe = regexp.MustCompile(`jobs\.ashbyhq\.com/([^/?]+)`)

func probeAshby(client *http.Client, token string) (*Outcome, error) {
	status, body, err := httpGet(client, fmt.Sprintf("https://api.ashbyhq.com/posting-api/job-board/%s", url.PathEscape(token)))
	if err != nil {
		return nil, err
	}
	if status == 404 {
		return nil, nil
	}
	if status != 200 {
		return nil, nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil
	}
	jobs, ok := raw["jobs"].([]interface{})
	if !ok {
		return nil, nil
	}

	notes := "no company-name field on Ashby board endpoint; token match unverified"
	if len(jobs) > 0 {
		if job, ok := jobs[0].(map[string]interface{}); ok {
			if jobURL, ok := job["jobUrl"].(string); ok {
				if m := ashbyJobURLRe.FindStringSubmatch(jobURL); m != nil {
					notes = fmt.Sprintf("first posting jobUrl site segment: %q", m[1])
				}
			}
		}
	} else {
		notes = "board has zero postings; token match unverified"
	}

	return &Outcome{
		ATS:        "ashby",
		Token:      token,
		Confidence: "low",
		Notes:      notes,
	}, nil
}

// politeSleep waits 300-500ms between requests, per docs/RESOLVER.md.
func politeSleep() {
	time.Sleep(300*time.Millisecond + time.Duration(rand.IntN(201))*time.Millisecond)
}
