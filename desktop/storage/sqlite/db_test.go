package sqlite_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/gosom/google-maps-scraper/desktop/storage/sqlite"
)

func TestMigrate_CleanDB(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := sqlite.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// idempotent second run
	if err := sqlite.Migrate(db); err != nil {
		t.Fatalf("migrate second: %v", err)
	}
	var cnt int
	if err := db.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&cnt); err != nil {
		t.Fatalf("count: %v", err)
	}
	if cnt == 0 {
		t.Fatal("no migrations recorded")
	}
	// spot check tables exist
	for _, tbl := range []string{"businesses", "search_runs", "source_snapshots", "lead_states", "settings"} {
		var name string
		if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tbl).Scan(&name); err != nil {
			t.Fatalf("table %s missing: %v", tbl, err)
		}
	}
}

func TestSQLite_Pragmas(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	var fk string
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil {
		t.Fatalf("fk pragma: %v", err)
	}
	// foreign_keys pragma returns 0/1 as integer string via sqlite driver may vary; just check query succeeds
	var busy int
	if err := db.QueryRow(`PRAGMA busy_timeout`).Scan(&busy); err != nil {
		t.Fatalf("busy: %v", err)
	}
	if busy != 5000 {
		t.Fatalf("busy_timeout=%d want 5000", busy)
	}
}

func TestSQLite_ForeignKeyEnforced(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := sqlite.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Insert lead_state without business should fail FK
	_, err = db.Exec(`INSERT INTO lead_states(business_id, stage) VALUES('missing','new')`)
	if err == nil {
		t.Fatal("expected FK error")
	}
}

func TestSQLite_PersistAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "persist.db")
	db, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := sqlite.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO businesses(id, title, first_seen_at, last_seen_at) VALUES('b1','Test Biz','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO settings(key,value) VALUES('theme','dark')`); err != nil {
		t.Fatalf("settings insert: %v", err)
	}
	_ = db.Close()

	db2, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()
	var title string
	if err := db2.QueryRow(`SELECT title FROM businesses WHERE id='b1'`).Scan(&title); err != nil {
		t.Fatalf("select: %v", err)
	}
	if title != "Test Biz" {
		t.Fatalf("title=%q", title)
	}
	var theme string
	if err := db2.QueryRow(`SELECT value FROM settings WHERE key='theme'`).Scan(&theme); err != nil {
		t.Fatalf("theme: %v", err)
	}
	if theme != "dark" {
		t.Fatalf("theme=%q", theme)
	}
}

func TestSQLite_TransactionRollback(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := sqlite.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := tx.Exec(`INSERT INTO businesses(id, title, first_seen_at, last_seen_at) VALUES('tx1','Tx Biz','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("tx insert: %v", err)
	}
	_ = tx.Rollback()
	var cnt int
	_ = db.QueryRow(`SELECT count(*) FROM businesses WHERE id='tx1'`).Scan(&cnt)
	if cnt != 0 {
		t.Fatal("rollback failed")
	}
}

var _ = sql.ErrNoRows // ensure import used
