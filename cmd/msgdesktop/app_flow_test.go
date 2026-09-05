package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gosom/google-maps-scraper/desktop/storage/sqlite"
	"github.com/gosom/google-maps-scraper/internal/msg/acquisition"
	"github.com/gosom/google-maps-scraper/internal/msg/ingest"
	"github.com/gosom/google-maps-scraper/internal/msg/leads"
	"github.com/gosom/google-maps-scraper/internal/msg/pipeline"
	"github.com/gosom/google-maps-scraper/internal/msg/outreach"
	"github.com/gosom/google-maps-scraper/internal/msg/scoring"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	dir, err := os.MkdirTemp("", "msg-appflow-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	db, err := sqlite.Open(dir + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := sqlite.Migrate(db); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	app.ctx = context.Background()
	app.db = db
	app.leads = leads.New(db)
	app.ingest = ingest.New(db)
	app.pipeline = pipeline.New(db)
	app.adapter = acquisition.NewAdapter()
	app.scoring = scoring.New(db)
	app.outreach = outreach.New(db)
	return app
}

func TestAppFullFlowRealCirebon(t *testing.T) {
	if testing.Short() {
		t.Skip("skip real scrape in short mode")
	}
	app := newTestApp(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	app.ctx = ctx

	runID, err := app.Search(acquisition.SearchRequest{
		Query: "rental mobil", LocationText: "cirebon",
		RadiusMeters: 10000, SpeedPreset: "balanced",
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if runID == "" {
		t.Fatal("empty runID")
	}

	res, err := app.ListLeads(leads.Query{Limit: 50, Offset: 0, SortBy: "updated"})
	if err != nil {
		t.Fatalf("ListLeads failed: %v", err)
	}
	if res.Total == 0 || len(res.Items) == 0 {
		t.Fatalf("expected real leads, got total=%d items=%d", res.Total, len(res.Items))
	}
	t.Logf("leads total=%d first=%q city=%q", res.Total, res.Items[0].Title, res.Items[0].City)
	if res.Items[0].City == "" {
		t.Errorf("expected City filled (backfill/mapper), got empty for %q", res.Items[0].Title)
	}

	board, err := app.GetKanbanBoard()
	if err != nil {
		t.Fatalf("GetKanbanBoard failed: %v", err)
	}
	if len(board["new"]) == 0 {
		t.Errorf("expected new-stage cards, board keys=%v", keys(board))
	} else {
		t.Logf("kanban new=%d", len(board["new"]))
	}

	firstID := res.Items[0].ID
	if err := app.SetLeadFollowUp(firstID, "2026-09-10"); err != nil {
		t.Fatalf("SetLeadFollowUp failed: %v", err)
	}
	fu, err := app.ListFollowUps()
	if err != nil {
		t.Fatalf("ListFollowUps failed: %v", err)
	}
	if len(fu) == 0 {
		t.Error("expected 1 follow-up after SetLeadFollowUp, got 0")
	} else {
		t.Logf("followups=%d first=%q at=%s", len(fu), fu[0].Title, fu[0].FollowUpAt)
	}

	acts, err := app.ListActivities(50)
	if err != nil {
		t.Fatalf("ListActivities failed: %v", err)
	}
	if len(acts) == 0 {
		t.Error("expected activities after search, got 0")
	} else {
		t.Logf("activities=%d first type=%s title=%q", len(acts), acts[0].Type, acts[0].Title)
	}
}

func TestAppListLeadsEmptyDB(t *testing.T) {
	app := newTestApp(t)
	res, err := app.ListLeads(leads.Query{Limit: 50})
	if err != nil {
		t.Fatalf("ListLeads on empty DB failed: %v", err)
	}
	if res.Items == nil {
		t.Error("Items must be empty slice, not nil (Wails null-tuple guard)")
	}
	if res.Total != 0 || len(res.Items) != 0 {
		t.Errorf("expected 0/0 on empty DB, got %d/%d", res.Total, len(res.Items))
	}
	fu, err := app.ListFollowUps()
	if err != nil {
		t.Fatalf("ListFollowUps on empty DB failed: %v", err)
	}
	if fu == nil {
		t.Error("follow-ups must be empty slice, not nil")
	}
	acts, err := app.ListActivities(50)
	if err != nil {
		t.Fatalf("ListActivities on empty DB failed: %v", err)
	}
	if len(acts) != 0 {
		t.Errorf("expected 0 activities, got %d", len(acts))
	}
}

func TestMigration002BackfillsCity(t *testing.T) {
	app := newTestApp(t)
	_, err := app.db.Exec(`INSERT INTO businesses(id, title, address_text, city, first_seen_at, last_seen_at) VALUES('b1','T','cirebon','',datetime('now'),datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec(`DELETE FROM schema_migrations WHERE version='002_backfill_city.sql'`); err != nil {
		t.Fatal(err)
	}
	if err := sqlite.Migrate(app.db); err != nil {
		t.Fatal(err)
	}
	var city string
	if err := app.db.QueryRow(`SELECT city FROM businesses WHERE id='b1'`).Scan(&city); err != nil {
		t.Fatal(err)
	}
	if city != "cirebon" {
		t.Errorf("expected backfilled city=cirebon, got %q", city)
	}
}

func keys(m map[string][]pipeline.StageCard) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		if len(v) > 0 {
			out = append(out, k)
		}
	}
	return out
}
