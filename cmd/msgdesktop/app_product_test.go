package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gosom/google-maps-scraper/internal/msg/acquisition"
)

func seedBusiness(t *testing.T, app *App, id, title string) {
	t.Helper()
	_, err := app.db.Exec(`INSERT INTO businesses(id, title, primary_category, city, website, phone, address_text, first_seen_at, last_seen_at) VALUES(?,?,?,?,?,?,?,datetime('now'),datetime('now'))`,
		id, title, "rental mobil", "cirebon", "", "+6281000111", "Jl. Test 1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec(`INSERT INTO lead_states(business_id, stage) VALUES(?,'new')`, id); err != nil {
		t.Fatal(err)
	}
}

func TestScoreMissingBackfills(t *testing.T) {
	app := newAIApp(t)
	seedBusiness(t, app, "sb1", "Rental Test")
	n, err := app.scoring.ScoreMissing()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 scored, got %d", n)
	}
	var score int
	_ = app.db.QueryRow(`SELECT overall_score FROM opportunity_scores WHERE business_id='sb1'`).Scan(&score)
	if score <= 0 {
		t.Errorf("expected positive score (no website + phone + active), got %d", score)
	}
	n, _ = app.scoring.ScoreMissing()
	if n != 0 {
		t.Errorf("expected idempotent 0, got %d", n)
	}
}

func TestDeleteLeadCascades(t *testing.T) {
	app := newAIApp(t)
	seedBusiness(t, app, "del1", "Hapus Saya")
	if err := app.leads.AddNote("del1", "catatan"); err != nil {
		t.Fatal(err)
	}
	if err := app.DeleteLead("del1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	var cnt int
	_ = app.db.QueryRow(`SELECT COUNT(*) FROM businesses WHERE id='del1'`).Scan(&cnt)
	if cnt != 0 {
		t.Error("business row still exists")
	}
	_ = app.db.QueryRow(`SELECT COUNT(*) FROM lead_states WHERE business_id='del1'`).Scan(&cnt)
	if cnt != 0 {
		t.Error("lead_states not cascaded")
	}
	_ = app.db.QueryRow(`SELECT COUNT(*) FROM notes WHERE business_id='del1'`).Scan(&cnt)
	if cnt != 0 {
		t.Error("notes not cascaded")
	}
	if err := app.DeleteLead("del1"); err == nil {
		t.Error("expected not-found on second delete")
	}
}

func TestDraftOutreachAIDNCBlockedFirst(t *testing.T) {
	app := newAIApp(t)
	seedBusiness(t, app, "dnc1", "Jangan Hubungi")
	if err := app.leads.SetDNC("dnc1", true); err != nil {
		t.Fatal(err)
	}
	_, err := app.DraftOutreachAI("dnc1", "whatsapp", "friendly", "")
	if err == nil || !strings.Contains(err.Error(), "DNC_BLOCKED") {
		t.Errorf("expected DNC_BLOCKED before any AI call, got %v", err)
	}
}

func TestBackupNowCreatesZip(t *testing.T) {
	app := newAIApp(t)
	seedBusiness(t, app, "b1", "Backup Test")
	p, err := app.BackupNow()
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	st, err := os.Stat(p)
	if err != nil || st.Size() == 0 {
		t.Errorf("backup zip missing/empty: %s err=%v", p, err)
	}
}

func TestSettingRoundtrip(t *testing.T) {
	app := newAIApp(t)
	if err := app.SetSetting("telemetry_opt_in", "1"); err != nil {
		t.Fatal(err)
	}
	v, err := app.GetSetting("telemetry_opt_in")
	if err != nil || v != "1" {
		t.Errorf("expected 1, got %q err=%v", v, err)
	}
}

func TestCancelSearchReal(t *testing.T) {
	if testing.Short() {
		t.Skip("needs live browser")
	}
	app := newTestApp(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	app.ctx = ctx
	done := make(chan error, 1)
	go func() {
		_, err := app.Search(acquisition.SearchRequest{Query: "rental mobil", LocationText: "cirebon", RadiusMeters: 10000, SpeedPreset: "balanced"})
		done <- err
	}()
	time.Sleep(4 * time.Second)
	app.CancelSearch()
	select {
	case err := <-done:
		t.Logf("search returned after cancel: %v", err)
		if err == nil {
			t.Error("expected error after cancel, got nil")
		}
	case <-time.After(75 * time.Second):
		t.Fatal("Search did not return after CancelSearch")
	}
}
