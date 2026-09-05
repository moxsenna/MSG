package main

import (
	"testing"

	"github.com/gosom/google-maps-scraper/internal/msg/ai"
)

func TestAnalyzeLeadStubSuccessAndPersist(t *testing.T) {
	app := newAIApp(t)
	_, err := app.db.Exec(`INSERT INTO businesses(id, title, primary_category, city, website, phone, address_text, first_seen_at, last_seen_at) VALUES('biz1','Toko Maju','retail','cirebon','https://maju.id','+6281000','Jl. Maju 1',datetime('now'),datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.db.Exec(`UPDATE businesses SET review_count=50, review_rating=4.5 WHERE id='biz1'`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec(`INSERT INTO lead_states(business_id, stage) VALUES('biz1','new')`); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec(`INSERT INTO ai_key_slots(id, provider, label, key_hint, enabled, priority, created_at, updated_at) VALUES('stub-1','gemini','stub akun','****0000',1,0,datetime('now'),datetime('now'))`); err != nil {
		t.Fatal(err)
	}
	app.aiChain = ai.NewChain(func(slot ai.KeySlot) (ai.Provider, error) {
		return ai.NewFakeProvider(), nil
	})
	res, err := app.AnalyzeLead("biz1")
	if err != nil {
		t.Fatalf("AnalyzeLead stub failed: %v", err)
	}
	if res.Summary == "" || res.UsedLabel != "stub akun" || res.UsedProvider != "gemini" {
		t.Errorf("unexpected result: %+v", res)
	}
	var cnt int
	_ = app.db.QueryRow(`SELECT COUNT(*) FROM ai_analyses WHERE business_id='biz1'`).Scan(&cnt)
	if cnt != 1 {
		t.Errorf("expected 1 persisted analysis, got %d", cnt)
	}
}

func TestAnalyzeLeadNoKeysHonest(t *testing.T) {
	app := newAIApp(t)
	_, err := app.db.Exec(`INSERT INTO businesses(id, title, primary_category, city, website, phone, address_text, first_seen_at, last_seen_at) VALUES('biz2','Toko Sepi','retail','cirebon','','+6281001','',datetime('now'),datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec(`INSERT INTO lead_states(business_id, stage) VALUES('biz2','new')`); err != nil {
		t.Fatal(err)
	}
	if _, err := app.AnalyzeLead("biz2"); err == nil {
		t.Error("expected honest no-key error, got nil")
	} else {
		t.Logf("honest error: %v", err)
	}
}
