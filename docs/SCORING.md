# Stage 3 — Scoring Criteria

What the model uses to rank a posting. Feeds the Claude API call in the fetcher pipeline.

---

## Hard filters (SQL, before any API call)

Cheap exclusions that don't need a model. Run these first so you're not paying to score sales roles.

**Title must match one of:**
`design engineer`, `ux engineer`, `design systems`, `design technologist`, `front-end` / `frontend`, `product designer`, `ui engineer`, `web engineer`, `creative technologist`

**Location must be one of:**
Remote (US), NYC metro, Philadelphia metro, New Jersey. Reject anything requiring relocation or onsite outside these.

**Reject on title:** anything containing `principal`, `director`, `head of`, `VP`, `manager`, `intern`, `contract` (unless remote), or a non-English location string.

Everything surviving this goes to the model.

**`staff` and `senior` are not hard-filtered.** Titles aren't calibrated across companies — staff at a 60-person startup often means what senior means at a large one, and the design engineer market is thin enough that filtering on the word costs more matches than it saves. Let these through and let seniority mismatch reduce Fit instead, so they surface ranked low rather than disappearing outright. See the Fit axis table below.

---

## Scoring axes

Three independent scores, 0–10 each, plus a composite. **Do not collapse them** — a role can be worth applying to on the strength of any single axis, and knowing which one is carrying it changes how you apply.

### 1. Want — do I want this job?

| Signal | Weight |
|---|---|
| Industry I'm actively interested in: browser-based tools, accessibility (especially for low-spec hardware), D2C, health/wellness/sleep, sports, video/SEO/creator tools, education | strong + |
| Product I personally use (Vercel, Linear, Supabase, Claude, Figma) | strong + |
| Open source / developer tooling | mild + |
| Industries I don't want: crypto, gambling, dating apps | strong − |
| Company-tier from the watchlist (`Yes` / `OK` / `Prefer Not`) | modifier |

### 2. Fit — would they want me?

| Signal | Weight |
|---|---|
| Design systems work named explicitly (tokens, component libraries, Figma→code) | strong + |
| Stack overlap: React/TS, vanilla HTML/CSS/JS, Go, Node, Postgres | strong + |
| Accessibility / WCAG named in the posting | strong + |
| Prior domain experience: e-commerce, enterprise travel, graphic design, help-desk/support tooling, education | mild + |
| Seniority reads as mid-level (3–6 yrs) | + |
| Title reads `staff` and the company is large (500+) | strong − |
| Title reads `staff` and the company is small (<150) | mild − |
| Scope described matches design-system ownership regardless of title | + |
| Requires heavy backend, native mobile, or ML engineering | − |
| Requires a degree/credential I don't have | − |

Prior domain experience belongs here, **not in Want.** It's a reason they should hire me, not a reason I want the job.

**Scoring-prompt rationale for `staff`-titled roles:** the candidate has ~3 years enterprise UX but staff-shaped scope — architected an atomic Figma design system and led the first Dev Mode rollout across four product lines. Seniority should be judged on described scope, not years or title alone.

### 3. Access — do I have a way in?

| Signal | Weight |
|---|---|
| Direct connection currently at the company | strong + |
| Direct connection formerly there | + |
| Rutgers CS alum / Collette alum there | mild + |
| Cold | neutral, 0 |

Populated in Stage 5. Until then, score 0 for everything and leave the field.

---

## Output contract

Strict JSON, no prose:

```json
{
  "want": 0-10,
  "fit": 0-10,
  "access": 0-10,
  "composite": 0-100,
  "lead_with": "which portfolio project to lead with",
  "reasoning": "two sentences max",
  "concerns": "what would make this a bad use of an application, or null"
}
```

`composite` = `(want × 3) + (fit × 5) + (access × 2)`. Fit weighted highest because it drives response rate; access second because warm applications convert far better than cold; want lowest because a job that's a strong fit is worth applying to even at an OK company.

---

## Calibration

The scorer will rate everything ~75 unless you anchor it. Include 3 hand-scored examples in the prompt, ideally one obvious yes, one obvious no, and one genuinely marginal role. Write these after reading a week of real postings — not before.

Tell the model explicitly to be harsh and that most postings should score below 50.

---

## Portfolio lead-in mapping

`lead_with` should pick from:

| Project | Lead with it when |
|---|---|
| matthewclau.com | Hand-coded, performance, or vanilla JS is emphasized |
| ollae | Full-stack, Go/Postgres, React 19 |
| pocalab | Print-spec precision, niche community tooling |
| Worship Slides Generator | Node/Express, file parsing, real users |
| Edison Dental 27 | Client delivery, Next.js, small-business work |
| Kumon Chrome extension | Automation, internal tooling, measurable time savings |
| Collette design system | Design systems, Figma atomic system, Dev Mode |

---

## Standing manual routine

Roughly a third of the `Yes`-tier companies aren't on Greenhouse/Lever/Ashby — Chess.com (Rippling), vidIQ (Recruitee), Lockheed, Google, Deloitte, KPMG, Capgemini, Atlassian, Wayfair (Workday/bespoke).

**Weekly calendar block to check these by hand.** Without it, the pipeline quietly biases the search toward companies that were only ever `OK`.
