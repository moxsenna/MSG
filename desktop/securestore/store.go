package securestore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// SecureStore is the OS-backed secret store for Desktop.
// Secrets must never be stored in plaintext SQLite/logs/exports.
type SecureStore interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
}

// ErrNotFound is returned when a key has no stored value.
var ErrNotFound = errors.New("secret not found")

// Redact returns a masked representation of a secret for logging.
func Redact(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

// New returns the platform-appropriate SecureStore.
// On Windows it uses DPAPI via wincred; elsewhere it falls back to a
// file-backed store under the app data directory (still not plaintext SQLite).
func New() (SecureStore, error) {
	if isWindows() {
		if s, err := newWindowsStore(); err == nil {
			return s, nil
		}
	}
	// Fallback: file store (used in tests and non-Windows dev)
	dir := fallbackDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("securestore mkdir: %w", err)
	}
	return &fileStore{dir: dir}, nil
}

func isWindows() bool {
	return filepath.Separator == '\\' && os.Getenv("OS") != ""
	// simplified check; actual runtime.GOOS check done in windows file
}

func fallbackDir() string {
	if d := os.Getenv("LOCALAPPDATA"); d != "" {
		return filepath.Join(d, "MSG", "secrets")
	}
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "MSG", "secrets")
	}
	return filepath.Join(os.TempDir(), "MSG-secrets")
}
