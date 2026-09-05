package scorer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
)

const model = "claude-haiku-4-5"

// Posting is the subset of a posting's fields the scorer needs.
type Posting struct {
	CompanyName string
	CompanyTier string
	Title       string
	Location    string
	Description string
}

// Score is the model's output, matching docs/SCORING.md's contract exactly.
type Score struct {
	Want      int    `json:"want"`
	Fit       int    `json:"fit"`
	Access    int    `json:"access"`
	Composite int    `json:"composite"`
	LeadWith  string `json:"lead_with"`
	Reasoning string `json:"reasoning"`
	Concerns  string `json:"concerns"`
}

var scoreSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"want":      map[string]any{"type": "integer", "description": "0-10"},
		"fit":       map[string]any{"type": "integer", "description": "0-10"},
		"access":    map[string]any{"type": "integer", "description": "0-10"},
		"composite": map[string]any{"type": "integer", "description": "(want*3)+(fit*5)+(access*2), 0-100"},
		"lead_with": map[string]any{"type": "string"},
		"reasoning": map[string]any{"type": "string", "description": "two sentences max"},
		"concerns": map[string]any{
			"anyOf": []map[string]any{
				{"type": "string"},
				{"type": "null"},
			},
		},
	},
	"required":             []string{"want", "fit", "access", "composite", "lead_with", "reasoning", "concerns"},
	"additionalProperties": false,
}

// systemPrompt encodes docs/SCORING.md's rubric. Calibration examples are
// deliberately absent — SCORING.md: "Write these after reading a week of
// real postings — not before." Add 2-3 hand-scored examples here once a
// week of real output has been read (see docs/DEVLOG.md).
const systemPrompt = `You score job postings for one specific candidate applying to Design Engineer / design systems roles. Access is currently always 0 (no connection data yet).

Candidate profile:
- ~3 years experience, reads as mid-level. Prior domain experience: e-commerce, enterprise travel (built an atomic Figma design system with Dev Mode rollout across four product lines at a travel company), graphic design, help-desk/support tooling, education.
- Stack: React/TS, vanilla HTML/CSS/JS, Go, Node, Postgres.
- Portfolio to pick lead_with from:
  - matthewclau.com: hand-coded, performance, vanilla JS
  - ollae: full-stack, Go/Postgres, React 19
  - pocalab: print-spec precision, niche community tooling
  - Worship Slides Generator: Node/Express, file parsing, real users
  - Edison Dental 27: client delivery, Next.js, small-business work
  - Kumon Chrome extension: automation, internal tooling, measurable time savings
  - Collette design system: design systems, Figma atomic system, Dev Mode

Score three independent axes, 0-10 each. Do not collapse them - a role can be worth applying to on the strength of any single axis.

1. Want - do I want this job? Strong +: browser-based tools, accessibility (especially low-spec hardware), D2C, health/wellness/sleep, sports, video/SEO/creator tools, education, a product the candidate personally uses (Vercel, Linear, Supabase, Claude, Figma). Mild +: open source / developer tooling. Strong -: crypto, gambling, dating apps. Company tier (Yes/OK/Prefer Not) is a modifier.

2. Fit - would they want me? Strong +: design systems work named explicitly (tokens, component libraries, Figma-to-code), stack overlap, accessibility/WCAG named in the posting. Mild +: prior domain experience overlap (this belongs here, not Want). +: seniority reads mid-level (3-6 yrs). Title reads "staff" and company is large (500+): strong -. Title reads "staff" and company is small (<150): mild -. Scope described matches design-system ownership regardless of title: +. -: requires heavy backend, native mobile, or ML engineering; requires a degree/credential the candidate doesn't have.

Staff-title rationale: judge seniority by described scope, not the title word or years. The candidate's own background is staff-shaped scope (architected an atomic Figma design system, led a Dev Mode rollout across four product lines) at only ~3 years - a "staff" or "senior" title should not be rejected outright, score the actual scope described.

3. Access - do I have a way in? Currently always 0 until connections data exists (Stage 5, not yet built).

composite = (want * 3) + (fit * 5) + (access * 2). Fit weighted highest because it drives response rate.

Be harsh. An uncalibrated scorer rates everything ~75 - most postings should score below 50. Only strong, specific matches should score high.

concerns: what would make this a bad use of an application, or null if none. reasoning: two sentences max. Respond with the JSON object only.`

// ScorePosting calls Claude Haiku once for a single posting and returns its score.
func ScorePosting(ctx context.Context, client anthropic.Client, p Posting) (Score, error) {
	userContent := fmt.Sprintf(
		"Company: %s (tier: %s)\nTitle: %s\nLocation: %s\n\nDescription:\n%s",
		p.CompanyName, p.CompanyTier, p.Title, p.Location, truncate(p.Description, 8000),
	)

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		OutputConfig: anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{Schema: scoreSchema},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userContent)),
		},
	})
	if err != nil {
		return Score{}, err
	}

	if resp.StopReason == anthropic.StopReasonRefusal {
		return Score{}, fmt.Errorf("scoring refused by safety classifier")
	}
	if resp.StopReason == anthropic.StopReasonMaxTokens {
		return Score{}, fmt.Errorf("response truncated at max_tokens")
	}

	var text string
	for _, block := range resp.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			text = tb.Text
			break
		}
	}
	if text == "" {
		return Score{}, fmt.Errorf("no text content in response")
	}

	var s Score
	if err := json.Unmarshal([]byte(text), &s); err != nil {
		return Score{}, fmt.Errorf("parsing score JSON: %w", err)
	}
	return s, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
