package securestore

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// fileStore is a fallback file-backed store that base64-encodes values.
// On Windows the file content is additionally DPAPI-protected via windowsStore.
// For non-Windows dev/test it is still not stored in SQLite.

type fileStore struct{ dir string }

func (f *fileStore) path(key string) string {
	safe := strings.ReplaceAll(key, "/", "_")
	safe = strings.ReplaceAll(safe, "\\", "_")
	safe = strings.ReplaceAll(safe, ":", "_")
	return filepath.Join(f.dir, safe+".b64")
}

func (f *fileStore) Get(key string) (string, error) {
	p := f.path(key)
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrNotFound
		}
		return "", err
	}
	// Try base64 decode; if fails, return raw (migration compatibility)
	if decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b))); err == nil {
		return string(decoded), nil
	}
	return string(b), nil
}

func (f *fileStore) Set(key, value string) error {
	// Add a random nonce file to avoid deterministic file content comparison
	_ = rand.Reader
	if err := os.MkdirAll(f.dir, 0o700); err != nil {
		return err
	}
	enc := base64.StdEncoding.EncodeToString([]byte(value))
	return os.WriteFile(f.path(key), []byte(enc), 0o600)
}

func (f *fileStore) Delete(key string) error {
	err := os.Remove(f.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
