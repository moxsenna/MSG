package ingest

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gosom/google-maps-scraper/internal/msg/domain"
)

// Service handles ingestion of RawPlaceV1 into SQLite with dedupe and CRM preservation.
type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

// Ingest persists a RawPlaceV1 as a source_snapshot and merges into businesses.
// It preserves CRM state (lead_states, notes, tags, follow_up, DNC) on re-scrape.
func (s *Service) Ingest(ctx context.Context, searchRunID string, place domain.RawPlaceV1) (businessID string, isNew bool, err error) {
	normalizedJSON, _ := json.Marshal(place)
	hash := sha256.Sum256(normalizedJSON)
	contentHash := fmt.Sprintf("%x", hash[:8])
	rawJSON := normalizedJSON // for v1 we store normalized as both

	// Dedupe identity order: place_id > cid > data_id > source_link
	businessID, err = s.findExistingBusiness(ctx, place)
	if err != nil {
		return "", false, err
	}
	isNew = businessID == ""
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now().UTC().Format(time.RFC3339)
	if isNew {
		businessID = uuid.NewString()
		_, err = tx.ExecContext(ctx, `INSERT INTO businesses(id, place_id, cid, data_id, source_link, title, primary_category, categories_json, address_text, borough, street, city, postal_code, state, country, website, phone, emails_json, review_count, review_rating, status, price_range, latitude, longitude, plus_code, timezone, first_seen_at, last_seen_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			businessID, place.PlaceID, place.CID, place.DataID, place.Link, place.Title, place.Category, mustJSON(place.Categories), place.Address, place.CompleteAddress.Borough, place.CompleteAddress.Street, place.CompleteAddress.City, place.CompleteAddress.PostalCode, place.CompleteAddress.State, place.CompleteAddress.Country, place.Website, place.Phone, mustJSON(place.Emails), place.ReviewCount, place.ReviewRating, place.Status, place.PriceRange, place.Latitude, place.Longitude, place.PlusCode, place.Timezone, now, now)
		if err != nil {
			return "", false, fmt.Errorf("insert business: %w", err)
		}
		// also insert primary contact if phone present
		if place.Phone != "" {
			_, _ = tx.ExecContext(ctx, `INSERT INTO business_contacts(id, business_id, type, value, normalized_value, is_primary, first_seen_at, last_seen_at) VALUES(?,?,?,?,?,?,?,?)`,
				uuid.NewString(), businessID, "phone", place.Phone, normalizePhone(place.Phone), 1, now, now)
		}
		for _, soc := range place.Socials {
			soc = strings.TrimSpace(soc)
			if soc == "" {
				continue
			}
			_, _ = tx.ExecContext(ctx, `INSERT INTO business_contacts(id, business_id, type, value, normalized_value, is_primary, first_seen_at, last_seen_at) VALUES(?,?,?,?,?,?,?,?)`,
				uuid.NewString(), businessID, "social", soc, strings.ToLower(soc), 0, now, now)
		}
		// create initial lead_state row
		_, _ = tx.ExecContext(ctx, `INSERT OR IGNORE INTO lead_states(business_id, stage) VALUES(?, 'new')`, businessID)
		_, _ = tx.ExecContext(ctx, `INSERT INTO activities(id, business_id, search_run_id, type, payload_json) VALUES(?,?,?, 'discovered', '{}')`, uuid.NewString(), businessID, searchRunID)
	} else {
		// Update source facts only, never touch lead_states
		_, err = tx.ExecContext(ctx, `UPDATE businesses SET title=?, primary_category=?, categories_json=?, address_text=?, borough=?, street=?, city=?, postal_code=?, state=?, country=?, website=?, phone=?, emails_json=?, review_count=?, review_rating=?, status=?, price_range=?, latitude=?, longitude=?, plus_code=?, timezone=?, last_seen_at=?, updated_at=? WHERE id=?`,
			place.Title, place.Category, mustJSON(place.Categories), place.Address, place.CompleteAddress.Borough, place.CompleteAddress.Street, place.CompleteAddress.City, place.CompleteAddress.PostalCode, place.CompleteAddress.State, place.CompleteAddress.Country, place.Website, place.Phone, mustJSON(place.Emails), place.ReviewCount, place.ReviewRating, place.Status, place.PriceRange, place.Latitude, place.Longitude, place.PlusCode, place.Timezone, now, now, businessID)
		if err != nil {
			return "", false, fmt.Errorf("update business: %w", err)
		}
		_, _ = tx.ExecContext(ctx, `INSERT INTO activities(id, business_id, search_run_id, type, payload_json) VALUES(?,?,?, 'rescraped', '{}')`, uuid.NewString(), businessID, searchRunID)
	}

	// Insert source_snapshot
	snapID := uuid.NewString()
	sourceKey := place.PlaceID
	if sourceKey == "" {
		sourceKey = place.CID
	}
	if sourceKey == "" {
		sourceKey = place.Link
	}
	// content_hash dedupe: if same searchRun+hash exists, skip
	_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO source_snapshots(id, search_run_id, business_id, source, source_key, schema_version, normalized_json, raw_json, collected_at, content_hash) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		snapID, searchRunID, businessID, "gmaps", sourceKey, place.SourceVersion, string(normalizedJSON), string(rawJSON), place.CapturedAt.Format(time.RFC3339), contentHash)
	if err != nil {
		return "", false, fmt.Errorf("insert snapshot: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return "", false, err
	}
	return businessID, isNew, nil
}

func (s *Service) findExistingBusiness(ctx context.Context, p domain.RawPlaceV1) (string, error) {
	// strong identifiers in order
	queries := []struct {
		field string
		value string
	}{
		{"place_id", p.PlaceID},
		{"cid", p.CID},
		{"data_id", p.DataID},
		{"source_link", p.Link},
	}
	for _, q := range queries {
		if q.value == "" {
			continue
		}
		var id string
		err := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT id FROM businesses WHERE %s=? LIMIT 1`, q.field), q.value).Scan(&id)
		if err == nil {
			return id, nil
		}
		if err != sql.ErrNoRows {
			return "", err
		}
	}
	// conservative fuzzy fallback: normalized name+phone+city if all present and strong
	// For V0.1 we only fuzzy when phone matches and city matches to avoid wrong merges
	if p.Phone != "" && p.Title != "" {
		normPhone := normalizePhone(p.Phone)
		var id string
		err := s.db.QueryRowContext(ctx, `SELECT b.id FROM businesses b JOIN business_contacts c ON c.business_id=b.id WHERE c.normalized_value=? AND lower(b.title)=lower(?) AND lower(b.city)=lower(?) LIMIT 1`, normPhone, strings.TrimSpace(p.Title), strings.TrimSpace(p.CompleteAddress.City)).Scan(&id)
		if err == nil {
			return id, nil
		}
		if err != sql.ErrNoRows {
			return "", err
		}
	}
	return "", nil
}

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	if string(b) == "null" {
		return "[]"
	}
	return string(b)
}

func normalizePhone(p string) string {
	var b strings.Builder
	for _, r := range p {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	s := b.String()
	// strip leading 0 or 62 for comparison, keep last 9+ digits
	if strings.HasPrefix(s, "0") {
		s = s[1:]
	}
	if strings.HasPrefix(s, "62") {
		s = s[2:]
	}
	return s
}
