package storage

import (
	"os"
	"path/filepath"
)

// AppDataDir returns the OS-appropriate app data directory for MSG Desktop.
// On Windows it prefers %LOCALAPPDATA%/MSG, otherwise falls back to user cache dir.
// It ensures the directory and standard subdirectories exist.
func AppDataDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		dir, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		base = dir
	}
	root := filepath.Join(base, "MSG")
	subs := []string{
		root,
		filepath.Join(root, "screenshots"),
		filepath.Join(root, "exports"),
		filepath.Join(root, "backups"),
		filepath.Join(root, "logs"),
		filepath.Join(root, "cache"),
	}
	for _, p := range subs {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return "", err
		}
	}
	return root, nil
}

// DBPath returns the canonical SQLite path under AppDataDir.
func DBPath() (string, error) {
	dir, err := AppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "msg.db"), nil
}
