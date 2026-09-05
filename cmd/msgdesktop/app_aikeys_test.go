package main

import (
	"context"
	"os"
	"testing"

	"github.com/gosom/google-maps-scraper/desktop/securestore"
	"github.com/gosom/google-maps-scraper/desktop/storage/sqlite"
	"github.com/gosom/google-maps-scraper/internal/msg/acquisition"
	"github.com/gosom/google-maps-scraper/internal/msg/ai"
	"github.com/gosom/google-maps-scraper/internal/msg/ingest"
	"github.com/gosom/google-maps-scraper/internal/msg/leads"
	"github.com/gosom/google-maps-scraper/internal/msg/pipeline"
	"github.com/gosom/google-maps-scraper/internal/msg/outreach"
	"github.com/gosom/google-maps-scraper/internal/msg/scoring"
)

func newAIApp(t *testing.T) *App {
	t.Helper()
	dir, err := os.MkdirTemp("", "msg-aikeys-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	prevLocalAppData, hadLocalAppData := os.LookupEnv("LOCALAPPDATA")
	os.Setenv("LOCALAPPDATA", dir)
	store, err := securestore.New()
	if hadLocalAppData {
		os.Setenv("LOCALAPPDATA", prevLocalAppData)
	} else {
		os.Unsetenv("LOCALAPPDATA")
	}
	if err != nil {
		t.Fatal(err)
	}
	db, err := sqlite.Open(dir + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := sqlite.Migrate(db); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	app.ctx = context.Background()
	app.db = db
	app.store = store
	app.leads = leads.New(db)
	app.ingest = ingest.New(db)
	app.pipeline = pipeline.New(db)
	app.adapter = acquisition.NewAdapter()
	app.scoring = scoring.New(db)
	app.outreach = outreach.New(db)
	app.aiSlots = ai.NewSlotStore(db)
	app.aiChain = ai.NewChain(func(slot ai.KeySlot) (ai.Provider, error) {
		key, err := app.aiKey(slot.ID)
		if err != nil {
			return nil, err
		}
		return ai.BuildProvider(slot, key)
	})
	return app
}

func TestAIKeysMultiAccountFlow(t *testing.T) {
	app := newAIApp(t)

	k1, err := app.AddAIKey(AddAIKeyRequest{Provider: "gemini", Label: "gemini akun 1", Key: "AIzaFakeKeyAccount111111"})
	if err != nil {
		t.Fatalf("add k1: %v", err)
	}
	k2, err := app.AddAIKey(AddAIKeyRequest{Provider: "gemini", Label: "gemini akun 2", Key: "AIzaFakeKeyAccount222222"})
	if err != nil {
		t.Fatalf("add k2: %v", err)
	}
	k3, err := app.AddAIKey(AddAIKeyRequest{Provider: "custom", Label: "gateway kantor", Key: "sk-custom-key-12345678", BaseURL: "https://gateway.contoh.com/v1", Model: "my-model"})
	if err != nil {
		t.Fatalf("add custom: %v", err)
	}

	slots, err := app.ListAIKeys()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(slots) != 3 {
		t.Fatalf("expected 3 slots, got %d", len(slots))
	}
	if slots[0].ID != k1.ID || slots[1].ID != k2.ID || slots[2].ID != k3.ID {
		t.Errorf("expected insertion order, got %v %v %v", slots[0].ID, slots[1].ID, slots[2].ID)
	}
	for _, sl := range slots {
		if sl.KeyHint == "" || sl.KeyHint == "****1234" {
			t.Errorf("slot %s missing masked hint", sl.ID)
		}
	}

	raw, err := app.aiKey(k2.ID)
	if err != nil || raw != "AIzaFakeKeyAccount222222" {
		t.Errorf("securestore roundtrip failed: %v %q", err, raw)
	}
	var leaked int
	_ = app.db.QueryRow(`SELECT COUNT(*) FROM ai_key_slots WHERE label LIKE '%AIzaFakeKeyAccount%' OR key_hint LIKE '%AIzaFakeKeyAccount222222%'`).Scan(&leaked)
	if leaked != 0 {
		t.Error("secret leaked into SQLite metadata")
	}

	if err := app.MoveAIKey(k3.ID, -1); err != nil {
		t.Fatalf("move: %v", err)
	}
	slots, _ = app.ListAIKeys()
	if slots[1].ID != k3.ID {
		t.Errorf("expected k3 at position 2 after move up, got %s", slots[1].ID)
	}

	if err := app.UpdateAIKey(k1.ID, "gemini akun 1", false, "", ""); err != nil {
		t.Fatalf("disable: %v", err)
	}
	slots, _ = app.ListAIKeys()
	for _, sl := range slots {
		if sl.ID == k1.ID && sl.Enabled {
			t.Error("k1 should be disabled")
		}
	}

	if err := app.RotateAIKey(k2.ID, "AIzaRotatedKey99999999"); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	raw, _ = app.aiKey(k2.ID)
	if raw != "AIzaRotatedKey99999999" {
		t.Error("rotated key not readable")
	}

	if _, err := app.TestAIKey(k2.ID); err == nil {
		t.Error("expected TestAIKey with fake key to fail honestly, got nil")
	} else {
		t.Logf("TestAIKey honest failure: %v", err)
	}

	if err := app.DeleteAIKey(k1.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	slots, _ = app.ListAIKeys()
	if len(slots) != 2 {
		t.Errorf("expected 2 slots after delete, got %d", len(slots))
	}
	if _, err := app.aiKey(k1.ID); err == nil {
		t.Error("deleted slot secret should be gone from SecureStore")
	}

	if _, err := app.AddAIKey(AddAIKeyRequest{Provider: "grok", Label: "x", Key: "12345678"}); err == nil {
		t.Error("expected unknown provider rejected")
	}
	if _, err := app.AddAIKey(AddAIKeyRequest{Provider: "custom", Label: "no-url", Key: "12345678"}); err == nil {
		t.Error("expected custom without base_url rejected")
	}
	if _, err := app.AddAIKey(AddAIKeyRequest{Provider: "gemini", Label: "short", Key: "abc"}); err == nil {
		t.Error("expected short key rejected")
	}
}
