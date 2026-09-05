package ai

import (
	"context"
	"fmt"
	"strings"
)

// OpportunityAnalysis is the structured output from AI opportunity reasoning.
type OpportunityAnalysis struct {
	Summary        string   `json:"summary"`
	OpportunityType string  `json:"opportunity_type"`
	SalesAngle     string   `json:"sales_angle"`
	SuggestedOffer string   `json:"suggested_offer"`
	Reasons        []string `json:"reasons"`
	Confidence     float64  `json:"confidence"`
}

// Provider defines the pluggable LLM interface (Gemini / OpenAI / Mock).
// Crucial: AI is optional and structured; failures must never block pipeline.
type Provider interface {
	Name() string
	AnalyzeOpportunity(ctx context.Context, businessName, category, city, websiteInfo string) (*OpportunityAnalysis, error)
	DraftOutreach(ctx context.Context, businessName, category, offer, channel, tone string) (string, error)
}

// FakeProvider provides offline deterministic responses without external API keys.
type FakeProvider struct{}

func NewFakeProvider() *FakeProvider { return &FakeProvider{} }

func (f *FakeProvider) Name() string { return "offline-mock" }

func (f *FakeProvider) AnalyzeOpportunity(ctx context.Context, businessName, category, city, websiteInfo string) (*OpportunityAnalysis, error) {
	summary := fmt.Sprintf("Bisnis %s di %s berpotensi tinggi untuk modernisasi digital.", businessName, city)
	oppType := "website"
	if strings.Contains(strings.ToLower(category), "restaurant") || strings.Contains(strings.ToLower(category), "cafe") {
		oppType = "booking_or_appointment"
	}
	return &OpportunityAnalysis{
		Summary:        summary,
		OpportunityType: oppType,
		SalesAngle:     "Tingkatkan konversi pelanggan lokal melalui landing page cepat dan integrasi WhatsApp.",
		SuggestedOffer: "Paket Website Profesional + Setup WhatsApp Order",
		Reasons:        []string{"Belum memiliki profil online teroptimasi", "Pasar lokal kompetitif"},
		Confidence:     0.85,
	}, nil
}

func (f *FakeProvider) DraftOutreach(ctx context.Context, businessName, category, offer, channel, tone string) (string, error) {
	if channel == "whatsapp" {
		return fmt.Sprintf("Halo kak dari %s, kami melihat potensi menarik untuk tingkatkan pesanan %s lewat website modern. Mau kami buatkan preview gratis?", businessName, category), nil
	}
	return fmt.Sprintf("Yth. Manajemen %s,\n\nKami menawarkan solusi digital %s untuk meningkatkan efisiensi dan jangkauan bisnis Anda.\n\nSalam,\nTim MSG", businessName, offer), nil
}
