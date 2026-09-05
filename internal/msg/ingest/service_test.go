package ingest_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/gosom/google-maps-scraper/desktop/storage/sqlite"
	"github.com/gosom/google-maps-scraper/internal/msg/domain"
	"github.com/gosom/google-maps-scraper/internal/msg/ingest"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := sqlite.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func seedSearchRun(t *testing.T, db *sql.DB) string {
	t.Helper()
	id := "run1"
	_, err := db.Exec(`INSERT INTO search_runs(id, query, status, started_at) VALUES(?,?, 'completed', ?)`, id, "test", time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("seed run: %v", err)
	}
	return id
}

func samplePlace(title, placeID, cid string) domain.RawPlaceV1 {
	return domain.RawPlaceV1{
		SourceVersion: domain.RawPlaceVersion,
		CapturedAt:    time.Now().UTC(),
		Link:          "https://maps.google.com/?cid=" + cid,
		CID:           cid,
		PlaceID:       placeID,
		DataID:        "data-" + placeID,
		Title:         title,
		Category:      "cafe",
		Categories:    []string{"cafe"},
		Address:       "Jl Test 1",
		CompleteAddress: domain.CompleteAddress{City: "Bandung"},
		Phone:         "+628123456789",
		Website:       "https://example.com",
		ReviewCount:   10,
		ReviewRating:  4.5,
		Latitude:      -6.2, Longitude: 106.8,
	}
}

func TestIngest_NewBusinessCreatesCRMRows(t *testing.T) {
	db := newTestDB(t)
	runID := seedSearchRun(t, db)
	svc := ingest.New(db)
	p := samplePlace("Warung A", "place1", "cid1")
	bid, isNew, err := svc.Ingest(context.Background(), runID, p)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if !isNew || bid == "" {
		t.Fatalf("isNew=%v bid=%q", isNew, bid)
	}
	var stage string
	if err := db.QueryRow(`SELECT stage FROM lead_states WHERE business_id=?`, bid).Scan(&stage); err != nil {
		t.Fatalf("lead_states: %v", err)
	}
	if stage != "new" {
		t.Fatalf("stage=%q", stage)
	}
}

func TestIngest_DedupeByPlaceID(t *testing.T) {
	db := newTestDB(t)
	runID := seedSearchRun(t, db)
	svc := ingest.New(db)
	p := samplePlace("Warung A", "place1", "cid1")
	bid1, _, _ := svc.Ingest(context.Background(), runID, p)
	p2 := samplePlace("Warung A Updated", "place1", "cid1")
	p2.ReviewCount = 99
	bid2, isNew, err := svc.Ingest(context.Background(), runID, p2)
	if err != nil {
		t.Fatalf("second ingest: %v", err)
	}
	if isNew {
		t.Fatal("should not be new")
	}
	if bid1 != bid2 {
		t.Fatalf("bid mismatch %q vs %q", bid1, bid2)
	}
	var cnt int
	_ = db.QueryRow(`SELECT count(*) FROM businesses`).Scan(&cnt)
	if cnt != 1 {
		t.Fatalf("businesses=%d want 1", cnt)
	}
	var rc int
	_ = db.QueryRow(`SELECT review_count FROM businesses WHERE id=?`, bid1).Scan(&rc)
	if rc != 99 {
		t.Fatalf("review not updated rc=%d", rc)
	}
}

func TestIngest_RescrapePreservesCRM(t *testing.T) {
	db := newTestDB(t)
	runID := seedSearchRun(t, db)
	svc := ingest.New(db)
	p := samplePlace("Warung B", "place2", "cid2")
	bid, _, _ := svc.Ingest(context.Background(), runID, p)

	// Simulate user CRM actions
	_, _ = db.Exec(`UPDATE lead_states SET stage='contacted', do_not_contact=1 WHERE business_id=?`, bid)
	_, _ = db.Exec(`INSERT INTO notes(id, business_id, body) VALUES('n1', ?, 'important')`, bid)
	_, _ = db.Exec(`INSERT INTO tags(id, name) VALUES('t1','vip')`)
	_, _ = db.Exec(`INSERT INTO business_tags(business_id, tag_id) VALUES(?, 't1')`, bid)

	// Re-ingest with updated facts
	p2 := samplePlace("Warung B Renamed", "place2", "cid2")
	p2.Phone = "+628999999999"
	_, _, _ = svc.Ingest(context.Background(), runID, p2)

	var stage string
	var dnc int
	_ = db.QueryRow(`SELECT stage, do_not_contact FROM lead_states WHERE business_id=?`, bid).Scan(&stage, &dnc)
	if stage != "contacted" || dnc != 1 {
		t.Fatalf("CRM not preserved stage=%q dnc=%d", stage, dnc)
	}
	var noteCnt int
	_ = db.QueryRow(`SELECT count(*) FROM notes WHERE business_id=?`, bid).Scan(&noteCnt)
	if noteCnt != 1 {
		t.Fatalf("notes lost %d", noteCnt)
	}
	var tagCnt int
	_ = db.QueryRow(`SELECT count(*) FROM business_tags WHERE business_id=?`, bid).Scan(&tagCnt)
	if tagCnt != 1 {
		t.Fatalf("tags lost %d", tagCnt)
	}
	var phone string
	_ = db.QueryRow(`SELECT phone FROM businesses WHERE id=?`, bid).Scan(&phone)
	if phone != "+628999999999" {
		t.Fatalf("phone not updated %q", phone)
	}
}

func TestIngest_SourceSnapshotPreserved(t *testing.T) {
	db := newTestDB(t)
	runID := seedSearchRun(t, db)
	svc := ingest.New(db)
	p := samplePlace("Warung C", "place3", "cid3")
	_, _, _ = svc.Ingest(context.Background(), runID, p)
	_, _, _ = svc.Ingest(context.Background(), runID, p)
	var cnt int
	_ = db.QueryRow(`SELECT count(*) FROM source_snapshots`).Scan(&cnt)
	if cnt < 1 {
		t.Fatalf("snapshots=%d", cnt)
	}
}
