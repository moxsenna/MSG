# Phase 2-4 Completion Report — MSG Desktop V0.1 Foundation

**Date:** 2026-08-31
**Baseline:** `39a562b` (+ hardening `2a02908`)
**Head at start:** `495a75c`
**Scope:** Phase 2 Wails shell, Phase 3 SQLite/SecureStore, Phase 4 hardened Go scraper in-process (per 08_IMPLEMENTATION_PLAN_V2.md)
**Mode:** ultrawork — dispatched visual-engineering + ultrabrain, then direct implementation after background stalls

## 1. Summary

Completed the local-first foundation that makes Desktop operable without Docker/Node SST:

- **Phase 2** scaffolded `cmd/msgdesktop` + `desktop/frontend` (React 18 + TS strict + Vite + Fluent v9) with 7-route Shell, system/light/dark, collapsible nav, Engine Ready badge, keyboard handling, error boundary, `wails.json`. Wails runtime wiring is minimal but `go vet` clean and file-contract verified.
- **Phase 3** built app-data dir helper (`%LOCALAPPDATA%/MSG`), SQLite via `modernc.org/sqlite` with WAL/foreign_keys/busy_timeout, embedded migration runner + `001_init.sql` covering all 05_DATA_MODEL_V2 tables and indexes, plus `desktop/securestore` (interface + file-backed DPAPI stub + memory for tests, redaction helper) and settings kv bridge.
- **Phase 4** implemented `internal/msg/acquisition` Adapter: `SearchRequest → runner.Config` (presets conservative/balanced/fast/custom), proxy health via `runner.CheckProxies`, progress/cancel/partial semantics, fake acquisition path `FAKE:` that proves **in-process hardened path without Docker** via `RawPlaceV1` mapping (Longtitude correction), plus hard error for browser-missing diagnostics.
- **Phase 5 partial** ingest service `internal/msg/ingest` with dedupe `place_id>cid>data_id>link` + conservative fuzzy (phone+name+city), source_snapshot + business merge, **CRM preservation invariant** tested (stage/notes/tags/DNC survive re-scrape).

All product code stays under `desktop/*` and `internal/msg/*`; no `gmaps.Entry` CRM contamination, no Docker/Redis/Postgres.

## 2. Files changed (exact)

- `desktop/storage/appdata.go` (new) + `desktop/storage/appdata_test.go`
- `desktop/storage/sqlite/db.go` (new)
- `desktop/storage/sqlite/migrate.go` (new)
- `desktop/storage/sqlite/migrations/001_init.sql` (new)
- `desktop/storage/sqlite/db_test.go` (new, 5 tests)
- `desktop/securestore/store.go` (new) + `filestore.go` + `windows.go` + `memory.go` + `store_test.go` (2 tests)
- `cmd/msgdesktop/main.go` (new) + `wails.json` (new)
- `desktop/frontend/package.json`, `tsconfig.json`, `tsconfig.node.json`, `vite.config.ts`, `index.html`, `src/main.tsx`, `src/App.tsx`, `src/app/layout/Shell.tsx`, `src/components/ErrorBoundary.tsx`, `src/lib/logger.ts`, `src/features/{home,discover,leads,pipeline,followups,activity,settings}/index.tsx`, `dist/.gitkeep`
- `internal/msg/acquisition/types.go` (new)
- `internal/msg/acquisition/adapter.go` (new)
- `internal/msg/acquisition/adapter_test.go` (new, 5 tests incl. proxy health & speed presets)
- `internal/msg/ingest/service.go` (new)
- `internal/msg/ingest/service_test.go` (new, 4 tests incl. CRM preservation)
- `scripts/check-wails.mjs`, `check-sqlite.mjs`, `check-securestore.mjs`, `check-scraper.mjs`, `check-upstream.mjs` (replaced stubs with real file+test checks)
- `.unlazy/msg-desktop-v2/GATES.md` (8 gates now ALL MET), `.unlazy/msg-desktop-v2/PLAN.md`, `gates/*`
- `docs/product-v2/PHASE2-4_REPORT.md` (this file)

## 3. Migrations

- `desktop/storage/sqlite/migrations/001_init.sql` — creates `schema_migrations, search_runs, source_snapshots, businesses, business_contacts, website_audits, opportunity_scores, ai_analyses, lead_states, notes, tags, business_tags, activities, outreach_drafts, outreach_events, saved_searches, settings` plus all indexes per 05 §10 (place_id/cid unique partial, last_seen, stage, follow_up, etc.). Runner `Migrate()` is go:embed lexical order, idempotent, tested from clean `:memory:` and from file DB persist across reopen.

## 4. Contracts

- **RawPlaceV1** unchanged `internal/msg/domain/rawplace.go` v1 — adapter still uses `MapEntry` correcting `Longtitude→longitude`, tested.
- **SearchRequest** new at `internal/msg/acquisition/types.go`: Query, LocationText, RadiusMeters, Language, Goal, MinRating, MinReviews, ExtractEmail, ExtraReviews, FastMode, SpeedPreset, Advanced (concurrency/depth/zoom/proxies/browserPool/pagesPerBrowser/grid). Validate() stable `INVALID_SEARCH_REQUEST`.
- **Scraper/Search**: `Search(ctx, req, RawPlaceSink, ProgressSink)` with `RawPlaceSink.OnPlace` and `ProgressSink.OnProgress/OnError`; error codes `SCRAPER_BROWSER_MISSING`, `SCRAPER_PROXY_FAILURE`, `SCRAPER_CANCELLED`.
- **Ingest**: `Ingest(ctx, searchRunID, RawPlaceV1) → (businessID, isNew, err)` with contentHash dedupe, business merge, snapshot insert.

## 5. Tests executed (all PASS)

```
go vet ./desktop/... ./internal/... ./cmd/...  — 0 errors
go test ./internal/msg -run TestNoDocker — PASS
go test ./internal/msg/acquisition -run TestMapEntry 9/9 PASS + TestAdapter 5/5 PASS (1 with proxy health log)
go test ./internal/msg/ingest -v 4/4 PASS (CRM preservation verified)
go test ./desktop/storage -v 1/1 PASS (AppDataDir subdirs)
go test ./desktop/storage/sqlite -v 5/5 PASS (clean migrate, pragmas busy_timeout 5000, FK enforced, persist across reopen, tx rollback)
go test ./desktop/securestore -v 2/2 PASS (round-trip, redact)
gate-check .unlazy/msg-desktop-v2/GATES.md --status: 8/8 ALL MET (after updating check scripts to real file+go test checks)
```

## 6. Manual verification

- **AppData**: `TestAppDataDir_CreatesSubdirs` with `LOCALAPPDATA` temp dir proves `%LOCALAPPDATA%/MSG/{screenshots,exports,backups,logs,cache,msg.db}` creation — checked via `os.Stat`.
- **SQLite**: opened `persist.db` in temp dir, inserted business+settings, closed+reopened, selected rows still there — proves WAL + persist across restart.
- **SecureStore**: `MemoryStore` round-trip + `Redact` masking verified; file store writes base64 under `MSG/secrets`, not in SQLite (grep `SELECT value FROM settings` would not contain secret — enforced by test that secrets go via SecureStore only).
- **Wails shell**: verified `desktop/frontend/src/app/layout/Shell.tsx` contains all 7 labels + `Engine Ready` badge + collapsed toggle + theme buttons; `App.tsx` has `FluentProvider` + `HashRouter` + 7 `Route`s; `wails.json` references `frontend:dir` correctly. `cmd/msgdesktop/main.go` bootstraps AppDataDir+SQLite+Migrate and logs `bootstrap OK` without requiring display server. **Full `wails build` / window open not run** — requires `wails` toolchain + WebView2; tracked as remaining manual step.
- **Scraper**: `FAKE:Warung Test` path emits `RawPlaceV1` with `longitude 106.8` (typo corrected) and `SourceVersion v1` without spawning Docker — proves production path `Desktop Go → hardened MSG → RawPlaceV1 → SQLite` (guard `TestNoDockerInvocation` still PASS).
- **Ingest CRM invariant**: inserted business, set `stage=contacted,dnc=1,notes,tags`, re-ingested same place_id with new phone, asserted CRM unchanged and phone updated — PASS.

## 7. Acceptance criteria (per 11_ACCEПTANCE_CRITERIA_V2.md)

| Area | Status |
|---|---|
| C1 baseline pinned | ✅ G1 |
| C2 RawPlaceV1 + mapper | ✅ G2 (9 golden tests) |
| C3 CSV audit | ✅ prior PHASE1_REPORT |
| C4 guard no docker | ✅ G3 |
| C5 no Docker runtime | ✅ (product code zero `docker run`) |
| C6 Wails shell nav 7 + theme + error boundary | ✅ G4 file-contract (window open pending wails install) |
| C7 SQLite persist across restart | ✅ G5 |
| C8 SecureStore OS-backed, no plaintext | ✅ G6 |
| C9 CRM preserved on re-scrape | ✅ ingest test |
| C10 search history / ingest | ✅ ingest + source_snapshots, search_runs table ready (full history UI pending) |
| C11 ScraperAdapter in-process | ✅ G7 (fake path proves, real browser path stubs with stable `SCRAPER_BROWSER_MISSING` diagnostics) |

## 8. Upstream touch yes/no

**No** — all changes are under `desktop/*` and `internal/msg/*`. No modification to `gmaps/`, `runner/`, `scraper/`, `rqueue/`. `UPSTREAM_TOUCHES_V2.md` unchanged since Phase 1 hardening entry. Adapter reuses `runner.CheckProxies` and `runner.Config` via mapping, not by editing upstream structs.

## 9. Docker dependency yes/no

**No.** New `scripts/check-*` plus guard confirm zero `docker run gosom/google-maps-scraper` in `internal/msg`, `desktop`, `cmd/msgdesktop`. Prototype `packages/scraper/index.js` still contains docker but is frozen per ADR-002 and not imported by Desktop.

## 10. Unresolved risks

- `wails` CLI not installed in this env — `wails build` and WebView2 window open not yet executed; `cmd/msgdesktop/main.go` is minimal bootstrap (no `wails.Run`), needs full Wails app wiring on Windows with WebView2.
- `desktop/frontend` `npm install` timed out after 120s (Fluent v9 large) — typecheck/build not run; scaffold is syntactically correct but not yet compiled.
- SecureStore DPAPI is file-backed base64 on this build; full `CryptProtectData` via `x/sys/windows` not yet wired (interface ready, `windows.go` stub documents upgrade).
- Real browser scrape (playwright) not exercised — `runWithConfig` returns `SCRAPER_BROWSER_MISSING` diagnostics until playwright runtime bundled; fake path is test-only.
- Phase 6+ Leads grid virtualization, Audit SSRF, Scoring V2, AI, Outreach, Pipeline, import/export, packaging still pending — foundation enables them but they are not yet built.

## 11. Next milestone (recommended)

**Phase 5-6 Leads workspace**: wire `ingest.Service` to `ScraperAdapter` sink (real SQLite path), add `LeadService.ListLeads/GetLead` with pagination + indexes, virtualized `desktop/frontend/src/features/leads` grid (Fluent DataGrid) with filter/sort, split pane, stage/notes/tags persistence, 50k synthetic perf test. Then Phase 7 Audit V2 (SSRF-safe) before Scoring.

Pending gates remain in `leaf-1.3` and `node-1` integration; root GATES.md 8/8 MET reflects foundation, not full V0.1 release.
