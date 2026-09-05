package fetcher

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchGreenhouseUnescapesContent(t *testing.T) {
	body := `{"jobs":[{"id":123,"title":"Design Engineer","absolute_url":"https://job-boards.greenhouse.io/acme/jobs/123","content":"&lt;p&gt;Build things&lt;/p&gt;","location":{"name":"Remote"}}]}`
	srv := testServer(t, http.StatusOK, body)

	postings, err := fetchGreenhouseFrom(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("fetchGreenhouseFrom: %v", err)
	}
	if len(postings) != 1 {
		t.Fatalf("got %d postings, want 1", len(postings))
	}
	p := postings[0]
	if p.ATS != "greenhouse" || p.ATSJobID != "123" {
		t.Errorf("ATS/ATSJobID = %q/%q", p.ATS, p.ATSJobID)
	}
	if p.Title != "Design Engineer" || p.Location != "Remote" {
		t.Errorf("Title/Location = %q/%q", p.Title, p.Location)
	}
	if p.URL != "https://job-boards.greenhouse.io/acme/jobs/123" {
		t.Errorf("URL = %q", p.URL)
	}
	if want := "<p>Build things</p>"; p.Description != want {
		t.Errorf("Description = %q, want %q (HTML-entity unescape)", p.Description, want)
	}
}

func TestFetchLever(t *testing.T) {
	body := `[{"id":"abc-123","text":"Product Designer","hostedUrl":"https://jobs.lever.co/acme/abc-123","categories":{"location":"NYC"},"description":"<p>Design stuff</p>"}]`
	srv := testServer(t, http.StatusOK, body)

	postings, err := fetchLeverFrom(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("fetchLeverFrom: %v", err)
	}
	if len(postings) != 1 {
		t.Fatalf("got %d postings, want 1", len(postings))
	}
	p := postings[0]
	if p.ATS != "lever" || p.ATSJobID != "abc-123" {
		t.Errorf("ATS/ATSJobID = %q/%q", p.ATS, p.ATSJobID)
	}
	if p.Title != "Product Designer" || p.Location != "NYC" {
		t.Errorf("Title/Location = %q/%q", p.Title, p.Location)
	}
	if p.URL != "https://jobs.lever.co/acme/abc-123" {
		t.Errorf("URL = %q", p.URL)
	}
	if p.Description != "<p>Design stuff</p>" {
		t.Errorf("Description = %q", p.Description)
	}
}

func TestFetchAshby(t *testing.T) {
	body := `{"jobs":[{"id":"uuid-1","title":"UX Engineer","location":"Remote","jobUrl":"https://jobs.ashbyhq.com/acme/uuid-1","descriptionHtml":"<p>Build UI</p>"}]}`
	srv := testServer(t, http.StatusOK, body)

	postings, err := fetchAshbyFrom(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("fetchAshbyFrom: %v", err)
	}
	if len(postings) != 1 {
		t.Fatalf("got %d postings, want 1", len(postings))
	}
	p := postings[0]
	if p.ATS != "ashby" || p.ATSJobID != "uuid-1" {
		t.Errorf("ATS/ATSJobID = %q/%q", p.ATS, p.ATSJobID)
	}
	if p.Title != "UX Engineer" || p.Location != "Remote" {
		t.Errorf("Title/Location = %q/%q", p.Title, p.Location)
	}
	if p.Description != "<p>Build UI</p>" {
		t.Errorf("Description = %q", p.Description)
	}
}

func TestFetchGreenhouseNon200(t *testing.T) {
	srv := testServer(t, http.StatusNotFound, "not found")
	if _, err := fetchGreenhouseFrom(srv.Client(), srv.URL); err == nil {
		t.Error("expected error on 404, got nil")
	}
}

func TestFetchLeverMalformedJSON(t *testing.T) {
	srv := testServer(t, http.StatusOK, "not json")
	_, err := fetchLeverFrom(srv.Client(), srv.URL)
	if err == nil {
		t.Fatal("expected error on malformed JSON, got nil")
	}
	if !strings.Contains(err.Error(), "lever board") {
		t.Errorf("error = %v, want it to mention the board", err)
	}
}

func TestFetchAshbyEmptyBoard(t *testing.T) {
	srv := testServer(t, http.StatusOK, `{"jobs":[]}`)
	postings, err := fetchAshbyFrom(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("fetchAshbyFrom: %v", err)
	}
	if len(postings) != 0 {
		t.Errorf("got %d postings, want 0", len(postings))
	}
}
