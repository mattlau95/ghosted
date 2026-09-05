# Stage 1 — ATS Token Resolver

Takes a list of company names, figures out which ATS each one uses and what its board token is. Output feeds the `companies` table.

**Why this exists:** board tokens are customizable and frequently aren't the company name. Asking a model to guess produces plausible-looking URLs that are wrong often enough to be useless. Probing the real endpoints is both faster and correct.

---

## Input

CSV with at least:

```
Company, Why good fit for you
```

Ignore any `Career / Jobs URL` column from the source list — it's unverified. If a row has a URL that is already an ATS board URL *and* was confirmed by hand, pass it in via a separate `known_ats,known_token` column and skip probing for that row.

## Output

```csv
company,ats,board_token,confidence,candidates_tried,resolved_name,needs_review,notes
```

- `confidence`: `high` (name matched) | `low` (200 but name didn't match) | `none`
- `needs_review`: true for anything not `high`

---

## Candidate slug generation

For each company name, generate candidates in this order and try each until one resolves:

1. Lowercase, strip all non-alphanumerics — `Chess.com` → `chesscom`, `Luxury Presence` → `luxurypresence`
2. Lowercase, spaces → hyphens — `Trust Wallet` → `trust-wallet`
3. Strip corporate suffixes first (`Inc`, `LLC`, `Corp`, `Ltd`, `Labs`, `Technologies`, `Group`), then 1 and 2
4. Original casing, no spaces — `Vetcove` → `Vetcove`, `US Mobile` → `USMobile`
5. First word only, lowercased — `Weights & Biases` → `weights`

**Casing matters.** Observed real tokens include `Vetcove` (Ashby) and `USMobile` (Lever), so don't only try lowercase. Dedupe the candidate list before probing.

Cap at 5 candidates per company. Skip a company entirely if it's flagged `manual` in the input (Workday/bespoke enterprise).

---

## Fallback candidates — second pass

Company-name-derived slugs miss when the real token carries a suffix or is an abbreviation. Confirmed real example: **Noom → `greenhouse` / `noomgrowth`**, not `noom`. Other known patterns: `ww` (Weight Watchers), `palantirtechnologies`.

Only tried after the primary 5-candidate pass misses on all three ATS — this keeps the primary probe budget as estimated above, and only spends extra requests on the ~25% that were already going to miss.

1. Base slug (rule 1 above) + each of: `growth`, `careers`, `inc`, `hq`, `us` — e.g. `noom` → `noomgrowth`, `noomcareers`, `noominc`, `noomhq`, `noomus`
2. Acronym of a multi-word name, lowercased, skipping punctuation-only words — `Weight Watchers` → `ww`, `Weights & Biases` → `wb`. Skipped for single-word names.

Same success/verification criteria as the primary pass apply.

---

## Probing

For each candidate, try all three in this order:

| ATS | Probe URL |
|---|---|
| Greenhouse | `https://boards-api.greenhouse.io/v1/boards/{token}/jobs` |
| Lever | `https://api.lever.co/v0/postings/{token}?mode=json` |
| Ashby | `https://api.ashbyhq.com/posting-api/job-board/{token}` |

Use the API endpoints, not the human-facing board URLs — the human URLs redirect and return HTML, which makes success detection unreliable. (Note: the public Greenhouse board display domain is now `job-boards.greenhouse.io`; `boards.greenhouse.io` still redirects there. The API host `boards-api.greenhouse.io` is unchanged and is what the resolver probes.)

### Success criteria — all must hold

1. HTTP 200
2. Body parses as JSON
3. The expected key exists and is an array: Greenhouse `jobs`, Lever (top-level array), Ashby `jobs`

A 200 with an empty array is still a valid board — the company just has no open roles. Record it as resolved.

### False-positive guard

Short or generic slugs resolve to the *wrong company* surprisingly often. After a successful probe, verify:

- Greenhouse: fetch `https://boards-api.greenhouse.io/v1/boards/{token}` and compare the returned `name` to the input company name (normalized, case-insensitive, ignore suffixes)
- Lever/Ashby: no company-name field on the board endpoint, so compare against the first job's `hostedUrl`/`jobUrl` host path, or fall back to `confidence=low`

If names don't match, still record the hit but set `confidence=low` and `needs_review=true`. Do not silently accept.

---

## Politeness

- 300–500ms sleep between HTTP requests, single-threaded
- Set a real `User-Agent` identifying the tool and your email
- Retry once on 5xx with backoff; treat 404 as a clean miss, not an error
- These are public unauthenticated endpoints — reasonable rates are fine, but don't hammer

At 80 companies × up to 5 candidates × 3 ATS, worst case is 1,200 requests. In practice most resolve on candidate 1 or 2. Expect a few minutes.

---

## Expected outcome

- ~50–60% resolve `high` on the first candidate
- ~15% resolve `low` and need a manual glance
- ~25–30% miss entirely — these are Workday, Rippling, Recruitee, SmartRecruiters, or bespoke career pages

For the misses, output them to a `unresolved.csv` and fix by hand: open the company's careers page, click a job, and read the ATS out of the apply URL. Budget an hour.

---

## Known values (skip probing)

Verified by hand, 2026-09-02:

```csv
company,ats,board_token
Ambrook,ashby,ambrook
PermitFlow,ashby,permitflow
Rogo,ashby,rogo
Trust Wallet,ashby,trust-wallet
Vetcove,ashby,Vetcove
Luxury Presence,lever,luxurypresence
US Mobile,lever,USMobile
Noom,greenhouse,noomgrowth
Grammarly,ashby,Superhuman Platform Inc
Canva,smartrecruiters,Canva
Pitch,workable,pitch-software
DoorDash,greenhouse,doordashusa
Weights & Biases,greenhouse,weights_and_biases
Hex,ashby,hex
Ramp,ashby,ramp
Higharc,ashby,higharc
Notion,ashby,notion
Ro,lever,ro
GameChanger,ashby,gamechanger
Buffer,ashby,buffer
Supabase,ashby,supabase
Circle,ashby,circle
Spotify,lever,spotify
HubSpot,greenhouse,hubspot
Sentry,ashby,sentry
Mapbox,ashby,mapbox
Twilio (Segment),greenhouse,twilio
Palantir,lever,palantir
Intercom,greenhouse,intercom
Plaid,ashby,plaid
Oomnitza,lever,oomnitza
```

The last 18 above were eyeballed by hand rather than re-probed: 15 were `low` purely because Ashby/Lever have no board-name field to verify against (token already matched the company name); HubSpot, Twilio (Segment), and Intercom had genuine name mismatches (board names "HubSpot Product," "Twilio," and "Fin" respectively) that were manually confirmed correct anyway.

Known unsupported (mark `manual`, don't probe): Chess.com (Rippling), vidIQ (Recruitee), Ramsey Theory Group (no ATS), Hinge (Match Group), KPMG, Deloitte, Capgemini, Lockheed Martin, Amazon, Google, Shopify, Atlassian, Windmill (email-only applications, no ATS), Etsy (Workday, tenant `wd5` / site `Etsy_Careers`, confirmed from a live posting URL), Retool (Gem, no confirmed public API — posting IDs look opaque/GraphQL-style, not a slug), GitHub (iCIMS, tenant `careers-githubinc` — `/xmlfeed` 200s but returns the empty Angular app shell, not real feed data), Framer (no discoverable ATS at all — no network call, no Apply link/button anywhere in the live DOM on the job page or `/careers` root; needs a human look), Vimeo (no first-party ATS found on `vimeo.com/careers`; only live listing found is via the third-party Vaia marketplace, not confirmed as Vimeo's actual/comprehensive hiring source; needs a human look).

**Possible future adapter — Workday.** Workday exposes JSON at `https://{tenant}.{cluster}.myworkdayjobs.com/wday/cxs/{tenant}/{site}/jobs` via POST with `{limit, offset, searchText}`. Not worth building now: the cluster (`wd5`, `wd1`, `wd103`, …) and site name vary per company with no single derivable URL pattern, and Workday rate-limits more aggressively than the public board APIs. Revisit only if the `manual` list stays large enough to justify it — Etsy, KPMG, Deloitte, Capgemini, Lockheed Martin, and Atlassian are all (likely) Workday, which is the cohort that would eventually justify a fourth response shape.

---

## Manual resolution: cut first, resolve second

The Stage 0 source list was AI-generated and some rows have fabricated or stale details — location claims in particular don't always survive contact with the real careers page. **Before hunting for a board token, check the row is still workable**: remote/timezone requirements, language requirements, hardware/OS requirements, anything else that would make the role a pass regardless of what ATS it's on. If it's a cut, mark it `manual` with the reason and move on — don't spend a probe or a candidate-slug cycle on a company that's getting rejected anyway.

For everything that survives the cut, open the careers page, click a job, and read the ATS off the URL. `go run ./cmd/detect-ats <url>` automates the lookup:

| URL contains | ATS |
|---|---|
| `gh_jid` or `greenhouse.io` | Greenhouse |
| `lever.co` or `lever-origin` | Lever |
| `ashbyhq.com` | Ashby |
| `recruitee.com` | Recruitee |
| `smartrecruiters.com` | SmartRecruiters |
| `jobs.personio.com` or `jobs.personio.de` | Personio — public XML feed at `{token}.jobs.personio.com/xml` |
| `jobs.gem.com` | Gem — mark `manual`, no confirmed public API |
| `apply.workable.com` | Workable — public widget feed at `apply.workable.com/api/v1/widget/accounts/{token}` |
| `icims.com` | iCIMS — mark `manual`; `/xmlfeed` exists on some tenants but is opt-in per client, not reliably public |
| `myworkdayjobs.com` | Workday — mark `manual` |
| `ats.rippling.com` | Rippling — mark `manual` |
| `ycombinator.com/companies/` or `workatastartup.com` | No ATS — YC listing, usually email application — mark `manual` |
| `talents.vaia.com` | No ATS — third-party recruitment marketplace, not the company's own board — mark `manual` |

**Greenhouse embed gotcha:** a company's own careers page using the Greenhouse iframe embed carries `?gh_jid={job_id}` in job links — that only proves the ATS is Greenhouse, it is not the board token. The token is in the embed URL itself:

```
https://job-boards.greenhouse.io/embed/job_board?for={token}
```

Find it via the iframe's `src` attribute (view page source, or inspect the embedded frame) rather than trying to derive it from the visible job link.

**JS-rendered embeds need a real browser, not `curl`.** Some careers pages (Next.js/React SPAs) render an empty `<div id="ashby_embed">` in the static HTML and inject the actual iframe client-side after a JS bundle runs — `curl`/`view-source` will show nothing useful. Load the page in a browser and inspect the live DOM (`document.querySelectorAll('iframe')`) or the network tab for the resulting `jobs.ashbyhq.com`/`job-boards.greenhouse.io` request instead.

**Ashby tokens aren't always a slug.** Confirmed real example: Grammarly's hiring is actually branded and hosted as **Superhuman** (Grammarly acquired Superhuman) at `superhuman.com/company/careers/jobs`, and its Ashby board token is the literal, space-containing org name `Superhuman Platform Inc` — not `superhuman` or any candidate our slug generator would produce. Verified via `https://api.ashbyhq.com/posting-api/job-board/Superhuman%20Platform%20Inc` (200, real jobs). The resolver URL-escapes tokens before probing (`net/url.PathEscape`) so a token like this works once you have it by hand — it just can't be *guessed*.

**Acquisitions move hiring to the acquirer's board.** Seen twice now: Grammarly/Superhuman above, and **Weights & Biases**, acquired by CoreWeave — its listings live on `coreweave.com/careers` with `?board=weights_and_biases` in the URL, and resolve on Greenhouse token `weights_and_biases` (board name literally `"Weights & Biases"`, W&B-specific job titles), not `coreweave` (CoreWeave's own general board, also live, but the wrong one). When a `board=` or similar disambiguating param shows up in a job URL, that's usually the real signal — probe the parent company's likely tokens *and* the acquired brand's name as a token on the parent's ATS.

---

## Second pass — optional

If a company misses on all three, add Recruitee, SmartRecruiters, and Workable, all of which have public JSON:

- Recruitee: `https://{token}.recruitee.com/api/offers/`
- SmartRecruiters: `https://api.smartrecruiters.com/v1/companies/{token}/postings`
- Workable: `https://apply.workable.com/api/v1/widget/accounts/{token}` — returns `{name, description, jobs: [...]}`, empty `jobs` array is a valid resolved-but-no-openings result, same as Greenhouse

Only worth doing if the miss rate is high enough to matter. Adding a fourth-plus response shape to normalize has a real cost in Stage 2a.

Confirmed live: **Canva → SmartRecruiters, token `Canva`** (264 postings). **Pitch → Workable, token `pitch-software`** (0 current openings, board resolved and verified via `name:"Pitch"` match). Two confirmed non-primary ATSes now — still not enough to force the Stage 2a decision on their own, but worth revisiting once a couple more watchlist companies land on either.
