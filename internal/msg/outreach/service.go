package outreach

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrDNCBlocked = fmt.Errorf("DNC_BLOCKED: outreach blocked by do-not-contact")

type Draft struct {
	ID         string `json:"id"`
	BusinessID string `json:"business_id"`
	Channel    string `json:"channel"` // whatsapp | email
	Tone       string `json:"tone"`
	Body       string `json:"body"`
	CreatedAt  string `json:"created_at"`
}

type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) isDNC(businessID string) (bool, error) {
	var v int
	err := s.db.QueryRow(`SELECT do_not_contact FROM lead_states WHERE business_id=?`, businessID).Scan(&v)
	if err != nil {
		return false, err
	}
	return v == 1, nil
}

func (s *Service) CreateDraft(businessID, channel, tone, body string) (*Draft, error) {
	if dnc, err := s.isDNC(businessID); err != nil {
		return nil, err
	} else if dnc {
		return nil, ErrDNCBlocked
	}
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("empty body")
	}
	if channel != "whatsapp" && channel != "email" {
		return nil, fmt.Errorf("invalid channel %q", channel)
	}
	if tone == "" {
		tone = "friendly"
	}
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`INSERT INTO outreach_drafts(id, business_id, channel, tone, body, created_at, updated_at) VALUES(?,?,?,?,?,?,?)`,
		id, businessID, channel, tone, body, now, now)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.Exec(`INSERT INTO activities(id, business_id, type) VALUES(?,?, 'outreach_draft_created')`, uuid.NewString(), businessID)
	_, _ = s.db.Exec(`INSERT INTO outreach_events(id, business_id, draft_id, type, channel) VALUES(?,?,?,?,?)`, uuid.NewString(), businessID, id, "draft_created", channel)
	return &Draft{ID: id, BusinessID: businessID, Channel: channel, Tone: tone, Body: body, CreatedAt: now}, nil
}

func (s *Service) UpdateDraft(draftID, body string) error {
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("empty body")
	}
	_, err := s.db.Exec(`UPDATE outreach_drafts SET body=?, updated_at=? WHERE id=?`, body, time.Now().UTC().Format(time.RFC3339), draftID)
	return err
}

func (s *Service) ListDrafts(businessID string) ([]Draft, error) {
	rows, err := s.db.Query(`SELECT id, business_id, channel, tone, body, created_at FROM outreach_drafts WHERE business_id=? ORDER BY created_at DESC`, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Draft
	for rows.Next() {
		var d Draft
		if err := rows.Scan(&d.ID, &d.BusinessID, &d.Channel, &d.Tone, &d.Body, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// OpenWhatsApp validates DNC and returns a wa.me URL (does not mark sent).
func (s *Service) OpenWhatsApp(businessID, draftID string) (string, error) {
	if dnc, err := s.isDNC(businessID); err != nil {
		return "", err
	} else if dnc {
		return "", ErrDNCBlocked
	}
	var phone string
	if err := s.db.QueryRow(`SELECT phone FROM businesses WHERE id=?`, businessID).Scan(&phone); err != nil {
		return "", err
	}
	phone = normalizePhone(phone)
	if phone == "" {
		return "", fmt.Errorf("no phone")
	}
	var body string
	if draftID != "" {
		_ = s.db.QueryRow(`SELECT body FROM outreach_drafts WHERE id=? AND business_id=?`, draftID, businessID).Scan(&body)
	}
	waURL := "https://wa.me/" + phone
	if body != "" {
		waURL += "?text=" + url.QueryEscape(body)
	}
	_, _ = s.db.Exec(`INSERT INTO outreach_events(id, business_id, draft_id, type, channel) VALUES(?,?,?,?,?)`, uuid.NewString(), businessID, draftID, "external_contact_opened", "whatsapp")
	_, _ = s.db.Exec(`INSERT INTO activities(id, business_id, type) VALUES(?,?, 'external_contact_opened')`, uuid.NewString(), businessID)
	return waURL, nil
}

func (s *Service) OpenEmail(businessID, draftID string) (string, error) {
	if dnc, err := s.isDNC(businessID); err != nil {
		return "", err
	} else if dnc {
		return "", ErrDNCBlocked
	}
	// find email via business_contacts or emails_json
	var email string
	_ = s.db.QueryRow(`SELECT value FROM business_contacts WHERE business_id=? AND type='email' LIMIT 1`, businessID).Scan(&email)
	if email == "" {
		var emailsJSON string
		_ = s.db.QueryRow(`SELECT emails_json FROM businesses WHERE id=?`, businessID).Scan(&emailsJSON)
		// naive parse: first email in JSON array
		email = firstEmailFromJSON(emailsJSON)
	}
	if email == "" {
		return "", fmt.Errorf("no email")
	}
	var body string
	if draftID != "" {
		_ = s.db.QueryRow(`SELECT body FROM outreach_drafts WHERE id=?`, draftID).Scan(&body)
	}
	mailto := "mailto:" + url.QueryEscape(email)
	if body != "" {
		mailto += "?body=" + url.QueryEscape(body)
	}
	_, _ = s.db.Exec(`INSERT INTO outreach_events(id, business_id, draft_id, type, channel) VALUES(?,?,?,?,?)`, uuid.NewString(), businessID, draftID, "external_contact_opened", "email")
	_, _ = s.db.Exec(`INSERT INTO activities(id, business_id, type) VALUES(?,?, 'external_contact_opened')`, uuid.NewString(), businessID)
	return mailto, nil
}

type Template struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Channel   string `json:"channel"`
	Tone      string `json:"tone"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func (s *Service) CreateTemplate(name, channel, tone, body string) (*Template, error) {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("name and body required")
	}
	if channel != "whatsapp" && channel != "email" {
		return nil, fmt.Errorf("invalid channel %q", channel)
	}
	if tone == "" {
		tone = "friendly"
	}
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`INSERT INTO outreach_templates(id, name, channel, tone, body, created_at, updated_at) VALUES(?,?,?,?,?,?,?)`,
		id, strings.TrimSpace(name), channel, tone, body, now, now)
	if err != nil {
		return nil, err
	}
	return &Template{ID: id, Name: strings.TrimSpace(name), Channel: channel, Tone: tone, Body: body, CreatedAt: now}, nil
}

func (s *Service) ListTemplates() ([]Template, error) {
	rows, err := s.db.Query(`SELECT id, name, channel, tone, body, created_at FROM outreach_templates ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Template{}
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.Name, &t.Channel, &t.Tone, &t.Body, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Service) DeleteTemplate(id string) error {
	res, err := s.db.Exec(`DELETE FROM outreach_templates WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("template not found")
	}
	return nil
}

// MarkContacted explicitly records that user sent the message (human-in-loop).
func (s *Service) MarkContacted(businessID, draftID, channel string) error {
	if dnc, err := s.isDNC(businessID); err != nil {
		return err
	} else if dnc {
		return ErrDNCBlocked
	}
	_, err := s.db.Exec(`INSERT INTO outreach_events(id, business_id, draft_id, type, channel) VALUES(?,?,?,?,?)`, uuid.NewString(), businessID, draftID, "contact_marked_sent", channel)
	if err != nil {
		return err
	}
	_, _ = s.db.Exec(`INSERT INTO activities(id, business_id, type) VALUES(?,?, 'contact_marked_sent')`, uuid.NewString(), businessID)
	_, _ = s.db.Exec(`UPDATE lead_states SET stage='contacted', updated_at=? WHERE business_id=? AND stage IN ('new','qualified','shortlisted')`, time.Now().UTC().Format(time.RFC3339), businessID)
	return nil
}

func IsMobilePhone(p string) bool {
	digits := ""
	for _, r := range p {
		if r >= '0' && r <= '9' {
			digits += string(r)
		}
	}
	m := digits
	if strings.HasPrefix(m, "0") {
		m = m[1:]
	}
	if strings.HasPrefix(m, "62") {
		m = m[2:]
	}
	return len(m) >= 9 && len(m) <= 14 && m[0] == '8'
}

func normalizePhone(p string) string {
	var b strings.Builder
	for _, r := range p {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	s := b.String()
	// wa.me expects full international without + or 00
	if strings.HasPrefix(s, "0") {
		s = "62" + s[1:]
	}
	if !strings.HasPrefix(s, "62") && len(s) >= 9 && s[0] == '8' {
		s = "62" + s
	}
	return s
}

func firstEmailFromJSON(j string) string {
	j = strings.TrimSpace(j)
	if !strings.HasPrefix(j, "[") {
		return ""
	}
	// very small parser: find first "..."
	start := strings.Index(j, "\"")
	if start == -1 {
		return ""
	}
	end := strings.Index(j[start+1:], "\"")
	if end == -1 {
		return ""
	}
	return j[start+1 : start+1+end]
}
