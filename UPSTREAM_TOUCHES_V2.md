# Upstream Touches V2

Log every product-required modification in upstream-derived Go areas. Preserve module path `github.com/gosom/google-maps-scraper` initially.

## 2026-08-31 — Hardening (included in baseline 2a02908)
- `scraper/centralwriter.go` — single slot `current *trackedJob` → `jobs map[string]*trackedJob` for concurrent River jobs. Justification: SaaS multi-worker overwrite bug. No CRM concerns.
- `gmaps/entry.go` / `gmaps/multiple.go` — WARN logs for missing Title/Cid/coordinates via `log.Printf`. Justification: observability for Google layout changes.
- `runner/proxy_health.go` (new) + `runner/filerunner/filerunner.go` + `scraper/scraper.go` — HEAD probe for http/https proxies, socks skip, blacklist, proxyErrorTotal metric. Justification: startup health gate.
- `gmaps/job.go`, `gmaps/place.go`, `gmaps/searchjob.go` — maxRetries 3→5, MaxRetryDelay 10s, `gmaps/backoff.go` Retry-After support, `fetchReviewPage` retry loop. Justification: resilience for 429/5xx.

## 2026-08-31 — Phase 1 product seam (no upstream logic change)
- `internal/msg/domain/rawplace.go` — new product namespace, no modification to `gmaps.Entry`.
- `internal/msg/acquisition/mapper.go` — pure mapper `gmaps.Entry` → `RawPlaceV1`, copies slices, handles Longtitude→Longitude.
- No embedding of CRM/pipeline fields into `gmaps.Entry` (per invariant).
- No change to `gmaps.Entry` struct tags or parsing logic.

Next touches must be appended here.
