package fetcher

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
)

const userAgent = "ghosted-fetcher/1.0 (+mailto:mattlau95@gmail.com)"

// Posting is one job listing normalized from any ATS's response shape.
// CompanyID is filled in by the caller, which knows which company this
// fetch belongs to.
type Posting struct {
	ATS         string
	ATSJobID    string
	Title       string
	Location    string
	URL         string
	Description string
}

func httpGet(client *http.Client, url string) (int, []byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}

// FetchGreenhouse pulls every open posting from a Greenhouse board.
func FetchGreenhouse(client *http.Client, token string) ([]Posting, error) {
	return fetchGreenhouseFrom(client, fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs?content=true", token))
}

func fetchGreenhouseFrom(client *http.Client, url string) ([]Posting, error) {
	status, body, err := httpGet(client, url)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("greenhouse board: unexpected status %d", status)
	}

	var parsed struct {
		Jobs []struct {
			ID          int64  `json:"id"`
			Title       string `json:"title"`
			AbsoluteURL string `json:"absolute_url"`
			Content     string `json:"content"`
			Location    struct {
				Name string `json:"name"`
			} `json:"location"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("greenhouse board: %w", err)
	}

	postings := make([]Posting, 0, len(parsed.Jobs))
	for _, j := range parsed.Jobs {
		postings = append(postings, Posting{
			ATS:      "greenhouse",
			ATSJobID: strconv.FormatInt(j.ID, 10),
			Title:    j.Title,
			Location: j.Location.Name,
			URL:      j.AbsoluteURL,
			// Greenhouse's content field is HTML-entity-escaped HTML.
			Description: html.UnescapeString(j.Content),
		})
	}
	return postings, nil
}

// FetchLever pulls every open posting from a Lever board.
func FetchLever(client *http.Client, token string) ([]Posting, error) {
	return fetchLeverFrom(client, fmt.Sprintf("https://api.lever.co/v0/postings/%s?mode=json", token))
}

func fetchLeverFrom(client *http.Client, url string) ([]Posting, error) {
	status, body, err := httpGet(client, url)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("lever board: unexpected status %d", status)
	}

	var jobs []struct {
		ID         string `json:"id"`
		Text       string `json:"text"`
		HostedURL  string `json:"hostedUrl"`
		Categories struct {
			Location string `json:"location"`
		} `json:"categories"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(body, &jobs); err != nil {
		return nil, fmt.Errorf("lever board: %w", err)
	}

	postings := make([]Posting, 0, len(jobs))
	for _, j := range jobs {
		postings = append(postings, Posting{
			ATS:         "lever",
			ATSJobID:    j.ID,
			Title:       j.Text,
			Location:    j.Categories.Location,
			URL:         j.HostedURL,
			Description: j.Description,
		})
	}
	return postings, nil
}

// FetchAshby pulls every open posting from an Ashby job board.
func FetchAshby(client *http.Client, token string) ([]Posting, error) {
	return fetchAshbyFrom(client, fmt.Sprintf("https://api.ashbyhq.com/posting-api/job-board/%s", token))
}

func fetchAshbyFrom(client *http.Client, url string) ([]Posting, error) {
	status, body, err := httpGet(client, url)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("ashby board: unexpected status %d", status)
	}

	var parsed struct {
		Jobs []struct {
			ID              string `json:"id"`
			Title           string `json:"title"`
			Location        string `json:"location"`
			JobURL          string `json:"jobUrl"`
			DescriptionHTML string `json:"descriptionHtml"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("ashby board: %w", err)
	}

	postings := make([]Posting, 0, len(parsed.Jobs))
	for _, j := range parsed.Jobs {
		postings = append(postings, Posting{
			ATS:         "ashby",
			ATSJobID:    j.ID,
			Title:       j.Title,
			Location:    j.Location,
			URL:         j.JobURL,
			Description: j.DescriptionHTML,
		})
	}
	return postings, nil
}
