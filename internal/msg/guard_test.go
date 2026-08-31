package msg_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoDockerInvocationInProductCode(t *testing.T) {
	productRoots := []string{"."}

	forbidden := "gosom/google-maps-scraper"
	alt := "docker run"

	for _, pr := range productRoots {
		err := filepath.Walk(pr, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			if strings.HasSuffix(path, "guard_test.go") {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			content := string(b)
			if strings.Contains(content, forbidden) && strings.Contains(content, alt) {
				t.Fatalf("forbidden docker invocation found in product code %s: contains %q and %q", path, alt, forbidden)
			}
			if strings.Contains(content, "docker run") && strings.Contains(content, "google-maps-scraper") {
				t.Fatalf("forbidden docker run google-maps-scraper in %s", path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk failed: %v", err)
		}
	}
}
