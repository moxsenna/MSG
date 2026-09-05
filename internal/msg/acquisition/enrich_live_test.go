package acquisition

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gosom/google-maps-scraper/internal/msg/domain"
)

type collectSink struct{ places []domain.RawPlaceV1 }

func (s *collectSink) OnPlace(ctx context.Context, p domain.RawPlaceV1) error {
	s.places = append(s.places, p)
	return nil
}

func TestDirectScrapeEnrichesPhoneLive(t *testing.T) {
	if testing.Short() {
		t.Skip("needs live Google Maps")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	sink := &collectSink{}
	req := SearchRequest{Query: "rental mobil", LocationText: "cirebon", RadiusMeters: 10000, SpeedPreset: "balanced"}
	w := &rawPlaceWriter{sink: sink}
	if err := directScrapeFallback(ctx, req, w, nil); err != nil {
		t.Fatalf("direct scrape failed: %v", err)
	}
	if len(sink.places) == 0 {
		t.Fatal("no places")
	}
	phones, websites := 0, 0
	for _, p := range sink.places {
		if p.Phone != "" {
			phones++
		}
		if p.Website != "" {
			websites++
		}
		t.Logf("%s | phone=%q web=%q rating=%.1f reviews=%d", p.Title, p.Phone, p.Website, p.ReviewRating, p.ReviewCount)
	}
	if phones == 0 {
		t.Errorf("expected at least 1 phone enriched, got 0/%d", len(sink.places))
	}
	t.Logf("enriched phones=%d/%d websites=%d/%d", phones, len(sink.places), websites, len(sink.places))
}

func TestDirectScrapeHonorsTargetAndAddress(t *testing.T) {
	if testing.Short() {
		t.Skip("needs live Google Maps")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	sink := &collectSink{}
	req := SearchRequest{Query: "rental mobil", LocationText: "cirebon", RadiusMeters: 10000, SpeedPreset: "balanced", TargetCount: 3}
	w := &rawPlaceWriter{sink: sink}
	if err := directScrapeFallback(ctx, req, w, nil); err != nil {
		t.Fatalf("direct scrape failed: %v", err)
	}
	if len(sink.places) == 0 || len(sink.places) > 3 {
		t.Fatalf("expected 1..3 places for target=3, got %d", len(sink.places))
	}
	for _, p := range sink.places {
		if p.Link == "" || !strings.Contains(p.Link, "/maps/") {
			t.Errorf("missing maps link for %q", p.Title)
		}
		if p.Address == "" {
			t.Errorf("missing address for %q", p.Title)
		}
		t.Logf("%s | addr=%q link=%.60s... socials=%v", p.Title, p.Address, p.Link, p.Socials)
	}
}
func TestParseCoordsAndHaversine(t *testing.T) {
	lat, lon, ok := parseCoordsFromURL("https://www.google.com/maps/place/X/@-6.7320,108.5500,15z/data=!3m1!4b1")
	if !ok || lat != -6.732 || lon != 108.55 {
		t.Errorf("at-pattern: got %v,%v,%v", lat, lon, ok)
	}
	lat, lon, ok = parseCoordsFromURL("https://www.google.com/maps/place/X/data=!4m7!3m6!1s0!2s1!3d-6.732!4d108.55!16s")
	if !ok || lat != -6.732 || lon != 108.55 {
		t.Errorf("data-pattern: got %v,%v,%v", lat, lon, ok)
	}
	if _, _, ok := parseCoordsFromURL("https://www.google.com/maps/search/foo"); ok {
		t.Error("expected no coords")
	}
	if d := haversineMeters(-6.732, 108.55, -6.732, 108.55); d > 1 {
		t.Errorf("same point should be ~0, got %v", d)
	}
	d := haversineMeters(-6.732, 108.55, -6.2, 106.8)
	if d < 150000 || d > 250000 {
		t.Errorf("cirebon-jakarta should be ~200km, got %v", d)
	}
	if _, _, ok := parseCenter(" -6.7320,108.5500 "); !ok {
		t.Error("parseCenter failed")
	}
	if _, _, ok := parseCenter("bogus"); ok {
		t.Error("parseCenter should fail")
	}
}
