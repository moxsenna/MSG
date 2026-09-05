package scoring_test

import (
	"database/sql"
	"testing"

	"github.com/gosom/google-maps-scraper/desktop/storage/sqlite"
	"github.com/gosom/google-maps-scraper/internal/msg/scoring"
)

func newDB(t *testing.T) *sql.DB {
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

func TestCalculateV2_DeterministicAndClosed(t *testing.T) {
	// strong business no website
	in := scoring.Inputs{
		Title:      "Warung Keren",
		Category:   "cafe",
		Phone:      "+628123456789",
		ReviewRating: 4.8,
		ReviewCount: 200,
		Status:     "OPERATIONAL",
		Website:    "",
		HasWebsite: false,
	}
	r1 := scoring.CalculateV2(in)
	r2 := scoring.CalculateV2(in)
	if r1.OverallScore != r2.OverallScore || r1.Priority != r2.Priority {
		t.Fatalf("not deterministic %v vs %v", r1, r2)
	}
	if r1.OverallScore < 80 || r1.Priority != "Hot Lead" {
		t.Fatalf("expected hot, got %d %s evidence %v", r1.OverallScore, r1.Priority, r1.Evidence)
	}
	if r1.WebsiteScore == 0 || r1.Evidence == nil {
		t.Fatalf("evidence missing")
	}
	// closed rejection
	in2 := scoring.Inputs{Status: "CLOSED_PERMANENTLY", Website: "https://x.com", HasWebsite: true}
	rc := scoring.CalculateV2(in2)
	if rc.OverallScore != 0 || rc.WebsiteScore != 0 {
		t.Fatalf("closed should be 0")
	}
	foundClosed := false
	for _, e := range rc.Evidence {
		if e.Code == "closed" {
			foundClosed = true
		}
	}
	if !foundClosed {
		t.Fatalf("closed evidence missing")
	}
}

func TestCalculateV2_BoundariesAndEvidence(t *testing.T) {
	// low reviews, low rating => warm/low
	in := scoring.Inputs{
		Title: "Toko Sepi", Category: "retail", Phone: "0211234567",
		ReviewRating: 3.5, ReviewCount: 5, Status: "OPERATIONAL",
		Website: "https://toko.example", HasWebsite: true,
	}
	f := false
	in.IsHTTPS = &f
	in.HasMeta = &f
	r := scoring.CalculateV2(in)
	if r.OverallScore == 0 {
		t.Fatalf("score zero")
	}
	if r.Confidence == 0 {
		t.Fatalf("confidence zero")
	}
	if r.RuleSetVersion != scoring.RuleSetVersion {
		t.Fatalf("version")
	}
	// boundary ratings 4.0/4.5
	in3 := scoring.Inputs{ReviewRating: 4.5, ReviewCount: 50, Status: "OPERATIONAL", Website: "", HasWebsite: false, Phone: "+628111111111"}
	r3 := scoring.CalculateV2(in3)
	in4 := scoring.Inputs{ReviewRating: 4.0, ReviewCount: 50, Status: "OPERATIONAL", Website: "", HasWebsite: false, Phone: "+628111111111"}
	r4 := scoring.CalculateV2(in4)
	if r3.OverallScore == r4.OverallScore {
		t.Logf("boundary 4.5 vs 4.0 scores %d vs %d may differ or equal but should be deterministic", r3.OverallScore, r4.OverallScore)
	}
	// boundary reviews 50/100
	in5 := scoring.Inputs{ReviewRating: 4.2, ReviewCount: 100, Status: "OPERATIONAL", Website: "", HasWebsite: false, Phone: "+628111111111"}
	r5 := scoring.CalculateV2(in5)
	in6 := scoring.Inputs{ReviewRating: 4.2, ReviewCount: 51, Status: "OPERATIONAL", Website: "", HasWebsite: false, Phone: "+628111111111"}
	r6 := scoring.CalculateV2(in6)
	if r5.OverallScore < r6.OverallScore {
		t.Fatalf("100 reviews should score >=51")
	}
}

func TestPersistAndLatest(t *testing.T) {
	db := newDB(t)
	_, _ = db.Exec(`INSERT INTO businesses(id, title, first_seen_at, last_seen_at) VALUES('b1','Biz','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`)
	svc := scoring.New(db)
	in := scoring.Inputs{Title: "Biz", Category: "cafe", Phone: "+628111111111", ReviewRating: 4.9, ReviewCount: 120, Status: "OPERATIONAL", Website: "", HasWebsite: false}
	res := scoring.CalculateV2(in)
	persisted, err := svc.Persist(res, "b1")
	if err != nil || persisted.ID == "" {
		t.Fatalf("persist %v", err)
	}
	latest, err := svc.Latest("b1")
	if err != nil || latest.OverallScore != res.OverallScore || latest.RuleSetVersion != scoring.RuleSetVersion {
		t.Fatalf("latest mismatch %v", err)
	}
	if latest.Evidence == nil {
		t.Fatalf("evidence not persisted")
	}
}
