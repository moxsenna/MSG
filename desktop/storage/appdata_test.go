package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gosom/google-maps-scraper/desktop/storage"
)

func TestAppDataDir_CreatesSubdirs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOCALAPPDATA", dir)
	root, err := storage.AppDataDir()
	if err != nil {
		t.Fatalf("AppDataDir: %v", err)
	}
	if filepath.Base(root) != "MSG" {
		t.Fatalf("root=%q", root)
	}
	for _, sub := range []string{"screenshots", "exports", "backups", "logs", "cache"} {
		p := filepath.Join(root, sub)
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing %s: %v", sub, err)
		}
	}
	dbp, err := storage.DBPath()
	if err != nil {
		t.Fatalf("DBPath: %v", err)
	}
	if filepath.Base(dbp) != "msg.db" {
		t.Fatalf("dbp=%q", dbp)
	}
}
