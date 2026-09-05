package ai

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ProviderGemini = "gemini"
	ProviderOpenAI = "openai"
	ProviderCustom = "custom"
)

func ValidProvider(p string) bool {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case ProviderGemini, ProviderOpenAI, ProviderCustom:
		return true
	}
	return false
}

type KeySlot struct {
	ID        string `json:"id"`
	Provider  string `json:"provider"`
	Label     string `json:"label"`
	KeyHint   string `json:"key_hint"`
	BaseURL   string `json:"base_url,omitempty"`
	Model     string `json:"model,omitempty"`
	Enabled   bool   `json:"enabled"`
	Priority  int    `json:"priority"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type SlotStore struct{ db *sql.DB }

func NewSlotStore(db *sql.DB) *SlotStore { return &SlotStore{db: db} }

func (s *SlotStore) List() ([]KeySlot, error) {
	rows, err := s.db.Query(`SELECT id, provider, label, key_hint, COALESCE(base_url,''), COALESCE(model,''), enabled, priority, created_at, updated_at FROM ai_key_slots ORDER BY priority ASC, created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []KeySlot{}
	for rows.Next() {
		var sl KeySlot
		var enabled int
		if err := rows.Scan(&sl.ID, &sl.Provider, &sl.Label, &sl.KeyHint, &sl.BaseURL, &sl.Model, &enabled, &sl.Priority, &sl.CreatedAt, &sl.UpdatedAt); err != nil {
			return nil, err
		}
		sl.Enabled = enabled == 1
		out = append(out, sl)
	}
	return out, rows.Err()
}

func (s *SlotStore) nextPriority() int {
	var max sql.NullInt64
	_ = s.db.QueryRow(`SELECT MAX(priority) FROM ai_key_slots`).Scan(&max)
	if !max.Valid {
		return 0
	}
	return int(max.Int64) + 1
}

func (s *SlotStore) Create(provider, label, keyHint, baseURL, model string) (KeySlot, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if !ValidProvider(provider) {
		return KeySlot{}, fmt.Errorf("unknown provider %q (use gemini|openai|custom)", provider)
	}
	label = strings.TrimSpace(label)
	if label == "" {
		return KeySlot{}, fmt.Errorf("label required (mis. akun 1)")
	}
	if provider == ProviderCustom && strings.TrimSpace(baseURL) == "" {
		return KeySlot{}, fmt.Errorf("base_url required for custom provider")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	sl := KeySlot{
		ID: provider + "-" + uuid.NewString()[:8],
		Provider: provider, Label: label, KeyHint: keyHint,
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Model:   strings.TrimSpace(model),
		Enabled: true, Priority: s.nextPriority(),
		CreatedAt: now, UpdatedAt: now,
	}
	_, err := s.db.Exec(`INSERT INTO ai_key_slots(id, provider, label, key_hint, base_url, model, enabled, priority, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		sl.ID, sl.Provider, sl.Label, sl.KeyHint, sl.BaseURL, sl.Model, 1, sl.Priority, now, now)
	if err != nil {
		return KeySlot{}, err
	}
	return sl, nil
}

func (s *SlotStore) Update(id, label string, enabled *bool, baseURL, model string) error {
	var cur KeySlot
	err := s.db.QueryRow(`SELECT id, provider FROM ai_key_slots WHERE id=?`, id).Scan(&cur.ID, &cur.Provider)
	if err == sql.ErrNoRows {
		return fmt.Errorf("slot %q not found", id)
	}
	if err != nil {
		return err
	}
	sets := []string{"label=?", "updated_at=?"}
	args := []interface{}{strings.TrimSpace(label), time.Now().UTC().Format(time.RFC3339)}
	if enabled != nil {
		v := 0
		if *enabled {
			v = 1
		}
		sets = append(sets, "enabled=?")
		args = append(args, v)
	}
	if strings.TrimSpace(baseURL) != "" || cur.Provider == ProviderCustom {
		sets = append(sets, "base_url=?")
		args = append(args, strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	}
	if model != "" || true {
		sets = append(sets, "model=?")
		args = append(args, strings.TrimSpace(model))
	}
	args = append(args, id)
	_, err = s.db.Exec(`UPDATE ai_key_slots SET `+strings.Join(sets, ", ")+` WHERE id=?`, args...)
	return err
}

func (s *SlotStore) Delete(id string) error {
	res, err := s.db.Exec(`DELETE FROM ai_key_slots WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("slot %q not found", id)
	}
	return nil
}

func (s *SlotStore) Move(id string, direction int) error {
	slots, err := s.List()
	if err != nil {
		return err
	}
	idx := -1
	for i, sl := range slots {
		if sl.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("slot %q not found", id)
	}
	j := idx + direction
	if j < 0 || j >= len(slots) {
		return fmt.Errorf("already at edge")
	}
	a, b := slots[idx], slots[j]
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE ai_key_slots SET priority=?, updated_at=? WHERE id=?`, b.Priority, time.Now().UTC().Format(time.RFC3339), a.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE ai_key_slots SET priority=?, updated_at=? WHERE id=?`, a.Priority, time.Now().UTC().Format(time.RFC3339), b.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func KeyHint(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 8 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}
