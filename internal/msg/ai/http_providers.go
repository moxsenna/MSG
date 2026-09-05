package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 25 * time.Second}

type QuotaError struct {
	Provider string
	Status   int
	Message  string
}

func (e *QuotaError) Error() string {
	return fmt.Sprintf("%s quota/rate-limited (http %d): %s", e.Provider, e.Status, e.Message)
}

func isRetryableStatus(code int) bool {
	return code == 429 || code == 500 || code == 502 || code == 503 || code == 504
}

func readBodySnippet(r io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(r, 512))
	return strings.TrimSpace(string(b))
}

type geminiProvider struct {
	key   string
	model string
}

func newGemini(key, model string) *geminiProvider {
	if strings.TrimSpace(model) == "" {
		model = "gemini-2.0-flash"
	}
	return &geminiProvider{key: strings.TrimSpace(key), model: strings.TrimSpace(model)}
}

func (g *geminiProvider) Name() string { return ProviderGemini }

func (g *geminiProvider) call(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.key)
	payload, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{{"parts": []map[string]any{{"text": prompt}}}},
		"generationConfig": map[string]any{"maxOutputTokens": 512, "temperature": 0.3},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini network: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return "", fmt.Errorf("gemini unauthorized: invalid API key (%d)", resp.StatusCode)
	}
	if isRetryableStatus(resp.StatusCode) {
		return "", &QuotaError{Provider: ProviderGemini, Status: resp.StatusCode, Message: readBodySnippet(resp.Body)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("gemini error http %d: %s", resp.StatusCode, readBodySnippet(resp.Body))
	}
	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("gemini decode: %w", err)
	}
	var sb strings.Builder
	for _, c := range out.Candidates {
		for _, p := range c.Content.Parts {
			sb.WriteString(p.Text)
		}
	}
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return "", fmt.Errorf("gemini empty response")
	}
	return text, nil
}

func (g *geminiProvider) AnalyzeOpportunity(ctx context.Context, businessName, category, city, websiteInfo string) (*OpportunityAnalysis, error) {
	prompt := fmt.Sprintf("Analisa peluang digital untuk bisnis %q kategori %q kota %q info %q. Jawab ringkas: ringkasan|tipe_peluang|angle|penawaran|alasan1;alasan2. Tipe salah satu: website, booking_or_appointment, ecommerce, ads.",
		businessName, category, city, websiteInfo)
	text, err := g.call(ctx, prompt)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(text, "|", 5)
	for len(parts) < 5 {
		parts = append(parts, "")
	}
	return &OpportunityAnalysis{
		Summary: strings.TrimSpace(parts[0]), OpportunityType: strings.TrimSpace(parts[1]),
		SalesAngle: strings.TrimSpace(parts[2]), SuggestedOffer: strings.TrimSpace(parts[3]),
		Reasons: strings.Split(strings.TrimSpace(parts[4]), ";"), Confidence: 0.7,
	}, nil
}

func (g *geminiProvider) DraftOutreach(ctx context.Context, businessName, category, offer, channel, tone string) (string, error) {
	return g.call(ctx, fmt.Sprintf("Tulis draft outreach %s %s untuk %q (%s), penawaran %q, maks 3 kalimat.", channel, tone, businessName, category, offer))
}

type openAICompatibleProvider struct {
	name    string
	key     string
	baseURL string
	model   string
}

func newOpenAICompatible(name, key, baseURL, model string) *openAICompatibleProvider {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if strings.TrimSpace(model) == "" {
		if name == ProviderOpenAI {
			model = "gpt-4o-mini"
		} else {
			model = "default"
		}
	}
	return &openAICompatibleProvider{name: name, key: strings.TrimSpace(key), baseURL: baseURL, model: strings.TrimSpace(model)}
}

func (o *openAICompatibleProvider) Name() string { return o.name }

func (o *openAICompatibleProvider) call(ctx context.Context, system, user string) (string, error) {
	payload, _ := json.Marshal(map[string]any{
		"model": o.model,
		"messages": []map[string]any{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"max_tokens": 512, "temperature": 0.3,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.key)
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s network: %w", o.name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return "", fmt.Errorf("%s unauthorized: invalid API key (%d)", o.name, resp.StatusCode)
	}
	if isRetryableStatus(resp.StatusCode) {
		return "", &QuotaError{Provider: o.name, Status: resp.StatusCode, Message: readBodySnippet(resp.Body)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("%s error http %d: %s", o.name, resp.StatusCode, readBodySnippet(resp.Body))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("%s decode: %w", o.name, err)
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("%s empty response", o.name)
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}

func (o *openAICompatibleProvider) AnalyzeOpportunity(ctx context.Context, businessName, category, city, websiteInfo string) (*OpportunityAnalysis, error) {
	text, err := o.call(ctx, "Jawab HANYA dalam format: ringkasan|tipe|angle|penawaran|alasan1;alasan2",
		fmt.Sprintf("Analisa peluang digital untuk %q (%s) kota %q info %q", businessName, category, city, websiteInfo))
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(text, "|", 5)
	for len(parts) < 5 {
		parts = append(parts, "")
	}
	return &OpportunityAnalysis{
		Summary: strings.TrimSpace(parts[0]), OpportunityType: strings.TrimSpace(parts[1]),
		SalesAngle: strings.TrimSpace(parts[2]), SuggestedOffer: strings.TrimSpace(parts[3]),
		Reasons: strings.Split(strings.TrimSpace(parts[4]), ";"), Confidence: 0.7,
	}, nil
}

func (o *openAICompatibleProvider) DraftOutreach(ctx context.Context, businessName, category, offer, channel, tone string) (string, error) {
	return o.call(ctx, "Penulis outreach singkat Bahasa Indonesia.",
		fmt.Sprintf("Draft outreach %s %s untuk %q (%s), penawaran %q, maks 3 kalimat.", channel, tone, businessName, category, offer))
}

func BuildProvider(slot KeySlot, key string) (Provider, error) {
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("empty key for slot %q", slot.ID)
	}
	switch slot.Provider {
	case ProviderGemini:
		return newGemini(key, slot.Model), nil
	case ProviderOpenAI:
		return newOpenAICompatible(ProviderOpenAI, key, "", slot.Model), nil
	case ProviderCustom:
		return newOpenAICompatible(ProviderCustom, key, slot.BaseURL, slot.Model), nil
	}
	return nil, fmt.Errorf("unknown provider %q", slot.Provider)
}
