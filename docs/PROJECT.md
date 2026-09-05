# ghosted — Project Spec

Personal job discovery + application tracking. Go / Postgres.

---

## Goals

**Primary:** Know about every relevant opening at companies I'd actually want to work at, within a day of it posting, without manually checking career pages.

**Secondary:** Capture applications in one tap from any device, so the tracker stays current.

**Tertiary:** It's a shipped service solving a real problem — a portfolio piece for Design Engineer applications.

## Non-goals

- No auto-applying
- No resume rewriting or keyword-mirroring
- No LinkedIn scraping
- Not a general job board — watchlist companies only
- No dashboard/analytics UI until the data is worth looking at

## Success metric

At 30 days: I can name at least one role I applied to that I'd have missed otherwise.
If not, the watchlist is wrong — not the code.

---

## Architecture

**Cost target: $0 through Stage 5.** Only the capture endpoint ever needs hosting, and it's deferred until the project has earned it.

| Component | Where | Runs | Cost |
|---|---|---|---|
| **Fetcher** | Main laptop, Task Scheduler | Daily, headless | $0 |
| **Postgres** | Neon free tier | Always | $0 |
| **Capture app** | Deferred — see Stage 2b | HTTP, single-user | $0–2/mo |

Postgres must be a hosted free tier rather than local, so the fetcher (laptop) and capture app (wherever it lands) share one database. `pg_trgm` needs to be available for Stage 5.

Missing a day's fetch when the laptop is off is acceptable — postings don't move that fast.

**Ruled out:** the DS218j NAS (ARMv7, 512MB, no Synology Docker for ARM) can't host any of this. It's still the `pg_dump` backup target over SMB.

---

## Data sources

Keyless, first-party ATS job board APIs. No auth, no anti-bot, full descriptions inline.

| ATS | Endpoint |
|---|---|
| Greenhouse | `boards-api.greenhouse.io/v1/boards/{token}/jobs?content=true` |
| Lever | `api.lever.co/v0/postings/{site}?mode=json` |
| Ashby | `api.ashbyhq.com/posting-api/job-board/{name}` |

Board token = the slug in the board URL (`job-boards.greenhouse.io/anthropic` → `anthropic`).

**Constraint:** these are per-company, not searchable. The company watchlist is the input, not an output.

---

## Schema

```sql
companies (
  id, name, name_norm, ats, board_token, active, notes
)

postings (
  id, company_id, ats, ats_job_id, title, location, url,
  description, first_seen, last_seen, closed_at,
  -- application tracking
  status,           -- null | interested | applied | screening | onsite | offer | rejected
  applied_at, last_contact, next_action, contact_name, notes,
  source            -- board | referral | direct | recruiter
)

scores (
  posting_id, fit_score, reasoning, stack_overlap, concerns, model, scored_at
)

connections (
  id, name, company, company_norm, title, tier, last_verified
)
```

Dedupe key: `(ats, ats_job_id)` unique. Manually-captured jobs from non-ATS sources key on normalized URL.

---

## Stages

Each stage produces something usable even if the next is never built.

### Stage 0 — Watchlist
**Tool:** Claude (chat) · **Effort:** a few evenings, no code

40–60 companies worth working at. Sources: products I admire, network overlap (Rutgers, Collette alumni), NJ/NYC product companies and studios, companies already using the "Design Engineer" title.

**Done when:** spreadsheet with `company, why, careers_url`.
**Note:** the only stage with no code, and the one that determines whether any of this is useful.

---

### Stage 1 — Token resolution
**Tool:** Claude Code · **Effort:** one afternoon

Script tries each company slug against all three ATS endpoints, keeps whichever returns 200. Expect ~60% auto-resolve; fix the rest by hand from careers page URLs.

**Done when:** `companies` table populated, remainder marked `unsupported`.

---

### Stage 2a — Fetcher
**Tool:** Claude Code · **Effort:** one weekend

Go binary. Pulls all boards, normalizes three response shapes into one struct, upserts into `postings`, bumps `last_seen`, sets `closed_at` on anything not seen this run. No scoring, no delivery, no HTTP. Run locally.

**Done when:** running it twice adds zero rows the second time.

---

### Stage 2b — Capture *(deferred)*
**Tool:** Claude Design (UI) → Claude Code (implementation) · **Effort:** one weekend

**Don't build this until Stages 2a–4 have run for a few weeks.** It's the only piece that requires paid always-on hosting, and it's only worth it if phone capture turns out to be something I actually reach for. Until then, log applications in a spreadsheet.

Hosting when the time comes, cheapest first:
- **Cloudflare Workers** — already on Cloudflare, generous free tier, and a small POST handler is exactly its shape
- **Fly** — single 256MB Machine, ~$2/mo now that Postgres lives elsewhere

`POST /jobs` — takes a URL, routes to ATS-parse (structured, authoritative) or LLM-parse (fetch HTML, Haiku extracts `{company, role, location, seniority}`), upserts with `status='applied'`.

`GET /review` — server-rendered page listing recent captures with pre-filled, editable fields. This is a **correction** surface, not a data-entry form.

Auth: long-lived cookie via `/login?token=<random>`. Visit once per device. Swap for Cloudflare Access if it becomes annoying.

**Done when:** I can log a job from my phone in one tap and fix the parse later from a laptop.

---

### Stage 3 — Filter + score
**Tool:** Claude Code · **Effort:** one day + a week of tuning

SQL pre-filter on title/location, then one Haiku call per *new* posting only. Returns strict JSON: `{score 0-100, reasoning, stack_overlap, concerns}`. Prompt with 2–3 hand-scored examples for calibration — an un-calibrated scorer rates everything 75.

Print to stdout first. Read a week of output before automating delivery.

**Done when:** I agree with the ranking.

---

### Stage 4 — Schedule + deliver
**Tool:** Claude Code · **Effort:** half a day

Fetcher runs daily via Windows Task Scheduler (`schtasks`, or the GUI). The binary runs and exits — no long-running process, no ticker.

Nightly email digest of new postings above threshold. An empty day sends nothing, not an empty email.

Nightly `pg_dump` to a NAS folder over SMB, so it's covered by whatever the NAS already backs up. Five minutes to set up; the thing everyone skips.

**Done when:** it runs without me.

---

### Stage 5 — Network overlap
**Tool:** Claude Code · **Effort:** one day

Export LinkedIn connections (Settings → Data Privacy → Get a copy of your data → Connections). Load into `connections`. Match `posting.company` against `connections.company_norm` with `pg_trgm` similarity > 0.6, plus a manual alias table for the ones fuzzy matching misses.

Tier the flags:
1. Direct connection currently there — warmest
2. Direct connection formerly there — intel
3. Alum/weak tie — cold opener
4. Nobody — normal application

Re-export quarterly; the CSV is a snapshot and rots.

**Done when:** the digest tiers roles by connection strength.

---

### Stage 6 — Chrome extension + iOS Shortcut
**Tool:** Claude Code · **Effort:** half a day

Manifest V3 extension, single button, grabs `document.location.href`, POSTs to `/jobs`. iOS Shortcut in the share sheet does the same from Safari.

Both hit the existing handler — no new backend work.

**Done when:** capture works from all three devices.

---

## Tool split

| Work | Tool |
|---|---|
| Watchlist research, ATS identification, scoring criteria, weekly review | Claude (chat) |
| `/review` page layout, digest email design, extension popup | Claude Design |
| All Go, SQL, Fly config, extension code | Claude Code |

Repo keeps `PROJECT.md` (this file) as the source of truth. Windows/PowerShell locally.

---

## Open decisions

- Email delivery: Resend vs. Postmark vs. plain SMTP
- Whether applied-job tracking stays in Postgres or mirrors into Linear
- Score threshold for the digest (start at 70, tune)

**Resolved:** hosting. Laptop + free-tier Postgres through Stage 5; capture app deferred and cheap when it lands.
**Resolved:** Postgres host is Neon — serverless/scale-to-zero fits a fetcher that only runs once daily, and `pg_trgm` is supported out of the box for Stage 5.

---

## Guardrails

This is a side project during an active job search. The portfolio and warm outreach are what convert; this tool is not. **If it starts eating weeks, that's the signal it's become productive procrastination.** Ship Stage 2a and 2b, then reassess before continuing.

Applications sent is the easiest metric to track and the least informative. Track it to notice stalls, not as a score.
