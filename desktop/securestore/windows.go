//go:build windows

package securestore

import (
	"fmt"
	"os"
	"runtime"
)

// windowsStore wraps fileStore but is selected on Windows.
// Full DPAPI (CryptProtectData) via x/sys/windows would be added here.
// For V0.1 we use the file store path under %LOCALAPPDATA%/MSG/secrets
// which is already OS ACL-protected, and document the upgrade path.

type windowsStore struct{ *fileStore }

func newWindowsStore() (SecureStore, error) {
	if runtime.GOOS != "windows" {
		return nil, fmt.Errorf("not windows")
	}
	dir := fallbackDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("securestore mkdir: %w", err)
	}
	// Defer actual DPAPI wrapping until dependency is added to go.mod to avoid
	// build breakage on non-Windows CI. The interface is already correct.
	return &windowsStore{fileStore: &fileStore{dir: dir}}, nil
}

func (w *windowsStore) Get(key string) (string, error) { return w.fileStore.Get(key) }
func (w *windowsStore) Set(key, value string) error    { return w.fileStore.Set(key, value) }
func (w *windowsStore) Delete(key string) error         { return w.fileStore.Delete(key) }
