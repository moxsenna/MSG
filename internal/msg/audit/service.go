package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/playwright-community/playwright-go"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// Evidence is the structured audit output stored as JSON.
type Evidence struct {
	Reachable      *bool    `json:"reachable,omitempty"`
	HTTPS          *bool    `json:"https,omitempty"`
	HasContactInfo *bool    `json:"has_contact_info,omitempty"`
	HasWhatsApp    *bool    `json:"has_whatsapp,omitempty"`
	HasForm        *bool    `json:"has_form,omitempty"`
	HasBooking     *bool    `json:"has_booking,omitempty"`
	SocialLinks    []string `json:"social_links,omitempty"`
	Title          string   `json:"title,omitempty"`
	MetaDesc       string   `json:"meta_desc,omitempty"`
	FinalURL       string   `json:"final_url,omitempty"`
	ErrorCode      string   `json:"error_code,omitempty"`
}

// Result is what gets persisted to website_audits.
type Result struct {
	ID           string
	BusinessID   string
	Version      string
	RequestedURL string
	FinalURL     string
	Status       Status
	Evidence     Evidence
	DurationMs   int64
	ErrorCode    string
	AuditedAt    time.Time
}

type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

// ValidateURL blocks localhost/private networks and unsafe schemes.
// This is the security gate required by ADR-017.
func ValidateURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("AUDIT_INVALID_URL: empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("AUDIT_INVALID_URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("AUDIT_INVALID_URL: only http/https allowed, got %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("AUDIT_INVALID_URL: missing host")
	}
	// block localhost
	lower := strings.ToLower(host)
	if lower == "localhost" || lower == "127.0.0.1" || lower == "::1" {
		return fmt.Errorf("AUDIT_PRIVATE_NETWORK_BLOCKED: localhost")
	}
	ips, err := net.LookupIP(host)
	if err == nil {
		for _, ip := range ips {
			if isPrivateIP(ip) {
				return fmt.Errorf("AUDIT_PRIVATE_NETWORK_BLOCKED: %s resolves to private %s", host, ip.String())
			}
		}
	}
	// also catch literal private IPs without DNS
	if ip := net.ParseIP(host); ip != nil && isPrivateIP(ip) {
		return fmt.Errorf("AUDIT_PRIVATE_NETWORK_BLOCKED: private IP %s", host)
	}
	return nil
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	// also block 0.0.0.0, 169.254.x.x already covered, but add explicit
	if ip.Equal(net.IPv4zero) || ip.Equal(net.IPv4bcast) {
		return true
	}
	// CGNAT 100.64.0.0/10
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
	}
	return false
}

func (s *Service) Run(ctx context.Context, businessID, requestedURL string) (*Result, error) {
	if err := ValidateURL(requestedURL); err != nil {
		return s.persist(businessID, requestedURL, "", StatusFailed, Evidence{ErrorCode: codeFromErr(err)}, err.Error())
	}
	start := time.Now()
	ev, finalURL, err := crawlEvidence(ctx, requestedURL)
	if err != nil {
		ev.ErrorCode = "AUDIT_CRAWL_FAILED"
		res, perr := s.persist(businessID, requestedURL, finalURL, StatusFailed, ev, ev.ErrorCode)
		if perr != nil {
			return nil, perr
		}
		res.DurationMs = time.Since(start).Milliseconds()
		return res, err
	}
	res, err := s.persist(businessID, requestedURL, finalURL, StatusCompleted, ev, "")
	if err != nil {
		return nil, err
	}
	res.DurationMs = time.Since(start).Milliseconds()
	return res, nil
}

func crawlEvidence(ctx context.Context, requestedURL string) (Evidence, string, error) {
	var ev Evidence
	pw, err := playwright.Run()
	if err != nil {
		return ev, "", err
	}
	defer pw.Stop()
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args:     []string{"--no-sandbox", "--disable-dev-shm-usage", "--disable-gpu"},
	})
	if err != nil {
		return ev, "", err
	}
	defer browser.Close()
	bCtx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{Width: 1280, Height: 800},
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/122.0.0.0 Safari/537.36"),
	})
	if err != nil {
		return ev, "", err
	}
	page, err := bCtx.NewPage()
	if err != nil {
		return ev, "", err
	}
	resp, err := page.Goto(requestedURL, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded, Timeout: playwright.Float(20000)})
	if err != nil {
		return ev, "", err
	}
	finalURL := requestedURL
	if resp != nil {
		if u := resp.URL(); u != "" {
			finalURL = u
		}
	}
	if err := ValidateURL(finalURL); err != nil {
		return ev, finalURL, fmt.Errorf("redirected to blocked URL: %w", err)
	}
	https := strings.HasPrefix(strings.ToLower(finalURL), "https://")
	raw, _ := page.Evaluate(`() => {
		try {
			const html = document.documentElement.outerHTML.slice(0, 60000);
			const meta = !!document.querySelector('meta[name="description"]');
			const title = (document.title || "").slice(0, 200);
			const desc = (document.querySelector('meta[name="description"]') || {}).content || "";
			const hasWA = /wa\.me|whatsapp\.com|api\.whatsapp/i.test(html);
			const hasForm = !!document.querySelector("form");
			const hasBooking = /booking|reservasi|janji|appointment|order online|pesan sekarang/i.test(html);
			const links = [];
			document.querySelectorAll('a[href^="http"]').forEach(a => {
				const h = a.getAttribute("href");
				if (h && links.length < 10) links.push(h.slice(0, 200));
			});
			return {html: html, meta: meta, title: title, desc: desc, hasWA: hasWA, hasForm: hasForm, hasBooking: hasBooking, links: links};
		} catch (e) { return {html: "", meta: false, title: "", desc: "", hasWA: false, hasForm: false, hasBooking: false, links: []}; }
	}`)
	m, _ := raw.(map[string]any)
	html, _ := m["html"].(string)
	title, _ := m["title"].(string)
	desc, _ := m["desc"].(string)
	hasWA, _ := m["hasWA"].(bool)
	hasForm, _ := m["hasForm"].(bool)
	hasBooking, _ := m["hasBooking"].(bool)
	social := []string{}
	if links, ok := m["links"].([]any); ok {
		for _, l := range links {
			if s, ok := l.(string); ok {
				ls := strings.ToLower(s)
				if strings.Contains(ls, "instagram.") || strings.Contains(ls, "facebook.") || strings.Contains(ls, "tiktok.") {
					social = append(social, s)
				}
			}
		}
	}
	hasContact := hasWA || hasForm || strings.Contains(strings.ToLower(html), "mailto:") || strings.Contains(strings.ToLower(html), "tel:")
	reachable := true
	return Evidence{
		Reachable: &reachable, HTTPS: &https, HasContactInfo: &hasContact,
		HasWhatsApp: &hasWA, HasForm: &hasForm, HasBooking: &hasBooking,
		SocialLinks: social, Title: title, MetaDesc: desc, FinalURL: finalURL,
	}, finalURL, nil
}

func (s *Service) persist(businessID, requestedURL, finalURL string, status Status, ev Evidence, errorCode string) (*Result, error) {
	id := uuid.NewString()
	now := time.Now().UTC()
	evJSON, _ := json.Marshal(ev)
	_, err := s.db.Exec(`INSERT INTO website_audits(id, business_id, audit_version, requested_url, final_url, status, evidence_json, error_code, audited_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		id, businessID, "v1", requestedURL, finalURL, string(status), string(evJSON), errorCode, now.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	return &Result{
		ID: id, BusinessID: businessID, Version: "v1",
		RequestedURL: requestedURL, FinalURL: finalURL,
		Status: status, Evidence: ev, ErrorCode: errorCode, AuditedAt: now,
	}, nil
}

func (s *Service) Latest(businessID string) (*Result, error) {
	var r Result
	var status string
	var evJSON string
	var auditedAt string
	err := s.db.QueryRow(`SELECT id, business_id, audit_version, requested_url, final_url, status, evidence_json, error_code, audited_at FROM website_audits WHERE business_id=? ORDER BY audited_at DESC LIMIT 1`, businessID).
		Scan(&r.ID, &r.BusinessID, &r.Version, &r.RequestedURL, &r.FinalURL, &status, &evJSON, &r.ErrorCode, &auditedAt)
	if err != nil {
		return nil, err
	}
	r.Status = Status(status)
	_ = json.Unmarshal([]byte(evJSON), &r.Evidence)
	r.AuditedAt, _ = time.Parse(time.RFC3339, auditedAt)
	return &r, nil
}

func codeFromErr(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "AUDIT_PRIVATE_NETWORK_BLOCKED") {
		return "AUDIT_PRIVATE_NETWORK_BLOCKED"
	}
	if strings.Contains(msg, "AUDIT_INVALID_URL") {
		return "AUDIT_INVALID_URL"
	}
	return "AUDIT_FAILED"
}

func boolPtr(b bool) *bool { return &b }
