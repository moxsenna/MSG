package main

import (
	"testing"
)

func TestSetDealOutcomeWonLost(t *testing.T) {
	app := newAIApp(t)
	_, err := app.db.Exec(`INSERT INTO businesses(id, title, primary_category, city, website, phone, address_text, first_seen_at, last_seen_at) VALUES('deal1','Toko Deal','retail','cirebon','','+6281','Jl. D',datetime('now'),datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec(`INSERT INTO lead_states(business_id, stage) VALUES('deal1','meeting')`); err != nil {
		t.Fatal(err)
	}
	if err := app.SetDealOutcome("deal1", "won", 15000000, ""); err != nil {
		t.Fatalf("won: %v", err)
	}
	d, err := app.GetLead("deal1")
	if err != nil {
		t.Fatal(err)
	}
	if d.Stage != "won" || d.WonValue == nil || *d.WonValue != 15000000 {
		t.Errorf("unexpected deal state: stage=%q won=%v", d.Stage, d.WonValue)
	}
	if err := app.SetDealOutcome("deal1", "lost", 0, "kemahalan"); err != nil {
		t.Fatalf("lost: %v", err)
	}
	d, _ = app.GetLead("deal1")
	if d.Stage != "lost" || d.LostReason != "kemahalan" {
		t.Errorf("unexpected lost state: %+v", d)
	}
	if err := app.SetDealOutcome("deal1", "meeting", 0, ""); err == nil {
		t.Error("expected rejection for non won/lost stage")
	}
	if err := app.SetDealOutcome("deal1", "won", -5, ""); err == nil {
		t.Error("expected rejection for negative value")
	}
}

func TestTagsRoundtrip(t *testing.T) {
	app := newAIApp(t)
	_, err := app.db.Exec(`INSERT INTO businesses(id, title, primary_category, city, website, phone, address_text, first_seen_at, last_seen_at) VALUES('tag1','Toko Tag','cafe','cirebon','','+6282','Jl. E',datetime('now'),datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec(`INSERT INTO lead_states(business_id, stage) VALUES('tag1','new')`); err != nil {
		t.Fatal(err)
	}
	if err := app.leads.AddTag("tag1", "prioritas"); err != nil {
		t.Fatal(err)
	}
	if err := app.leads.AddTag("tag1", "prioritas"); err != nil {
		t.Fatal(err)
	}
	d, err := app.GetLead("tag1")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Tags) != 1 || d.Tags[0] != "prioritas" {
		t.Errorf("expected deduped tag, got %v", d.Tags)
	}
}

func TestSavedSearchesCRUD(t *testing.T) {
	app := newAIApp(t)
	s, err := app.SaveSearch("rental cirebon", "rental mobil", "cirebon", `{"target":"20"}`)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if s.ID == "" || s.Query != "rental mobil" {
		t.Errorf("unexpected saved: %+v", s)
	}
	rows, err := app.ListSavedSearches()
	if err != nil || len(rows) != 1 {
		t.Fatalf("list: %v %d", err, len(rows))
	}
	if _, err := app.SaveSearch("", "x", "", ""); err == nil {
		t.Error("expected rejection for empty name")
	}
	if err := app.DeleteSavedSearch(s.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	rows, _ = app.ListSavedSearches()
	if len(rows) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(rows))
	}
	if err := app.DeleteSavedSearch(s.ID); err == nil {
		t.Error("expected not-found on second delete")
	}
}

func TestWeeklyReportCounts(t *testing.T) {
	app := newAIApp(t)
	_, err := app.db.Exec(`INSERT INTO businesses(id, title, primary_category, city, website, phone, address_text, first_seen_at, last_seen_at, created_at) VALUES('w1','Baru','cafe','cirebon','','+6281','Jl',datetime('now'),datetime('now'),strftime('%Y-%m-%dT%H:%M:%SZ','now'))`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec(`INSERT INTO lead_states(business_id, stage, won_value, updated_at) VALUES('w1','won',2500000,strftime('%Y-%m-%dT%H:%M:%SZ','now'))`); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec(`INSERT INTO search_runs(id, query, started_at) VALUES('r1','q',strftime('%Y-%m-%dT%H:%M:%SZ','now'))`); err != nil {
		t.Fatal(err)
	}
	r, err := app.WeeklyReport()
	if err != nil {
		t.Fatal(err)
	}
	if r.NewLeads != 1 || r.Won != 1 || r.WonValue != 2500000 || r.SearchRuns != 1 {
		t.Errorf("unexpected report: %+v", r)
	}
	t.Logf("report=%+v", r)
}

func TestGetScoreDetailEvidence(t *testing.T) {
	app := newAIApp(t)
	_, err := app.db.Exec(`INSERT INTO businesses(id, title, primary_category, city, website, phone, address_text, first_seen_at, last_seen_at) VALUES('sc1','Toko Skor','cafe','cirebon','','+6281','Jl. S',datetime('now'),datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec(`INSERT INTO lead_states(business_id, stage) VALUES('sc1','new')`); err != nil {
		t.Fatal(err)
	}
	n, err := app.scoring.ScoreMissing()
	if err != nil || n != 1 {
		t.Fatalf("score missing: %d %v", n, err)
	}
	d, err := app.GetScoreDetail("sc1")
	if err != nil {
		t.Fatal(err)
	}
	if d.Overall <= 0 || len(d.Evidence) == 0 {
		t.Errorf("expected positive score with evidence, got %+v", d)
	}
	t.Logf("score=%d evidence=%d first=%q", d.Overall, len(d.Evidence), d.Evidence[0].Label)
}
