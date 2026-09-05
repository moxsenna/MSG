-- 002_backfill_city: fill empty city from address_text for rows scraped before City was persisted
UPDATE businesses SET city = address_text, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
WHERE (city IS NULL OR city = '') AND address_text IS NOT NULL AND address_text != '';
