package main

import (
	"strings"
	"testing"
)

func TestRunWebsiteAuditLiveRescores(t *testing.T) {
	if testing.Short() {
		t.Skip("needs live network + browser")
	}
	app := newAIApp(t)
	_, err := app.db.Exec(`INSERT INTO businesses(id, title, primary_category, city, website, phone, address_text, first_seen_at, last_seen_at) VALUES('audit1','RentGo Test','rental mobil','cirebon','http://rentgoindonesia.com/','', 'Jl. A',datetime('now'),datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec(`INSERT INTO lead_states(business_id, stage) VALUES('audit1','new')`); err != nil {
		t.Fatal(err)
	}
	res, err := app.RunWebsiteAudit("audit1")
	if err != nil {
		t.Fatalf("audit failed: %v", err)
	}
	t.Logf("reachable=%v https=%v contact=%v wa=%v title=%q newScore=%d errcode=%q", res.Reachable, res.HTTPS, res.HasContact, res.HasWhatsApp, res.Title, res.NewScore, res.ErrorCode)
	if !res.Reachable {
		t.Error("expected reachable=true")
	}
	if res.NewScore <= 0 {
		t.Errorf("expected rescore >0, got %d", res.NewScore)
	}
	var cnt int
	_ = app.db.QueryRow(`SELECT COUNT(*) FROM website_audits WHERE business_id='audit1'`).Scan(&cnt)
	if cnt == 0 {
		t.Error("expected website_audits row persisted")
	}
}

func TestRunWebsiteAuditNoWebsiteHonest(t *testing.T) {
	app := newAIApp(t)
	_, err := app.db.Exec(`INSERT INTO businesses(id, title, primary_category, city, website, phone, address_text, first_seen_at, last_seen_at) VALUES('audit2','Tanpa Web','cafe','cirebon','','+6281','Jl. B',datetime('now'),datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec(`INSERT INTO lead_states(business_id, stage) VALUES('audit2','new')`); err != nil {
		t.Fatal(err)
	}
	_, err = app.RunWebsiteAudit("audit2")
	if err == nil || !strings.Contains(err.Error(), "no website") {
		t.Errorf("expected no-website honest error, got %v", err)
	}
}

func TestOpenEmailURLNoEmailHonest(t *testing.T) {
	app := newAIApp(t)
	_, err := app.db.Exec(`INSERT INTO businesses(id, title, primary_category, city, website, phone, address_text, first_seen_at, last_seen_at) VALUES('mail1','Tanpa Email','cafe','cirebon','','+6281','Jl. C',datetime('now'),datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec(`INSERT INTO lead_states(business_id, stage) VALUES('mail1','new')`); err != nil {
		t.Fatal(err)
	}
	_, err = app.OpenEmailURL("mail1", "")
	if err == nil || !strings.Contains(err.Error(), "no email") {
		t.Errorf("expected no-email honest error, got %v", err)
	}
}
