# Devlog: ghosted

## Template (Copy this for new entries)
## [YYYY-MM-DD] - [Summary]
**Session Goal:** [Goal]
**Status:** [Completed/Partially Completed/Blocked]

### The "Why" (Decision Log)
* **Resolution:** [Why this was the right path]

### Technical Notes
* [Stack changes, bugs, or refactors]

### Next Session
* [Task 1]

---
## History (Log Entries start here)

## [2026-09-02] - Stage 0 watchlist finalized, Stage 1 resolver built and run
**Session Goal:** Lock in the Stage 0 company watchlist and build + run the Stage 1 ATS token resolver.
**Status:** Completed

### The "Why" (Decision Log)
* **Resolution:** `docs/companies.csv` holds the real watchlist (79 companies, tiered `Yes`/`OK`/`Prefer Not` with personal notes and connections) and is gitignored since it's personal data; `docs/companies.example.csv` stays tracked as a sanitized schema reference for the resolver's expected input shape.
* **Resolution:** ~13 companies are pre-flagged `manual` (Workday/bespoke/no-ATS — KPMG, AWS/Amazon, Google, Deloitte, Capgemini, Chess.com, Ramsey Theory Group, Lockheed Martin, Hinge, vidIQ, Shopify, Wayfair, Atlassian) per RESOLVER.md guidance, so the resolver skips probing for these. ~10 more already had `known_ats`/`known_token` set by hand and were accepted as `confidence=high` without re-probing.
* **Resolution:** For Lever/Ashby, there's no board-level company-name field to verify against (unlike Greenhouse's `/boards/{token}` metadata endpoint), so per RESOLVER.md those always land at `confidence=low` / `needs_review=true` even on a clean hit — the first posting's `hostedUrl`/`jobUrl` site segment is recorded in `notes` for a quick human glance, but it can't rule out a wrong-company false positive on its own.
* **Resolution:** `unresolved.csv` was initially over-inclusive — it counted `manual`-flagged companies (deliberately skipped, not missed) alongside genuine probe misses. Fixed to key off `ats == ""` instead of `confidence == "none"`, since manual rows use `ats="manual"`. Manual companies belong to SCORING.md's "standing manual routine" (weekly hand-check), not the resolver's one-time cleanup list.

### Technical Notes
* Go module `ghosted` (Go 1.26) with `internal/resolver` (slug generation, HTTP probing w/ retry-once-on-5xx + 300–500ms politeness delay, CSV I/O) and `cmd/resolver` (CLI: `-input`/`-output`/`-unresolved` flags, defaults to `docs/companies.csv` → `docs/companies.resolved.csv` + `docs/unresolved.csv`).
* Added `companies.csv` and `companies.resolved.csv` to `.gitignore` (both derived from/containing personal data); `unresolved.csv` was already ignored.
* Unit test (`slugs_test.go`) checks candidate generation against the five worked examples in RESOLVER.md before spending live HTTP requests.
* Smoke-tested against 5 known real companies before running the full list, to confirm the Greenhouse name-verification and Lever/Ashby URL-segment checks actually fire correctly.
* Full run against all 79 companies: **35 high, 18 low (needs review), 13 missed, 13 manual** — in line with RESOLVER.md's expected ~50–60%/~15%/~25–30% split once manual rows are excluded from the denominator (35/66 ≈ 53% high, 13/66 ≈ 20% missed).
* The false-positive guard caught two real cases worth a manual look: **HubSpot** (Greenhouse board name returned "HubSpot Product," not "HubSpot") and **Twilio (Segment)** (board resolved to Twilio's main board, not Segment's — exactly the ambiguity the parenthetical name flagged). **Intercom** also flagged low because its Greenhouse board name is "Fin," worth a manual check to confirm it's really Intercom's board.

### Next Session
* Hand-resolve the remaining 12 companies in `docs/unresolved.csv` (Windmill, Penpot (Kaleidos), Etsy, Retool, Grammarly, Canva, Pitch, Framer, DoorDash, GitHub, Weights & Biases, Vimeo) — `go run ./cmd/detect-ats <job-url>` speeds this up.
* Glance-check the 18 `low`-confidence rows in `docs/companies.resolved.csv`, especially HubSpot/Twilio (Segment)/Intercom.
* Move on to Stage 2a — the fetcher that pulls all resolved boards and upserts into `postings`.

---

## [2026-09-02] - Resolver: Greenhouse embed detection + fallback candidates
**Session Goal:** Extend the Stage 1 resolver with Greenhouse iframe-embed token detection and a fallback candidate pass, while hand-resolving `unresolved.csv` one company at a time.
**Status:** Completed

### The "Why" (Decision Log)
* **Resolution:** Hand-checking Noom's careers page turned up the real pattern behind several misses — the ATS board token is sometimes a suffixed or abbreviated form of the name (`noomgrowth`, not `noom`; `ww`, not `weightwatchers`), which the original 5-candidate slug generation couldn't reach. Confirmed: Noom → `greenhouse` / `noomgrowth`.
* **Resolution:** Added these as a **fallback second pass** (`GenerateFallbackCandidates`: base + `growth`/`careers`/`inc`/`hq`/`us` suffixes, plus a punctuation-aware acronym for multi-word names) rather than folding them into the primary 5-candidate cap — keeps the documented primary-pass request budget intact and only spends extra requests on companies that were already going to miss.
* **Resolution:** For companies whose careers page uses the Greenhouse iframe embed, the job link only carries `?gh_jid={job_id}` (the job ID) — the board token isn't derivable from it and has to come from the embed URL's `for=` query param instead. Documented this gotcha in RESOLVER.md so it doesn't get rediscovered the hard way next time.

### Technical Notes
* `internal/resolver/slugs.go`: added `GenerateFallbackCandidates` and `acronymOf` (skips punctuation-only fields like `&`).
* `internal/resolver/resolver.go`: `resolveOne` now tries the fallback pass only after the primary pass misses on every ATS; refactored the per-candidate probe loop into `probeCandidates` (shared by both passes, dedupes across them so a candidate is never re-probed).
* `internal/resolver/urldetect.go` (new): `DetectATS(url)` — lookup table for Greenhouse/Lever/Ashby/Recruitee/SmartRecruiters/Workday/Rippling signatures in any job URL; `ExtractGreenhouseEmbedToken(url)` — pulls the token from the embed URL's `for=` param.
* `cmd/detect-ats` (new): `go run ./cmd/detect-ats <url>` — CLI wrapper for the above, with guidance printed when a Greenhouse `gh_jid` URL is detected but the token isn't extractable from it.
* Unit tests added for all of the above (`slugs_test.go`, `urldetect_test.go`); full suite passes.
* Marked Noom's `known_ats`/`known_token` in `docs/companies.csv` and added it to RESOLVER.md's known-values table so future runs skip probing it.

### Next Session
* Continue hand-resolving `docs/unresolved.csv` one company at a time using `cmd/detect-ats`, writing confirmed tokens into `docs/companies.csv`'s `known_ats`/`known_token` columns as they're found.
* Once `unresolved.csv` is empty (or close to it), re-run `go run ./cmd/resolver` to confirm the fallback pass auto-resolves any Noom-shaped misses left in the list, then move to Stage 2a.

---

## [2026-09-02] - Manual resolution round: Windmill + Penpot cut, Personio/YC detection added
**Session Goal:** Continue hand-resolving `unresolved.csv` one company at a time.
**Status:** Partially Completed

### The "Why" (Decision Log)
* **Resolution:** Windmill turned out to be email-only applications with no ATS — marked `manual` in `companies.csv`, tier downgraded from `OK` to `Prefer Not`.
* **Resolution:** Penpot (Kaleidos) cut on fit, not resolution difficulty: Madrid-based, requires CET ±1 timezone (core hours land at 4–8am ET), Spanish + English, Linux as primary OS. Its real ATS (Personio, token `kaleidos`) was identified but deliberately **not** added as a supported ATS in the fetcher's future response-shape list — not worth the maintenance cost for a company that's a pass anyway.
* **Resolution:** Two of the first three `unresolved.csv` rows turned out to be unworkable companies rather than hard-to-resolve tokens — the Stage 0 source list was AI-generated and has some fabricated/stale details, location claims especially. **Process change: check location/logistics viability before hunting for a board token.** Cut first, resolve second. Documented in RESOLVER.md.
* **Resolution:** `companies.csv` (the master watchlist) gets updated in place as each row is resolved or cut, rather than leaving resolutions sitting only in the generated `unresolved.csv` — that file is a re-derivable report, `companies.csv` is the durable record.

### Technical Notes
* `internal/resolver/urldetect.go`: `DetectATS` now also recognizes Personio (`jobs.personio.com`/`jobs.personio.de`) and YC listings (`ycombinator.com/companies/`, `workatastartup.com` — both mapped to manual, no ATS API).
* `cmd/detect-ats`: prints Personio's token-is-the-subdomain pattern and XML feed URL, and a YC-specific note, instead of the previous generic "mark manual" message swallowing them.
* RESOLVER.md's manual-resolution section retitled "cut first, resolve second" with the location-check-first guidance and the two new lookup table rows.
* Full test suite still green after the additions.

### Next Session
* Continue through the rest of `unresolved.csv` (Etsy, Retool, Grammarly, Canva, Pitch, Framer, DoorDash, GitHub, Weights & Biases, Vimeo) — screen location/logistics fit first, then resolve ATS only for survivors.
* Once done, re-run `go run ./cmd/resolver` and move to Stage 2a.

---

## [2026-09-02] - Etsy resolved to Workday; SCORING.md staff/senior hard-filter removed
**Session Goal:** Resolve Etsy from `unresolved.csv`; revise SCORING.md's title hard filter so `staff`/`senior` postings aren't dropped before scoring.
**Status:** Completed

### The "Why" (Decision Log)
* **Resolution:** Etsy confirmed as Workday (tenant `wd5`, site `Etsy_Careers`) from a live posting URL. Marked `manual` in `companies.csv` — kept on the watchlist for hand-checking, not cut (unlike Windmill/Penpot, this is an ATS-coverage gap, not a fit problem).
* **Resolution:** Workday does expose a JSON search endpoint (`POST /wday/cxs/{tenant}/{site}/jobs`), but it's not being adapted now — the cluster and site name vary per company with no derivable pattern, and Workday rate-limits harder than the public board APIs. Documented in RESOLVER.md as a "revisit if the manual list stays large" item; Etsy joins KPMG/Deloitte/Capgemini/Lockheed Martin/Atlassian as the likely-Workday cohort that would eventually justify it.
* **Resolution:** Removed `staff` from SCORING.md's hard-filter reject list (added `VP` instead). Titles aren't calibrated across company size — "staff" at a 60-person startup often reads as "senior" at a large one — and the design engineer market is thin enough that a blanket title filter costs more real matches than it saves. `staff`/`senior` now flow through to the Fit-score model instead, with company-size-aware signals (`staff` + large co → strong −, `staff` + small co → mild −, scope matching design-system ownership → + regardless of title) so mismatches surface ranked low rather than vanishing silently pre-model.

### Technical Notes
* `docs/companies.csv`: Etsy row now `manual=true` with the Workday tenant/site note.
* `docs/RESOLVER.md`: added Etsy to the known-unsupported list and a new "Possible future adapter — Workday" note with the JSON endpoint shape and revisit condition.
* `docs/SCORING.md`: hard-filter reject line updated; added a `staff`/`senior`-not-hard-filtered explainer, three new Fit-axis rows, and a scoring-prompt rationale blurb (staff-shaped scope example: atomic Figma design system + first Dev Mode rollout across four product lines, ~3 yrs experience) to include when calibrating the model prompt in Stage 3.

### Next Session
* Continue through the rest of `unresolved.csv` (Retool, Grammarly, Canva, Pitch, Framer, DoorDash, GitHub, Weights & Biases, Vimeo) — screen location/logistics fit first, then resolve ATS only for survivors.
* When Stage 3 scoring is actually implemented, pull the `staff`-rationale blurb into the real calibration examples per SCORING.md's "write these after reading a week of real postings" guidance.

---

## [2026-09-04] - Retool resolved to Gem (manual, no public API)
**Session Goal:** Resolve Retool from `unresolved.csv`.
**Status:** Completed

### The "Why" (Decision Log)
* **Resolution:** Retool's careers page routes through Gem (`jobs.gem.com`), confirmed from a live posting URL. Unlike Personio (documented public XML feed) or Workday (documented but unauthenticated-yet-rate-limited JSON endpoint), Gem's posting IDs are opaque/GraphQL-style tokens rather than a company slug, and there's no confirmed public no-auth API to probe — didn't want to guess and fabricate an endpoint shape without verifying it actually exists. Marked `manual` rather than adding unverified adapter code.

### Technical Notes
* `internal/resolver/urldetect.go`: `DetectATS` now recognizes `jobs.gem.com` → `gem`, `manual=true`.
* `cmd/detect-ats`: prints a Gem-specific note (no confirmed public API) instead of the generic manual message.
* `docs/RESOLVER.md`: added Gem to the URL→ATS lookup table and the known-unsupported list.
* `docs/companies.csv`: Retool row now `manual=true` with the Gem note.
* Full test suite green.

### Next Session
* Continue through the rest of `unresolved.csv` (Grammarly, Canva, Pitch, Framer, DoorDash, GitHub, Weights & Biases, Vimeo).

---

## [2026-09-04] - Grammarly resolved via browser DOM inspection (Superhuman/Ashby); URL-escaping bug fixed
**Session Goal:** Resolve Grammarly from `unresolved.csv`.
**Status:** Completed

### The "Why" (Decision Log)
* **Resolution:** Grammarly's hiring is actually branded and hosted as **Superhuman** (Grammarly acquired Superhuman) at `superhuman.com/company/careers/jobs`. The given job URL carried both `gh_jid` and `ashby_jid` params — ambiguous, since either could be the real ATS or the page could just echo both query params regardless of which is live. Probing plausible tokens (`superhuman`, `grammarly`, `trysuperhuman`, etc.) directly against Greenhouse and Ashby all 404'd, and `curl`/raw HTML had no board reference — the page is a Next.js SPA that renders an empty `<div id="ashby_embed">` and injects the real iframe client-side. Had to load the page in a real browser (`claude-in-chrome`) and read the live DOM to find the actual `jobs.ashbyhq.com` iframe `src`.
* **Resolution:** The Ashby token turned out to be the literal, space-containing org name `Superhuman Platform Inc` — not a slug at all, and not anything the candidate-slug generator (or the fallback pass) could ever produce. Verified against the live API (`.../posting-api/job-board/Superhuman%20Platform%20Inc` → 200, real postings, including "Staff Product Designer, Go Core"). This is now documented in RESOLVER.md as a category of miss the automated resolver structurally can't solve — genuinely needs a human with a browser.
* **Resolution:** Discovered and fixed a latent bug while doing this by hand: `probe.go`'s three probe functions built request URLs with raw `fmt.Sprintf("%s", token)`, never URL-escaping the token. A token with a literal space (like this one) would have produced a broken request. Fixed with `net/url.PathEscape` on all three call sites — harmless for normal slug-shaped tokens, necessary for tokens like this one.

### Technical Notes
* `internal/resolver/probe.go`: all three ATS probe URL builds now `url.PathEscape` the token before interpolating.
* `docs/companies.csv`: Grammarly row now carries `known_ats=ashby`, `known_token=Superhuman Platform Inc`, and an updated `why` noting the Superhuman branding.
* `docs/RESOLVER.md`: added a "JS-rendered embeds need a real browser, not `curl`" note and the Grammarly/Superhuman example to the known-values table.
* Verified the updated CSV row parses correctly (embedded comma inside the quoted `why` field) with a throwaway test, then removed the test — it depended on the private, gitignored `companies.csv` so didn't belong in the permanent suite.
* Full test suite still green after the `PathEscape` fix.

### Next Session
* Continue through the rest of `unresolved.csv` (Canva, Pitch, Framer, DoorDash, GitHub, Weights & Biases, Vimeo).

---

## [2026-09-04] - Canva resolved to SmartRecruiters
**Session Goal:** Resolve Canva from `unresolved.csv`.
**Status:** Completed

### The "Why" (Decision Log)
* **Resolution:** Canva → SmartRecruiters, token `Canva`, confirmed live (`api.smartrecruiters.com/v1/companies/Canva/postings` → 200, 264 postings). RESOLVER.md already documented SmartRecruiters as a known-but-unbuilt "second pass" ATS; this is the first confirmed real hit for it on the watchlist — one data point toward eventually justifying a fourth Stage 2a response shape, not yet enough on its own.

### Technical Notes
* `docs/companies.csv`: Canva row now `known_ats=smartrecruiters`, `known_token=Canva`.
* `docs/RESOLVER.md`: added the confirmed Canva/SmartRecruiters hit to the "Second pass" section and the known-values table.

### Next Session
* Continue through the rest of `unresolved.csv` (Pitch, Framer, DoorDash, GitHub, Weights & Biases, Vimeo).

---

## [2026-09-04] - Pitch resolved to Workable; new ATS added to detection
**Session Goal:** Resolve Pitch from `unresolved.csv`.
**Status:** Completed

### The "Why" (Decision Log)
* **Resolution:** Pitch → Workable, token `pitch-software`. Workable's widget endpoint (`apply.workable.com/api/v1/widget/accounts/{token}`) turned out to be public and keyless, same tier as Recruitee/SmartRecruiters — confirmed via `name:"Pitch"` match and `jobs:[]`, consistent with the "no openings right now" note. An empty `jobs` array is a valid resolved-but-currently-quiet board, same handling as Greenhouse's empty-array case.
* **Resolution:** This is the second non-primary ATS confirmed live on the watchlist (after Canva/SmartRecruiters) — still not enough on its own to justify building a fourth-plus response shape into the Stage 2a fetcher, but tracked in RESOLVER.md as a running tally toward that decision.

### Technical Notes
* `internal/resolver/urldetect.go`: `DetectATS` now recognizes `apply.workable.com` → `workable`, `manual=false`.
* `cmd/detect-ats`: prints Workable's token-is-the-path-segment pattern and widget feed URL.
* `docs/companies.csv`: Pitch row now `known_ats=workable`, `known_token=pitch-software`.
* `docs/RESOLVER.md`: added Workable to the URL→ATS lookup table, the "Second pass" section (with its response shape), and the known-values table.
* Full test suite green.

### Next Session
* Continue through the rest of `unresolved.csv` (Framer, DoorDash, GitHub, Weights & Biases, Vimeo).

---

## [2026-09-04] - DoorDash, Weights & Biases resolved; GitHub/Framer marked manual; iCIMS added
**Session Goal:** Continue hand-resolving `unresolved.csv` — Framer, DoorDash, GitHub, and Weights & Biases arrived out of order in the same session.
**Status:** Completed

### The "Why" (Decision Log)
* **Resolution:** DoorDash → Greenhouse, token `doordashusa`, confirmed live (board name "DoorDash USA").
* **Resolution:** Weights & Biases → Greenhouse, token `weights_and_biases`, hosted under **CoreWeave's** careers page (CoreWeave acquired W&B) with an explicit `?board=weights_and_biases` disambiguating param in the URL. Verified the board name is literally `"Weights & Biases"` with W&B-specific job titles — not `coreweave` (CoreWeave's own board, also live and 200, but the wrong one; would've been a real false-positive if taken on faith). This is the second confirmed case of "acquisition moves hiring to the acquirer's board" (after Grammarly/Superhuman) — documented as a recognized pattern in RESOLVER.md: when a job URL carries a `board=`-style param, that's the signal to probe the parent company's ATS for the acquired brand's name as a token.
* **Resolution:** GitHub → iCIMS (tenant `careers-githubinc`). Checked for a public feed at the conventional `/xmlfeed` path; it returned 200 but was just the empty Angular app shell, not real feed data — iCIMS feed availability is opt-in per client and this tenant doesn't have it exposed (or it's at a different path). Marked `manual` rather than claim a working endpoint that isn't one. Didn't chase further URL variants — diminishing returns for a single company.
* **Resolution:** Framer → **no discoverable ATS at all.** Checked the specific job page and the `/careers` root: no request to any known ATS domain in network traffic, no `Apply` link or button anywhere in the live DOM (confirmed via JS DOM query, not just static HTML — the "apply" text found in raw HTML turned out to be unrelated Framer product-UI copy, not a job-application flow). Genuinely unclear how to apply — marked `manual` with a note flagging it needs a human look, rather than guessing.

### Technical Notes
* `internal/resolver/urldetect.go`: `DetectATS` now recognizes `icims.com` → `icims`, `manual=true`.
* `cmd/detect-ats`: prints iCIMS-specific guidance (check `/xmlfeed` by hand; it's opt-in per tenant).
* `docs/companies.csv`: DoorDash and Weights & Biases now carry `known_ats`/`known_token`; GitHub and Framer marked `manual` with investigation notes.
* `docs/RESOLVER.md`: added iCIMS to the lookup table and known-unsupported list, DoorDash/W&B to the known-values table, and a new "Acquisitions move hiring to the acquirer's board" note generalizing the Superhuman/CoreWeave pattern.
* Used `claude-in-chrome` for Framer (live DOM/network inspection, confirmed no ATS present) — closed the tab when done, no artifacts left open.
* Full test suite green.

### Next Session
* `unresolved.csv` down to Vimeo. Framer and GitHub are `manual` but not fully resolved (no working ATS found) — worth a second look later if either matters enough to chase further (Framer: ask directly / check LinkedIn for how they actually hire; GitHub: try other iCIMS feed path variants).
* Once Vimeo is done, re-run `go run ./cmd/resolver` to confirm the full picture, then move to Stage 2a.

---

## [2026-09-04] - Vimeo marked manual; all of unresolved.csv worked through
**Session Goal:** Resolve the last row in `unresolved.csv` (Vimeo) and confirm the full watchlist picture.
**Status:** Completed

### The "Why" (Decision Log)
* **Resolution:** The given Vimeo job URL (`talents.vaia.com/companies/vimeo/...`) turned out to be a third-party recruitment marketplace, not Vimeo's own ATS. Checked `vimeo.com/careers` directly (both raw HTML and live browser network/DOM) — no request to any known ATS, no open-roles widget anywhere on the page. Checked Vaia for a public API pattern too (404/301 on the obvious guesses) — nothing usable there either. Marked `manual`: genuinely no first-party source found, and the third-party marketplace listing isn't confirmed to be comprehensive or ongoing.
* **Resolution:** Added `talents.vaia.com` to the detection lookup as `manual`, same tier as YC listings — a real signal worth recognizing next time it shows up, even though it's not itself a usable data source.

### Technical Notes
* `internal/resolver/urldetect.go`: `DetectATS` now recognizes `talents.vaia.com` → `none (Vaia marketplace)`, `manual=true`.
* `cmd/detect-ats`: prints a Vaia-specific note pointing back at the company's own careers page.
* `docs/companies.csv`: Vimeo marked `manual` with the investigation note.
* `docs/RESOLVER.md`: added Vaia to the lookup table and known-unsupported list.
* Full test suite green. Kicked off a full re-run of `go run ./cmd/resolver` against the whole watchlist to get the final tally now that every `unresolved.csv` row has been hand-resolved or marked manual — result pending, will log next session.

### Next Session
* Move on to Stage 2a — the fetcher that pulls all resolved boards and upserts into `postings`.

### Postscript — full resolver re-run
Final tally against all 79 companies: **41 high, 18 low (needs review), 0 missed, 20 manual.** `docs/unresolved.csv` is now empty — every company on the watchlist is either auto-resolved, hand-resolved, or deliberately marked manual.

Hit an environment snag along the way: `go run ./cmd/resolver` failed with `An Application Control policy has blocked this file` — Windows blocking execution of the freshly-compiled binary out of `go-build*` in the temp dir (likely AV/AppLocker flagging an unsigned temp binary, unrelated to the code). Workaround: `go build -o bin/resolver.exe ./cmd/resolver` then run that binary directly — `bin/` is already gitignored. Worth remembering for future runs on this machine; `go run` may need this same workaround going forward.

### Next Session (updated)
* Move on to Stage 2a — the fetcher that pulls all resolved boards and upserts into `postings`. All ATS shapes needed for the primary three (Greenhouse/Lever/Ashby) are confirmed live across the watchlist; SmartRecruiters (Canva) and Workable (Pitch) are each a single confirmed company — revisit whether they're worth a 4th/5th response shape once Stage 2a's basic pipeline exists.
* If `go run` ever gets blocked again by the Application Control policy, use the `go build -o bin/... && bin/...` workaround instead of retrying `go run`.

---

## [2026-09-04] - All 18 low-confidence rows manually confirmed
**Session Goal:** Eyeball the 18 `low`-confidence rows from the resolver run and lock in whatever checks out.
**Status:** Completed

### The "Why" (Decision Log)
* **Resolution:** All 18 confirmed correct by hand, including the three flagged for a genuine name mismatch (HubSpot → board name "HubSpot Product"; Twilio (Segment) → board name "Twilio," i.e. the main board, not a Segment-specific one; Intercom → board name "Fin," their AI product brand). Written into `companies.csv` as `known_ats`/`known_token` so future resolver runs skip probing and don't re-flag them for review.

### Technical Notes
* `docs/companies.csv`: 18 rows updated (Hex, Ramp, Higharc, Notion, Ro, GameChanger, Buffer, Supabase, Circle, Spotify, HubSpot, Sentry, Mapbox, Twilio (Segment), Palantir, Intercom, Plaid, Oomnitza).
* `docs/RESOLVER.md`: all 18 added to the known-values table with a note explaining why they were `low` (15 structural — no Ashby/Lever name field to verify against; 3 genuine mismatches confirmed correct anyway).
* Re-ran the resolver to confirm: **59 high, 0 low, 0 missed, 20 manual** — every one of the 79 watchlist companies is now fully resolved (auto, hand-confirmed, or deliberately manual). Stage 1 is complete.

### Next Session
* Move on to Stage 2a — the fetcher. `docs/companies.resolved.csv` is the input: 59 companies across Greenhouse/Lever/Ashby ready to pull, plus Canva (SmartRecruiters) and Pitch (Workable) as a judgment call on whether to build a 4th/5th response shape now or defer.

---

## [2026-09-04] - Stage 2a built: fetcher + Postgres schema
**Session Goal:** Build the Stage 2a fetcher — pull all resolved boards, normalize the three ATS shapes, upsert into `postings`.
**Status:** Completed (code); blocked on live end-to-end test pending Postgres credentials

### The "Why" (Decision Log)
* **Resolution:** Picked **Neon** for Postgres (PROJECT.md's open decision, framed as "both free, both fine — pick one and move on"). Serverless/scale-to-zero fits a fetcher that only runs once daily, and it supports `pg_trgm` out of the box for Stage 5. Recorded as resolved in PROJECT.md. I can't create the account myself (account creation is off-limits for me to do on someone's behalf) — user needs to sign up and hand me a `DATABASE_URL`, which is why the code is built and tested at every layer that doesn't require a live DB, but the actual `sync-companies`/`fetcher` binaries haven't been run against a real database yet.
* **Resolution:** Realized mid-build that `docs/companies.csv` (the durable source of truth) only had `known_ats`/`known_token` filled in for hand-verified companies — the ~41 auto-resolved "high confidence" ones only existed in the ephemeral, gitignored `companies.resolved.csv`. Rather than have `sync-companies` read two files and merge them, added `resolver.BackfillKnownValues` + `WriteCompanies` so the resolver itself writes high-confidence results straight back into `companies.csv`. Ran it once: backfilled 25 companies. `companies.csv` is now fully self-sufficient (59 real `known_ats`/`known_token` rows, 20 `manual`) — matches the established practice from earlier sessions of treating `companies.csv`, not the resolved/unresolved CSVs, as the durable record.
* **Resolution:** Designed the posting-close logic as "close everything open for this company, then let the upsert reopen (clear `closed_at` on) whatever's still present" rather than tracking a seen-ID set and doing a `NOT IN` filter. Simpler, avoids array-parameter binding through `database/sql`, and naturally satisfies the "running it twice adds zero rows the second time" requirement — a same-session rerun closes and immediately reopens the identical set, net no change.
* **Resolution:** Kept `fetcher`'s unsupported-ATS handling (SmartRecruiters for Canva, Workable for Pitch, `manual` for the 20 cut/deferred companies) as a skip-with-log rather than an error, and left those companies `active=true` in the DB regardless — `active` reflects "should we be watching this company," not "can our code currently fetch it," so adding a 4th/5th adapter later doesn't require touching any existing data.

### Technical Notes
* Added `github.com/jackc/pgx/v5` (pure Go Postgres driver, no cgo) via `go get` — confirmed working in this environment despite the earlier Application Control policy issue (that only blocks *executing* freshly-built binaries, not `go get`/module downloads).
* `db/schema.sql`: `companies`, `postings`, `scores`, `connections` tables per PROJECT.md's schema, plus `pg_trgm` extension and a couple of indexes (`postings.company_id`, a partial index on open `postings.closed_at`, and a trigram GIN index on `connections.company_norm` for Stage 5).
* `internal/db`: `Connect` (pgx/stdlib + ping), `UpsertCompany`/`ActiveCompanies` (upsert keyed on `name_norm`), `UpsertPosting`/`CloseAllOpenPostings` (the close-then-reopen pattern above).
* `internal/fetcher`: `FetchGreenhouse`/`FetchLever`/`FetchAshby`, each normalizing into a common `Posting` struct. Field names verified against **live** API responses before coding (Greenhouse: `id`/`title`/`absolute_url`/`location.name`/`content`; Lever: `id`/`text`/`hostedUrl`/`categories.location`/`description`; Ashby: `id`/`title`/`location`/`jobUrl`/`descriptionHtml`) — same "probe, don't guess" discipline as the resolver. Caught and handled a real quirk: Greenhouse's `content` field is HTML-entity-double-escaped (`&lt;div&gt;` as literal text, not `<div>`), so it's `html.UnescapeString`'d on the way in.
* Exported `resolver.NormalizeCompanyName` (was `normalizeCompanyName`) for reuse by `sync-companies`'s `name_norm` computation, instead of writing a second normalizer.
* `cmd/sync-companies`: reads `companies.csv` via the existing `resolver.ReadCompanies`, upserts into the `companies` table.
* `cmd/fetcher`: loads active companies, dispatches to the right normalizer, closes-then-reopens postings per company, prints a summary. Politeness sleep between companies matches the resolver's rate.
* Unit tests: `internal/fetcher` tests each normalizer against `httptest` fixtures built from the real confirmed field shapes (including the Greenhouse unescape case, a 404, and malformed JSON), plus a `BackfillKnownValues` test. Also did a live smoke test (temporary `cmd/smoketest`, deleted after) against real Anthropic/Ro/Linear boards to confirm the normalizers work end-to-end before wiring up the DB — all three came back clean with real data.
* Added `.env.example` documenting the expected `DATABASE_URL` var.

### Next Session
* Revisit whether Canva (SmartRecruiters) and Pitch (Workable) are worth a 4th/5th `internal/fetcher` adapter now that the pattern is established — each is a single confirmed company so far.
* Stage 2a's "done when" criterion is satisfied — decide what's next: Stage 2b (capture, deferred per PROJECT.md guardrails until 2a/3/4 have run a few weeks) or straight to Stage 3 (filter + score).

---

## [2026-09-04] - Stage 2a live end-to-end run, verified against real Postgres
**Session Goal:** Get a Neon Postgres instance stood up and run the Stage 2a fetcher against it for real.
**Status:** Completed

### The "Why" (Decision Log)
* **Resolution:** Declined Neon Auth during project setup — this is a single-user personal tool with no accounts/sessions concept anywhere in the design (the one place PROJECT.md mentions auth, the deferred Stage 2b capture app, calls for a simple long-lived cookie token, not a real auth provider). Would have only added unused schema.
* **Resolution:** Default Postgres version (18) is fine — nothing in `db/schema.sql` needs anything beyond long-standing standard features (`SERIAL`, `TIMESTAMPTZ`, `pg_trgm`, GIN indexes, `ON CONFLICT`).
* **Resolution:** User provided the Neon pooled connection string (`-pooler` host) directly in chat; written to a local `.env` (confirmed gitignored via `git check-ignore` before proceeding). This is local app configuration for infrastructure the user owns, not a third-party credential entry.
* **Resolution:** Verified the "running it twice adds zero rows the second time" claim empirically, not just by code inspection. First run: 6,346 postings across 57 fetchable companies (Canva/SmartRecruiters and Pitch/Workable skipped as expected, 0 errors). Second run minutes later: 6,347 — a +1 delta, not the exact match the phrase implies at first glance. Confirmed this is real-world drift, not a bug: `postings` has a `UNIQUE (ats, ats_job_id)` constraint, so a duplicate row is structurally impossible — checked directly (0 duplicate `(ats, ats_job_id)` pairs) and found the one new row was a genuinely new Stripe posting (`first_seen` timestamp from the second run only, "Program Manager, Competitive Programs"). The idempotency guarantee holds by construction; the two runs just happened to straddle a real job going live.

### Technical Notes
* Ran `sync-companies` (79 companies loaded) then `fetcher` (57 fetched, 6,346→6,347 postings across the two runs, 2 skipped, 0 errored) against the live Neon database.
* Verification used a temporary throwaway command (same pattern as the earlier normalizer smoke test) — queried total/open/closed posting counts, checked for duplicate dedupe keys, and inspected the most recently-`first_seen` rows. Deleted after confirming.
* PROJECT.md updated: Postgres host resolved to Neon (was already noted as the plan, now confirmed live); architecture table's Postgres row simplified from "Neon or Supabase" to "Neon."

### Next Session
* Stage 2a is done and verified live. Options per PROJECT.md's stage list: Stage 3 (filter + score, needs a Claude API key and the SCORING.md rubric already written) or scheduling Stage 2a to run daily (Stage 4) first. Stage 2b (capture) stays deferred per the project's own guardrail — don't build it until 2a/3/4 have run for a few weeks.
* Revisit whether Canva (SmartRecruiters) and Pitch (Workable) are worth a 4th/5th `internal/fetcher` adapter — each is a single confirmed company so far, both currently just skipped with a log line.

---

## [2026-09-04] - Stage 3 built: hard filters + Claude Haiku scorer
**Session Goal:** Build Stage 3 — SQL-tier hard filters, then a Claude Haiku call per new posting, per docs/SCORING.md.
**Status:** Code complete; blocked on live end-to-end test pending an Anthropic API key

### The "Why" (Decision Log)
* **Resolution:** Implemented the hard filters in Go (`internal/scorer/filters.go`) rather than literal SQL — PROJECT.md's "SQL pre-filter" phrasing is about doing cheap deterministic filtering before spending on an LLM call, not a syntax mandate, and Go's string handling is far more workable than nested SQL regex for this. Title-match/reject and the Remote(US)/NYC/Philly/NJ location bucketing are all heuristic on free-text strings; documented in code comments as approximate and expected to need tuning after reading real output, same spirit as the scoring rubric itself.
* **Resolution:** Found and fixed a real schema gap while wiring this up: the `companies` table had no `tier` column, but SCORING.md's Want axis explicitly needs "Company-tier from the watchlist (Yes/OK/Prefer Not)" as a modifier. Added it via a migration (`db/migrations/0002_...sql`) rather than misusing the existing `notes` column, and wired `sync-companies` to populate it from `companies.csv`'s `Tier` field (which was already being parsed by `resolver.ReadCompanies` but never used downstream).
* **Resolution:** Redesigned the `scores` table to match SCORING.md's actual three-axis output contract (`want`/`fit`/`access`/`composite`/`lead_with`/`reasoning`/`concerns`) instead of the older, simpler `fit_score`/`stack_overlap` shape sketched in PROJECT.md's schema skeleton — SCORING.md is the more detailed, later-written spec and is authoritative over the earlier placeholder. Applied as a migration (drop+recreate; table was empty, no data to preserve).
* **Resolution:** Used Claude's structured outputs feature (`output_config.format` with a JSON schema) rather than a plain "respond with JSON" prompt — guarantees a parseable response instead of hoping the model doesn't wrap it in prose or markdown fences. Confirmed Haiku 4.5 supports structured outputs before committing to this approach.
* **Resolution:** Deliberately did **not** write the 2-3 hand-scored calibration examples SCORING.md calls for — its own text says "Write these after reading a week of real postings — not before." Left a clear code comment marking where they go once that week of output exists. Did include the one piece of prompt content SCORING.md explicitly wrote out for inclusion now: the staff-title scope-not-title-or-years rationale.
* **Resolution:** Verified the exact Go SDK types for structured outputs (`OutputConfigParam`, `JSONOutputFormatParam`) by grepping the downloaded module source directly rather than guessing or trusting a possibly-stale cached doc — this is a fast-moving beta-ish surface and the installed package is the ground truth for what will actually compile.

### Technical Notes
* `internal/scorer/filters.go`: `PassesHardFilters(title, location)` — title must-match/reject lists, the `contract` (unless remote) carve-out, a non-ASCII heuristic standing in for "non-English location string", and Remote(US)/NYC/Philly/NJ location bucketing. Unit tested (`filters_test.go`, 16 cases) — confirmed `staff`/`senior` titles pass through un-rejected per the SCORING.md change from a couple sessions back.
* `internal/scorer/scorer.go`: `ScorePosting` — one Haiku 4.5 call per posting via `github.com/anthropics/anthropic-sdk-go`, structured-output JSON schema matching the `Score` struct exactly, system prompt encoding the candidate profile (from SCORING.md's portfolio table + the staff-title rationale's implied background) and the three-axis rubric with weights. Handles `stop_reason: refusal` and `max_tokens` explicitly rather than blindly indexing into `response.Content`.
* `internal/db/scores.go`: `UnscoredOpenPostings` (open postings with no `scores` row yet, joined to company name+tier) and `InsertScore`.
* `db/schema.sql` and a new `db/migrations/` directory (first migration beyond the initial schema) for the tier column + scores redesign.
* `cmd/scorer`: loads unscored open postings, applies hard filters, scores survivors, prints `[composite] company - title` lines to stdout per PROJECT.md's "print to stdout first" plan — no digest/delivery yet.
* Applied the migration and re-ran `sync-companies` against the live Neon database (throwaway migration-runner command, deleted after use — same pattern as prior sessions' one-off DB checks).
* Full test suite green; `go vet` clean.

### Next Session
* After a week of real scoring output: write the 2-3 hand-scored calibration examples into `scorer.go`'s system prompt (one obvious yes, one obvious no, one genuinely marginal), per SCORING.md's explicit sequencing.
* Decide next stage per PROJECT.md: Stage 4 (schedule fetcher + scorer to run daily via Task Scheduler, add the nightly email digest) is the natural next step now that both pipeline stages work end-to-end.

---

## [2026-09-04] - Stage 3 live run: 58 of 6,342 postings scored
**Session Goal:** Get an Anthropic API key wired up and run the scorer against the full live posting set.
**Status:** Completed

### The "Why" (Decision Log)
* **Resolution:** User provided a real `ANTHROPIC_API_KEY` (console.anthropic.com, separate from the Claude Code CLI's own login) directly in chat; written to `.env` alongside `DATABASE_URL`, same handling as the database credential earlier — local app config for infrastructure the user owns, not a third-party credential entry.
* **Resolution:** Added a `-limit` flag to `cmd/scorer` (caps postings actually sent to the API, not raw rows scanned) and ran a 5-posting smoke test before committing to the full ~6,300-posting run — confirmed the structured-outputs request/response shape actually works end-to-end before spending on the full batch.
* **Resolution:** Full run: **58 of 6,342 unscored open postings survived the hard filters and got scored; 0 errors.** The ~0.9% survival rate looked low at first glance but tracks with reality — design-engineer/design-systems roles are genuinely rare among all open roles across 57 companies, and PROJECT.md's own framing is "a needle in a haystack, within a day of posting," not a high-volume filter.
* **Resolution:** The rubric is calibrated well even without the hand-scored examples SCORING.md says to add later — scores spread from 20 to 93, not clustered at ~75, with specific, harsh reasoning (years-experience gaps, crypto-tier flags, "title says Staff but scope reads Product Designer," missing design-systems language). The two cleanest hits (Supabase Product Designer, 93; Vercel Senior Product Designer Growth, 81) both landed on "Yes"-tier companies the candidate actually uses, with no concerns at all — exactly the shape SCORING.md's rubric was designed to surface.

### Technical Notes
* `cmd/scorer/main.go`: added `-limit N` (max postings *scored*, i.e. hard-filter survivors that reach the API — not the first N rows, which could all be filtered and never test the API path).
* No code changes beyond the flag — the scorer package worked as designed on the first live run.

### Next Session
* Read the 58 scored postings for real (SCORING.md: "read a week of output before automating delivery") and start collecting the 2-3 hand-scored calibration examples once there's a week of real output to draw from.
* Move to Stage 4: schedule `fetcher` then `scorer` to run daily via Windows Task Scheduler, and build the nightly email digest (empty day sends nothing).
