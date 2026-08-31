# Phase 0/1 Completion Report — MSG Desktop V0.1

**Date:** 2026-08-31
**Baseline:** `39a562b` (+ hardening `2a02908`)
**Head:** `1ce3edf`

## Baseline verification
- HEAD confirmed via `git log --oneline`: 39a562b merge localbiz-ai, 2a02908 hardening, both included in V2 baseline per README.
- Go checks: `go test -run TestMapEntry ./internal/msg/acquisition -v` 9/9 PASS, `go test -run TestNoDockerInvocation` initially FAIL (guard caught itself) → fixed to PASS, `go test -run Test_EntryFromJSON ./gmaps` PASS, `go test -run TestCreateSeedJobs` PASS, `TestCentralWriter` 12/12 PASS earlier. Full suite paging-limited on Windows but no pre-existing failing tests observed.
- No production Desktop code requires Docker (guard proves).

## Files changed (exact)
- `internal/msg/domain/rawplace.go` (new)
- `internal/msg/acquisition/mapper.go` (new)
- `internal/msg/acquisition/mapper_test.go` (new, 9 scenarios)
- `internal/msg/acquisition/testdata/golden_*.json` (4 fixtures)
- `internal/msg/guard_test.go` (new)
- `UPSTREAM_TOUCHES_V2.md` (new)
- `docs/product-v2/BASELINE.md`, `FIELD_MAPPING.md`, `CSV_CONTRACT_AUDIT.md` (new)

## RawPlaceV1 contract
- Package `internal/msg/domain`, const `RawPlaceVersion="v1"`, struct with 26 fields + CompleteAddress + LinkSource, JSON snake_case, correct `longitude` spelling, CapturedAt UTC. No import of gmaps inside domain. See `docs/product-v2/FIELD_MAPPING.md`.

## Field mapping matrix
- See `docs/product-v2/FIELD_MAPPING.md` — 26 rows, all direct copies except Longtitude→Longitude and slice copies with nil→[] guard.

## Tests and output
- `TestMapEntry_NormalComplete` — PASS (verifies link/CID/place_id/categories/website/emails/rating/coordinates)
- `TestMapEntry_MissingOptional` — PASS (nil→[] , empty website)
- `TestMapEntry_MultipleEmails` — PASS (copy not alias)
- `TestMapEntry_WebsiteNoWebsite` — PASS
- `TestMapEntry_Coordinates` — PASS
- `TestMapEntry_ClosedBusiness` — PASS
- `TestMapEntry_LegacyLongitude` — PASS (JSON emits longitude not longtitude)
- `TestMapEntry_Nil` — PASS
- `TestMapEntry_LinkAndIDs` — PASS
- `TestNoDockerInvocationInProductCode` — initial FAIL `forbidden docker invocation found in guard_test.go` → fixed to PASS
- Evidence artifacts: `go test ./internal/msg/acquisition -v` output 9 PASS, `go test ./internal/msg -v` guard PASS.

## Discovered CSV mismatch details
- Go `review_rating` vs Node `rating`, `review_count` vs `reviews`, `link` vs `url` → Node gets undefined, score wrong, maps_url empty. Full matrix in `docs/product-v2/CSV_CONTRACT_AUDIT.md`.

## Upstream touches yes/no
- No new upstream touches in Phase 1 beyond those already logged for hardening. Mapper is product-side only.

## Docker dependency introduced yes/no
- **No.** Product code has zero docker invocations (guard PASS). Prototype `packages/scraper/index.js` still has docker but frozen, not used by Desktop.

## Unresolved risks
- Windows paging file error prevents full `go test ./...` and `golangci-lint` locally; CI must verify.
- Golden JSON fixtures not yet auto-generated from EntryFromJSON real samples; current goldens are hand-crafted minimal — recommend generating from `testdata/entry_*.json` via mapper for stricter parity.
- CSV adapter for legacy pipeline not built (intentionally deferred to backlog).

## Recommended Phase 2 start
- Scaffold `cmd/msgdesktop` + Wails v2 + React Fluent v9 per 08_IMPLEMENTATION_PLAN Phase 2, with navigation Home/Discover/Leads/Pipeline/Follow-ups/Activity/Settings, light/dark, titlebar persistence, error boundary, logging bootstrap. Provide dev build that opens without localhost server.
