# Legacy CSV Contract Audit — Go writer vs Node prototype

**Date:** 2026-08-31
**Scope:** `gmaps.Entry.CsvHeaders()` vs `packages/scraper/process.js` + `packages/scraper/index.js`

## Go writer headers (35)
`input_id, link, title, category, address, open_hours, popular_times, website, phone, plus_code, review_count, review_rating, reviews_per_rating, latitude, longitude, cid, status, descriptions, reviews_link, thumbnail, timezone, price_range, data_id, street_view_url, place_id, images, reservations, order_online, menu, owner, complete_address, credit_cards_accepted, about, user_reviews, user_reviews_extended, emails`

Source: `gmaps/entry.go:235 CsvHeaders()`

## Node prototype reads (process.js)
- `lead.place_id || lead.title` → expects `place_id`
- `lead.title` → `title` ✓
- `lead.website` → `website` ✓
- `lead.category` → `category` ✓
- `lead.rating` ← expects `rating`, Go provides `review_rating` ✗ mismatch
- `lead.reviews` ← expects `reviews`, Go provides `review_count` ✗ mismatch
- `lead.url` ← expects `url`, Go provides `link` ✗ mismatch
- `lead.address` → `address` ✓
- `lead.phone` → `phone` ✓
- `lead.status` → `status` ✓
- Final outputmaps to `maps_url: lead.url` (so link→url mismatch propagates to output)

Also `index.js` writes single query to `temp_query.txt` then `docker run gosom/google-maps-scraper -input /queries.txt -results /out/temp_results.csv` — Go writes CSV with above headers, Node then parses via `csv-parser`.

## Mismatch matrix

| Node expects | Go provides | Match? | Impact |
|---|---|---|---|
| `title` | `title` | ✓ | ok |
| `category` | `category` | ✓ | ok |
| `phone` | `phone` | ✓ | ok |
| `address` | `address` | ✓ | ok |
| `website` | `website` | ✓ | ok |
| `status` | `status` | ✓ | ok |
| `place_id` | `place_id` | ✓ | ok |
| `url` | `link` | ✗ | Node gets undefined `lead.url` → `maps_url` empty in leads.csv |
| `rating` | `review_rating` | ✗ | `lead.rating` undefined → score input wrong |
| `reviews` | `review_count` | ✗ | `lead.reviews` undefined → score bucket wrong |
| `reviews_link` | `reviews_link` | ✓ but Node never reads | unused |
| `cid` | `cid` | ✓ but Node never reads | lost identity |
| `emails` | `emails` | ✓ but Node never reads | lost emails |

## Decision
Do not silently fix prototype. Record mismatch and require compatibility adapter + golden tests before production Desktop switches to structured `RawPlaceV1` instead of CSV.

**Recommendation:**
- Keep legacy prototype unchanged (frozen).
- Product Desktop must NOT read CSV headers directly; must use `RawPlaceV1` via `MapEntry` (already enforced - no product code reads CSV).
- If legacy pipeline must be supported temporarily, create adapter `internal/msg/compat/csvadapter.go` that maps `review_rating→rating`, `review_count→reviews`, `link→url` with tests. Not in Phase 1 scope; backlog.

## Verification
- `grep -r "docker run.*gosom/google-maps-scraper" packages/scraper/index.js` confirms prototype Docker path.
- Guard test `internal/msg/guard_test.go` ensures future product code never uses that path.
