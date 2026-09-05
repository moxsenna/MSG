package storage_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/gosom/google-maps-scraper/desktop/storage"
	"github.com/gosom/google-maps-scraper/desktop/storage/sqlite"
)

func TestBackupAndExport(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("LOCALAPPDATA", tempDir)

	dbPath := filepath.Join(tempDir, "MSG", "msg.db")
	_ = os.MkdirAll(filepath.Dir(dbPath), 0o755)
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := sqlite.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Insert a sample lead
	_, _ = db.Exec(`INSERT INTO businesses(id, title, first_seen_at, last_seen_at) VALUES('b_exp', 'Export Corp', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`)
	_, _ = db.Exec(`INSERT INTO lead_states(business_id, stage, do_not_contact) VALUES('b_exp', 'qualified', 0)`)
	_ = db.Close()

	// 1. Test Backup ZIP
	zipPath, err := storage.Backup(dbPath)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("zip file not found: %s", zipPath)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer zr.Close()
	if len(zr.File) == 0 {
		t.Fatalf("zip is empty")
	}

	// Reopen DB for export tests
	db2, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer db2.Close()

	// 2. Test Export CSV
	csvPath := filepath.Join(tempDir, "leads.csv")
	if err := storage.ExportCSV(db2, csvPath); err != nil {
		t.Fatalf("export csv: %v", err)
	}
	content, _ := os.ReadFile(csvPath)
	if len(content) == 0 {
		t.Fatalf("csv export empty")
	}

	// 3. Test Export JSON
	jsonPath := filepath.Join(tempDir, "leads.json")
	if err := storage.ExportJSON(db2, jsonPath); err != nil {
		t.Fatalf("export json: %v", err)
	}
	jsonContent, _ := os.ReadFile(jsonPath)
	if len(jsonContent) == 0 {
		t.Fatalf("json export empty")
	}
}
