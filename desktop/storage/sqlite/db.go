package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open opens a SQLite database at path with Desktop pragmas.
// path may be ":memory:" for tests.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)", path)
	// For :memory: the file: URI needs special handling
	if path == ":memory:" {
		dsn = "file::memory:?cache=shared&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	// Verify pragmas by explicit exec for modernc driver
	pragmas := []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA foreign_keys=ON`,
		`PRAGMA busy_timeout=5000`,
		`PRAGMA synchronous=NORMAL`,
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			// WAL on :memory: may return 'memory' journal_mode — not fatal for tests
			if p == `PRAGMA journal_mode=WAL` && path == ":memory:" {
				continue
			}
			_ = db.Close()
			return nil, fmt.Errorf("pragma %s: %w", p, err)
		}
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return db, nil
}
