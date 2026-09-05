package pipeline_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/gosom/google-maps-scraper/desktop/storage/sqlite"
	"github.com/gosom/google-maps-scraper/internal/msg/pipeline"
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

func seedPipelineLead(t *testing.T, db *sql.DB, id, title, stage, followUp string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO businesses(id, title, first_seen_at, last_seen_at) VALUES(?,?, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`, id, title)
	if err != nil {
		t.Fatalf("seed business: %v", err)
	}
	var fuVal interface{} = nil
	if followUp != "" {
		fuVal = followUp
	}
	_, err = db.Exec(`INSERT INTO lead_states(business_id, stage, next_follow_up_at) VALUES(?,?,?)`, id, stage, fuVal)
	if err != nil {
		t.Fatalf("seed lead_state: %v", err)
	}
}

func TestPipeline_MoveStageAndKanban(t *testing.T) {
	db := newTestDB(t)
	seedPipelineLead(t, db, "p1", "Klinik Sehat", "qualified", "")
	seedPipelineLead(t, db, "p2", "Resto Enak", "contacted", "")

	svc := pipeline.New(db)

	board, err := svc.GetKanbanBoard()
	if err != nil {
		t.Fatalf("get board: %v", err)
	}
	if len(board["qualified"]) != 1 || len(board["contacted"]) != 1 {
		t.Fatalf("unexpected board counts: %+v", board)
	}

	// Move p1 to proposal
	if err := svc.MoveStage("p1", "proposal"); err != nil {
		t.Fatalf("move stage: %v", err)
	}

	boardAfter, _ := svc.GetKanbanBoard()
	if len(boardAfter["qualified"]) != 0 || len(boardAfter["proposal"]) != 1 {
		t.Fatalf("stage transition failed in board: %+v", boardAfter)
	}

	// Invalid stage should fail
	if err := svc.MoveStage("p1", "bogus_stage"); err == nil {
		t.Fatalf("expected error on invalid stage")
	}

	// Check activity logged
	var actCount int
	_ = db.QueryRow(`SELECT count(*) FROM activities WHERE business_id='p1' AND type='stage_changed'`).Scan(&actCount)
	if actCount != 1 {
		t.Fatalf("activity not logged for stage move, got count %d", actCount)
	}
}

func TestPipeline_FollowUpsOverdue(t *testing.T) {
	db := newTestDB(t)
	past := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	future := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)

	seedPipelineLead(t, db, "fu1", "Overdue Lead", "contacted", past)
	seedPipelineLead(t, db, "fu2", "Upcoming Lead", "qualified", future)

	svc := pipeline.New(db)
	items, err := svc.ListFollowUps()
	if err != nil {
		t.Fatalf("list followups: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 followups, got %d", len(items))
	}

	var overdueCount int
	for _, it := range items {
		if it.IsOverdue {
			overdueCount++
			if it.BusinessID != "fu1" {
				t.Fatalf("expected fu1 to be overdue, got %s", it.BusinessID)
			}
		}
	}
	if overdueCount != 1 {
		t.Fatalf("expected exactly 1 overdue, got %d", overdueCount)
	}
}
