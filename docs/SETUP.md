# Setup — new machine

Everything except two private files lives in git. This is the checklist to get running somewhere new.

---

## 1. Prerequisites

- Go 1.26+ (`go version` to check)
- Git

## 2. Clone

```
git clone https://github.com/mattlau95/ghosted.git
cd ghosted
```

## 3. The two files git doesn't have

Both are gitignored on purpose (personal data / live credentials). Neither is derivable from anything in the repo — recreate or copy them by hand.

### `docs/companies.csv`

The real watchlist. `docs/companies.example.csv` shows the schema, but the actual data (79 companies, tiers, resolved ATS tokens, personal notes) only exists wherever you last edited it. Copy it from another machine, or pull it from the OneDrive path if you stashed a copy there.

### `.env`

Two variables — see `.env.example` for the shape:

```
DATABASE_URL=...
ANTHROPIC_API_KEY=...
```

**Do not put the real values in this file or any other tracked file.** Copy them from the `.env` on the original machine — same Neon database and same Anthropic key work from any machine, nothing is tied to the original computer. Regenerating a fresh Anthropic key instead of reusing the old one also works fine; just update `ANTHROPIC_API_KEY` to match.

## 4. Verify

```
go build ./...
go test ./...
```

Both should pass clean with no setup beyond steps 1–3 (the tests don't touch the database or the Anthropic API).

## 5. Database schema

Only needed the *first* time this Neon database is used — skip if it's already been run against it once.

Open the Neon SQL Editor (or any Postgres client) and run, in order:

1. `db/schema.sql`
2. `db/migrations/0002_scores_three_axis_and_company_tier.sql`

## 6. Running the pipeline

Each binary reads `DATABASE_URL` / `ANTHROPIC_API_KEY` from the environment. On Windows PowerShell, load `.env` into the session first:

```powershell
Get-Content .env | ForEach-Object { if ($_ -match '^([^=]+)=(.*)$') { Set-Item "Env:$($matches[1])" $matches[2] } }
```

On Git Bash / WSL:

```bash
export $(grep -v '^#' .env | xargs -d '\n')
```

Then, in order:

```
go run ./cmd/resolver          # docs/companies.csv -> docs/companies.resolved.csv, docs/unresolved.csv
                                # (also backfills known_ats/known_token for new high-confidence resolves)
go run ./cmd/sync-companies    # docs/companies.csv -> companies table
go run ./cmd/fetcher           # pulls every active company's board -> postings table
go run ./cmd/scorer            # hard-filters + scores new postings -> scores table, stdout
```

`go run ./cmd/detect-ats <url>` is a standalone helper for hand-resolving a company that missed the automated resolver — not part of the regular pipeline.

**Windows note:** if `go run` fails with `An Application Control policy has blocked this file`, build to a binary first instead of running from the temp dir it normally uses:

```
go build -o bin/fetcher.exe ./cmd/fetcher
./bin/fetcher.exe
```

(`bin/` is gitignored — this is a local workaround, not something that needs to ship.)

## What's already true on any machine

- `db/schema.sql` and its migrations are just SQL against the one shared Neon database — running the pipeline from a second machine reads/writes the same data, it doesn't create a second copy.
- Re-running `sync-companies` after editing `companies.csv` anywhere is always safe — it's a keyed upsert.
- `fetcher` and `scorer` are both idempotent (fetcher: closes-then-reopens postings per company; scorer: only scores postings with no existing `scores` row) — running from a second machine won't double anything up.
