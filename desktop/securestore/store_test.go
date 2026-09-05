package securestore_test

import (
	"testing"

	"github.com/gosom/google-maps-scraper/desktop/securestore"
)

func TestMemoryStore_RoundTrip(t *testing.T) {
	s := securestore.NewMemory()
	if err := s.Set("ai_key", "sk-12345"); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := s.Get("ai_key")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != "sk-12345" {
		t.Fatalf("got %q", got)
	}
	if err := s.Delete("ai_key"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Get("ai_key"); err != securestore.ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestRedact(t *testing.T) {
	if securestore.Redact("") != "" {
		t.Fatal("empty")
	}
	if securestore.Redact("ab") != "****" {
		t.Fatal("short")
	}
	if got := securestore.Redact("sk-1234567890"); got == "sk-1234567890" {
		t.Fatal("not redacted")
	}
}
