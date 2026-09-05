package leads_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/gosom/google-maps-scraper/desktop/storage/sqlite"
	"github.com/gosom/google-maps-scraper/internal/msg/domain"
	"github.com/gosom/google-maps-scraper/internal/msg/ingest"
	"github.com/gosom/google-maps-scraper/internal/msg/leads"
)

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil { t.Fatalf("open: %v", err) }
	if err := sqlite.Migrate(db); err != nil { t.Fatalf("migrate: %v", err) }
	t.Cleanup(func(){ db.Close() })
	return db
}

func seedBusiness(t *testing.T, db *sql.DB, title, city, website, phone, stage string) string {
	t.Helper()
	// ensure search_runs exists for FK
	_, _ = db.Exec(`INSERT OR IGNORE INTO search_runs(id, query, status) VALUES('run-test','q','completed')`)
	svc := ingest.New(db)
	place := domain.RawPlaceV1{
		SourceVersion: domain.RawPlaceVersion,
		Title: title,
		Category: "cafe",
		CompleteAddress: domain.CompleteAddress{City: city},
		Website: website,
		Phone: phone,
		PlaceID: title + "-pid",
		CID: title + "-cid",
	}
	bid, _, err := svc.Ingest(context.Background(), "run-test", place)
	if err != nil { t.Fatalf("ingest %s: %v", title, err) }
	if stage != "new" {
		_, _ = db.Exec(`UPDATE lead_states SET stage=? WHERE business_id=?`, stage, bid)
	}
	return bid
}

func TestList_PaginationAndFilters(t *testing.T) {
	db := newDB(t)
	// seed 5 businesses
	seedBusiness(t, db, "Warung A", "Bandung", "https://a.com", "+628111111111", "new")
	seedBusiness(t, db, "Warung B", "Bandung", "", "+628222222222", "qualified")
	seedBusiness(t, db, "Hotel C", "Jakarta", "https://c.com", "+628333333333", "contacted")
	seedBusiness(t, db, "Cafe D", "Bandung", "", "+628444444444", "new")
	bidE := seedBusiness(t, db, "Warung E", "Surabaya", "https://e.com", "+628555555555", "new")
	// set DNC on E
	db.Exec(`UPDATE lead_states SET do_not_contact=1 WHERE business_id=?`, bidE)

	svc := leads.New(db)

	// total 5
	items, total, err := svc.List(leads.Query{Limit: 10})
	if err != nil || total != 5 || len(items) != 5 { t.Fatalf("list total=%d len=%d err=%v", total, len(items), err) }

	// pagination limit 2
	items, total, _ = svc.List(leads.Query{Limit: 2, Offset: 0})
	if len(items) != 2 || total != 5 { t.Fatalf("pagination") }
	items, _, _ = svc.List(leads.Query{Limit: 2, Offset: 2})
	if len(items) != 2 { t.Fatalf("offset") }

	// filter stage
	items, total, _ = svc.List(leads.Query{Stage: "new"})
	if total != 3 { t.Fatalf("stage new total=%d", total) }

	// filter city
	items, total, _ = svc.List(leads.Query{City: "Bandung"})
	if total != 3 { t.Fatalf("city total=%d", total) }

	// hasWebsite false => 2 (B,D)
	f := false
	items, total, _ = svc.List(leads.Query{HasWebsite: &f})
	if total != 2 { t.Fatalf("hasWebsite false total=%d", total) }

	// DNC true => 1 (E)
	tr := true
	items, total, _ = svc.List(leads.Query{DNC: &tr})
	if total != 1 || items[0].Title != "Warung E" { t.Fatalf("DNC") }

	// search
	items, total, _ = svc.List(leads.Query{Search: "warung"})
	if total != 3 { t.Fatalf("search warung total=%d", total) }
	_ = items
}

func TestUpdateStageAndDNC(t *testing.T) {
	db := newDB(t)
	bid := seedBusiness(t, db, "Test Biz", "Bandung", "", "+628111111111", "new")
	svc := leads.New(db)
	if err := svc.UpdateStage(bid, "qualified"); err != nil { t.Fatalf("update stage: %v", err) }
	d, _ := svc.Get(bid)
	if d.Stage != "qualified" { t.Fatalf("stage %s", d.Stage) }
	if err := svc.UpdateStage(bid, "invalid"); err == nil { t.Fatalf("expected invalid stage error") }
	if err := svc.SetDNC(bid, true); err != nil { t.Fatalf("set dnc") }
	d, _ = svc.Get(bid)
	if !d.DNC { t.Fatalf("dnc") }
}

func TestAddNoteAndTag(t *testing.T) {
	db := newDB(t)
	bid := seedBusiness(t, db, "Note Biz", "Bandung", "", "+628111111111", "new")
	svc := leads.New(db)
	if err := svc.AddNote(bid, "hello"); err != nil { t.Fatalf("add note") }
	if err := svc.AddNote(bid, " "); err == nil { t.Fatalf("empty note should fail") }
	if err := svc.AddTag(bid, "vip"); err != nil { t.Fatalf("add tag") }
	if err := svc.AddTag(bid, "vip"); err != nil { t.Fatalf("duplicate tag should be idempotent") }
	d, _ := svc.Get(bid)
	if len(d.Notes) != 1 || len(d.Tags) != 1 { t.Fatalf("notes/tags len %d %d", len(d.Notes), len(d.Tags)) }
}

func TestGetDetailIncludesCRM(t *testing.T) {
	db := newDB(t)
	bid := seedBusiness(t, db, "Detail Biz", "Bandung", "https://x.com", "+628111111111", "contacted")
	svc := leads.New(db)
	svc.AddNote(bid, "call tomorrow")
	d, err := svc.Get(bid)
	if err != nil { t.Fatalf("get: %v", err) }
	if d.Title != "Detail Biz" || d.Stage != "contacted" { t.Fatalf("detail") }
	if len(d.Notes) == 0 { t.Fatalf("notes") }
	if len(d.Activities) == 0 { t.Fatalf("activities") }
}
