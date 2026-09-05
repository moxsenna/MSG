package pipeline

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ValidStage checks against the canonical business pipeline IDs from ADRs / PRD.
var ValidStages = map[string]bool{
	"new":             true,
	"qualified":       true,
	"shortlisted":     true,
	"contacted":       true,
	"replied":         true,
	"interested":      true,
	"meeting":         true,
	"proposal":        true,
	"won":             true,
	"lost":            true,
	"follow_up_later": true,
	"no_response":     true,
	"bad_lead":        true,
}

type StageCard struct {
	BusinessID string  `json:"business_id"`
	Title      string  `json:"title"`
	Category   string  `json:"category"`
	City       string  `json:"city"`
	Score      *int    `json:"score,omitempty"`
	Stage      string  `json:"stage"`
	FollowUpAt *string `json:"follow_up_at,omitempty"`
}

type FollowUpItem struct {
	BusinessID string `json:"business_id"`
	Title      string `json:"title"`
	Phone      string `json:"phone"`
	Stage      string `json:"stage"`
	FollowUpAt string `json:"follow_up_at"`
	IsOverdue  bool   `json:"is_overdue"`
}

type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

// MoveStage atomically moves a business to a new stage, logging activity.
func (s *Service) MoveStage(businessID, newStage string) error {
	if !ValidStages[newStage] {
		return fmt.Errorf("invalid pipeline stage: %s", newStage)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE lead_states SET stage=?, updated_at=? WHERE business_id=?`, newStage, now, businessID)
	if err != nil {
		return fmt.Errorf("update stage: %w", err)
	}

	_, err = tx.Exec(`INSERT INTO activities(id, business_id, type, payload_json, created_at) VALUES(?,?, 'stage_changed', ?, ?)`,
		uuid.NewString(), businessID, fmt.Sprintf(`{"new_stage":"%s"}`, newStage), now)
	if err != nil {
		return fmt.Errorf("insert activity: %w", err)
	}

	return tx.Commit()
}

// GetKanbanBoard returns cards grouped by active sales stages.
// Includes "new" so freshly scraped leads are visible instead of silently dropped.
func (s *Service) GetKanbanBoard() (map[string][]StageCard, error) {
	board := make(map[string][]StageCard)
	activeStages := []string{"new", "qualified", "shortlisted", "contacted", "replied", "interested", "meeting", "proposal", "won", "lost"}
	for _, st := range activeStages {
		board[st] = []StageCard{}
	}

	rows, err := s.db.Query(`
		SELECT b.id, b.title, COALESCE(b.primary_category, ''), COALESCE(b.city, ''), ls.stage, ls.next_follow_up_at, os.overall_score
		FROM businesses b
		JOIN lead_states ls ON ls.business_id=b.id
		LEFT JOIN opportunity_scores os ON os.business_id=b.id AND os.id=(SELECT id FROM opportunity_scores WHERE business_id=b.id ORDER BY calculated_at DESC LIMIT 1)
		ORDER BY b.updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c StageCard
		var fu sql.NullString
		var sc sql.NullInt64
		if err := rows.Scan(&c.BusinessID, &c.Title, &c.Category, &c.City, &c.Stage, &fu, &sc); err != nil {
			return nil, err
		}
		if fu.Valid {
			s := fu.String
			c.FollowUpAt = &s
		}
		if sc.Valid {
			v := int(sc.Int64)
			c.Score = &v
		}
		if _, exists := board[c.Stage]; exists {
			board[c.Stage] = append(board[c.Stage], c)
		}
	}

	return board, rows.Err()
}

// ListFollowUps returns follow-up tasks and flags overdue ones.
func (s *Service) ListFollowUps() ([]FollowUpItem, error) {
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	rows, err := s.db.Query(`
		SELECT b.id, b.title, COALESCE(b.phone, ''), ls.stage, ls.next_follow_up_at
		FROM businesses b
		JOIN lead_states ls ON ls.business_id=b.id
		WHERE ls.next_follow_up_at IS NOT NULL AND ls.next_follow_up_at != ''
		ORDER BY ls.next_follow_up_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []FollowUpItem
	for rows.Next() {
		var it FollowUpItem
		if err := rows.Scan(&it.BusinessID, &it.Title, &it.Phone, &it.Stage, &it.FollowUpAt); err != nil {
			return nil, err
		}
		if it.FollowUpAt < nowStr {
			it.IsOverdue = true
		}
		items = append(items, it)
	}

	return items, rows.Err()
}
