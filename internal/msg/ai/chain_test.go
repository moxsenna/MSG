package ai

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type stubProvider struct {
	name string
	fail error
	text string
}

func (s *stubProvider) Name() string { return s.name }

func (s *stubProvider) AnalyzeOpportunity(ctx context.Context, b, c, ci, w string) (*OpportunityAnalysis, error) {
	if s.fail != nil {
		return nil, s.fail
	}
	return &OpportunityAnalysis{Summary: s.text, Confidence: 1}, nil
}

func (s *stubProvider) DraftOutreach(ctx context.Context, b, c, o, ch, t string) (string, error) {
	if s.fail != nil {
		return "", s.fail
	}
	return s.text, nil
}

func TestChainFallsBackToSecondKey(t *testing.T) {
	quota := &QuotaError{Provider: "gemini", Status: 429, Message: "quota exceeded"}
	chain := NewChain(func(sl KeySlot) (Provider, error) {
		if sl.ID == "k1" {
			return &stubProvider{name: "gemini", fail: quota}, nil
		}
		return &stubProvider{name: "gemini", text: "ok-dari-akun-2"}, nil
	})
	slots := []KeySlot{
		{ID: "k1", Provider: "gemini", Label: "akun 1", Enabled: true, Priority: 0},
		{ID: "k2", Provider: "gemini", Label: "akun 2", Enabled: true, Priority: 1},
	}
	text, used, err := chain.DraftOutreach(context.Background(), slots, "B", "C", "O", "whatsapp", "friendly")
	if err != nil {
		t.Fatalf("expected fallback success, got %v", err)
	}
	if used != "k2" || text != "ok-dari-akun-2" {
		t.Errorf("expected k2 success, got used=%q text=%q", used, text)
	}
}

func TestChainSkipsDisabledAndCooldown(t *testing.T) {
	chain := NewChain(func(sl KeySlot) (Provider, error) {
		if sl.ID == "k1" {
			return &stubProvider{name: "gemini", fail: fmt.Errorf("boom")}, nil
		}
		return &stubProvider{name: "gemini", text: "ok"}, nil
	})
	slots := []KeySlot{
		{ID: "k0", Provider: "gemini", Label: "mati", Enabled: false, Priority: 0},
		{ID: "k1", Provider: "gemini", Label: "akun 1", Enabled: true, Priority: 1},
		{ID: "k2", Provider: "gemini", Label: "akun 2", Enabled: true, Priority: 2},
	}
	if _, used, err := chain.DraftOutreach(context.Background(), slots, "B", "C", "O", "wa", "f"); err != nil || used != "k2" {
		t.Fatalf("expected k2, got used=%q err=%v", used, err)
	}
	if _, used, err := chain.DraftOutreach(context.Background(), slots, "B", "C", "O", "wa", "f"); err != nil || used != "k2" {
		t.Fatalf("expected k2 again (k1 in cooldown, k2 healthy), got used=%q err=%v", used, err)
	}
}

func TestChainNoKeysHonestError(t *testing.T) {
	chain := NewChain(func(sl KeySlot) (Provider, error) { return &stubProvider{}, nil })
	_, _, err := chain.DraftOutreach(context.Background(), nil, "B", "C", "O", "wa", "f")
	if err == nil || !strings.Contains(err.Error(), "no enabled AI key") {
		t.Fatalf("expected honest no-key error, got %v", err)
	}
}

func TestSlotValidation(t *testing.T) {
	if ValidProvider("grok") {
		t.Error("grok should be invalid")
	}
	if !ValidProvider("GEMINI") {
		t.Error("GEMINI should be valid case-insensitive")
	}
	if KeyHint("123") != "****" || KeyHint("AIzaSy1234567890") != "****7890" {
		t.Error("KeyHint masking wrong")
	}
}
