# Baseline Reconciliation — MSG V2

**Baseline commit:** `39a562bdede752225a8eeb171421b087580745e1` (hybrid: Go scraper + hardening `2a02908` + localbiz-ai + alie-core)
**Head at start of Phase 0/1:** `39a562b` (+ hardening history)
**Date:** 2026-08-31
**Agent:** Sisyphus

## Commits newer than V2 baseline
- `2a02908` — hardening (concurrent CentralWriter, parser WARN, proxy health, backoff) — included in baseline per pack
- `39a562b` — merge localbiz-ai monorepo into MSG (hybrid)
- No commits beyond those at Phase 0 start

## Go checks (pre-existing)
- `go vet ./gmaps ./scraper ./runner ./deduper ./grid ./exiter` — attempted, Windows go tool paging file error under constrained env; on CI expected clean per .golangci.yaml (30 linters, timeout 3m). Recorded as env limitation, not code failure.
- `go test -short ./gmaps ./grid ./runner ./scraper` — partial success: `TestCentralWriter_*` 12/12 PASS, `Test_EntryFromJSON*` PASS, `TestCreateSeedJobs` PASS. Full suite not run due to paging; no pre-existing failing tests observed in those packages.
- `go test ./internal/msg/...` — new product code not yet in baseline; no pre-existing failures to carry.

## Component ownership (label)
- **Production:** `gmaps/`, `runner/`, `scraper/`, `rqueue/`, `api/`, `admin/`, `deduper/`, `exiter/`, `grid/` — hardened Go acquisition
- **Prototype (non-production):** `packages/scraper/index.js` calls `docker run gosom/google-maps-scraper` (confirmed string search), `packages/audit`, `packages/scoring`, `packages/ai`
- **Frozen V0.1:** `alie-core/` (Nest/Turbo, Redis/BullMQ/PG/MinIO) — docs only
- **New product:** `internal/msg/*`, `desktop/*` (not yet, Phase 2), `docs/product-v2/*`

## Docker dependency check
- No production Desktop code exists yet that requires Docker. `packages/scraper/index.js` prototype does, but product path `internal/msg` does not. Guard test `internal/msg/guard_test.go` enforces.

## Pre-existing failures recorded
- None observed in Go packages tested. Env paging error noted as infra, not code. Lint not run locally due to same env.
