package outreach_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/gosom/google-maps-scraper/desktop/storage/sqlite"
	"github.com/gosom/google-maps-scraper/internal/msg/outreach"
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

func seedTestBusiness(t *testing.T, db *sql.DB, id, title, phone, email string, dnc bool) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO businesses(id, title, phone, emails_json, first_seen_at, last_seen_at) VALUES(?,?,?,?,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`,
		id, title, phone, `["`+email+`"]`)
	if err != nil {
		t.Fatalf("seed business: %v", err)
	}
	dncInt := 0
	if dnc {
		dncInt = 1
	}
	_, err = db.Exec(`INSERT INTO lead_states(business_id, stage, do_not_contact) VALUES(?, 'new', ?)`, id, dncInt)
	if err != nil {
		t.Fatalf("seed lead_state: %v", err)
	}
}

func TestOutreach_DNCBlocksAllChannels(t *testing.T) {
	db := newTestDB(t)
	seedTestBusiness(t, db, "b_dnc", "DNC Corp", "+628123456789", "boss@dnc.com", true)

	svc := outreach.New(db)

	// 1. CreateDraft blocked
	_, err := svc.CreateDraft("b_dnc", "whatsapp", "friendly", "Hello there")
	if err != outreach.ErrDNCBlocked {
		t.Fatalf("expected ErrDNCBlocked on CreateDraft, got %v", err)
	}

	// 2. OpenWhatsApp blocked
	_, err = svc.OpenWhatsApp("b_dnc", "")
	if err != outreach.ErrDNCBlocked {
		t.Fatalf("expected ErrDNCBlocked on OpenWhatsApp, got %v", err)
	}

	// 3. OpenEmail blocked
	_, err = svc.OpenEmail("b_dnc", "")
	if err != outreach.ErrDNCBlocked {
		t.Fatalf("expected ErrDNCBlocked on OpenEmail, got %v", err)
	}

	// 4. MarkContacted blocked
	err = svc.MarkContacted("b_dnc", "", "whatsapp")
	if err != outreach.ErrDNCBlocked {
		t.Fatalf("expected ErrDNCBlocked on MarkContacted, got %v", err)
	}
}

func TestOutreach_HappyPathAndExternalOpenNotSent(t *testing.T) {
	db := newTestDB(t)
	seedTestBusiness(t, db, "b_ok", "Active Biz", "081298765432", "info@active.com", false)

	svc := outreach.New(db)

	// Create Draft
	draft, err := svc.CreateDraft("b_ok", "whatsapp", "friendly", "Penawaran website toko")
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if draft.ID == "" || draft.Body != "Penawaran website toko" {
		t.Fatalf("unexpected draft: %+v", draft)
	}

	// Open WhatsApp - should return wa.me URL with text and NOT mark stage contacted
	waURL, err := svc.OpenWhatsApp("b_ok", draft.ID)
	if err != nil {
		t.Fatalf("open wa: %v", err)
	}
	if !strings.Contains(waURL, "https://wa.me/6281298765432") {
		t.Fatalf("invalid wa URL format: %s", waURL)
	}

	// Invariant: external open creates event but stage stays 'new'
	var stage string
	_ = db.QueryRow(`SELECT stage FROM lead_states WHERE business_id='b_ok'`).Scan(&stage)
	if stage != "new" {
		t.Fatalf("stage should remain 'new' after opening external app, got %s", stage)
	}

	// Open Email
	mailURL, err := svc.OpenEmail("b_ok", draft.ID)
	if err != nil {
		t.Fatalf("open email: %v", err)
	}
	if !strings.HasPrefix(mailURL, "mailto:info%40active.com") && !strings.HasPrefix(mailURL, "mailto:info@active.com") {
		t.Fatalf("invalid mailto URL: %s", mailURL)
	}

	// Explicit mark contacted by user
	if err := svc.MarkContacted("b_ok", draft.ID, "whatsapp"); err != nil {
		t.Fatalf("mark contacted: %v", err)
	}
	_ = db.QueryRow(`SELECT stage FROM lead_states WHERE business_id='b_ok'`).Scan(&stage)
	if stage != "contacted" {
		t.Fatalf("stage should become 'contacted' after explicit user mark, got %s", stage)
	}
}
