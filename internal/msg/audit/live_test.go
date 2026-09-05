package audit

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`CREATE TABLE website_audits (
		id TEXT PRIMARY KEY, business_id TEXT NOT NULL, audit_version TEXT NOT NULL DEFAULT 'v1',
		requested_url TEXT NOT NULL, final_url TEXT, status TEXT NOT NULL DEFAULT 'pending',
		evidence_json TEXT NOT NULL DEFAULT '{}', screenshot_path TEXT, duration_ms INTEGER,
		error_code TEXT, audited_at TEXT NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.Getenv
	return db
}

func TestAuditRealWebsiteLive(t *testing.T) {
	if testing.Short() {
		t.Skip("needs live network + browser")
	}
	db := openTestDB(t)
	svc := New(db)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	res, err := svc.Run(ctx, "biz-web", "http://rentgoindonesia.com/")
	if err != nil {
		t.Fatalf("audit failed: %v", err)
	}
	t.Logf("reachable=%v https=%v contact=%v wa=%v form=%v title=%q final=%q",
		boolVal(res.Evidence.Reachable), boolVal(res.Evidence.HTTPS),
		boolVal(res.Evidence.HasContactInfo), boolVal(res.Evidence.HasWhatsApp),
		boolVal(res.Evidence.HasForm), res.Evidence.Title, res.Evidence.FinalURL)
	if res.Evidence.Reachable == nil || !*res.Evidence.Reachable {
		t.Error("expected reachable=true for a real site")
	}
}

func TestAuditBlocksLocalhost(t *testing.T) {
	db := openTestDB(t)
	svc := New(db)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	res, err := svc.Run(ctx, "biz-local", "http://localhost:8080/admin")
	if err != nil {
		t.Fatalf("persist failed: %v", err)
	}
	if res.Status != StatusFailed || res.Evidence.ErrorCode == "" {
		t.Fatalf("expected failed+ErrorCode, got status=%q code=%q", res.Status, res.Evidence.ErrorCode)
	}
	t.Logf("blocked honestly: status=%s code=%s", res.Status, res.Evidence.ErrorCode)
}

func boolVal(b *bool) bool { return b != nil && *b }
