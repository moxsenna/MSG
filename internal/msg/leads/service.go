package leads

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gosom/google-maps-scraper/internal/msg/outreach"
)

// LeadListItem is the DTO for grid rows.
type LeadListItem struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Category   string  `json:"category"`
	City       string  `json:"city"`
	Address    string  `json:"address"`
	MapsLink   string  `json:"maps_link"`
	Website    string  `json:"website"`
	Phone      string  `json:"phone"`
	Stage      string  `json:"stage"`
	Score      *int    `json:"score,omitempty"`
	DNC        bool    `json:"dnc"`
	IsMobile   bool    `json:"is_mobile"`
	FollowUpAt *string `json:"follow_up_at,omitempty"`
}

// Detail includes CRM state.
type Detail struct {
	LeadListItem
	Address      string   `json:"address"`
	MapsLink     string   `json:"maps_link"`
	Socials      []string `json:"socials"`
	WonValue     *float64 `json:"won_value,omitempty"`
	LostReason   string   `json:"lost_reason,omitempty"`
	ReviewRating float64  `json:"review_rating"`
	ReviewCount  int      `json:"review_count"`
	Notes        []Note   `json:"notes"`
	Tags         []string `json:"tags"`
	Activities   []Activity `json:"activities"`
}

type Note struct {
	ID        string `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}
type Activity struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	At   string `json:"created_at"`
}

type Query struct {
	Search      string  `json:"search"`
	Stage       string  `json:"stage"`
	City        string  `json:"city"`
	HasWebsite  *bool   `json:"has_website"`
	HasContact  *bool   `json:"has_contact"`
	MinRating   float64 `json:"min_rating"`
	MinReviews  int     `json:"min_reviews"`
	DNC         *bool   `json:"dnc"`
	Limit       int     `json:"limit"`
	Offset      int     `json:"offset"`
	SortBy      string  `json:"sort_by"` // "score" | "updated"
}

type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

// List returns paginated leads server-side (required for 50k).
func (s *Service) List(q Query) ([]LeadListItem, int, error) {
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 50
	}
	where := []string{"1=1"}
	args := []interface{}{}
	if q.Search != "" {
		where = append(where, "(lower(b.title) LIKE ? OR lower(b.primary_category) LIKE ?)")
		like := "%" + strings.ToLower(q.Search) + "%"
		args = append(args, like, like)
	}
	if q.Stage != "" {
		where = append(where, "ls.stage=?")
		args = append(args, q.Stage)
	}
	if q.City != "" {
		where = append(where, "lower(b.city)=lower(?)")
		args = append(args, q.City)
	}
	if q.HasWebsite != nil {
		if *q.HasWebsite {
			where = append(where, "b.website IS NOT NULL AND b.website!=''")
		} else {
			where = append(where, "(b.website IS NULL OR b.website='')")
		}
	}
	if q.DNC != nil {
		where = append(where, "ls.do_not_contact=?")
		if *q.DNC {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	if q.HasContact != nil {
		if *q.HasContact {
			where = append(where, "(b.phone IS NOT NULL AND b.phone!='')")
		} else {
			where = append(where, "(b.phone IS NULL OR b.phone='')")
		}
	}
	if q.MinRating > 0 {
		where = append(where, "b.review_rating>=?")
		args = append(args, q.MinRating)
	}
	if q.MinReviews > 0 {
		where = append(where, "b.review_count>=?")
		args = append(args, q.MinReviews)
	}
	countSQL := `SELECT count(*) FROM businesses b JOIN lead_states ls ON ls.business_id=b.id WHERE ` + strings.Join(where, " AND ")
	var total int
	if err := s.db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	order := "b.updated_at DESC"
	if q.SortBy == "score" {
		order = "COALESCE(os.overall_score, -1) DESC, b.updated_at DESC"
	}
	sqlStr := `SELECT b.id,b.title,b.primary_category,b.city,b.address_text,b.source_link,b.website,b.phone,ls.stage,ls.do_not_contact,ls.next_follow_up_at, os.overall_score
		FROM businesses b JOIN lead_states ls ON ls.business_id=b.id LEFT JOIN opportunity_scores os ON os.business_id=b.id AND os.id=(SELECT id FROM opportunity_scores WHERE business_id=b.id ORDER BY calculated_at DESC LIMIT 1)
		WHERE ` + strings.Join(where, " AND ") + ` ORDER BY ` + order + ` LIMIT ? OFFSET ?`
	args = append(args, q.Limit, q.Offset)
	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []LeadListItem
	for rows.Next() {
		var it LeadListItem
		var stage string
		var dnc int
		var followUp sql.NullString
		var score sql.NullInt64
		var cat, city, addr, link, website, phone sql.NullString
		if err := rows.Scan(&it.ID, &it.Title, &cat, &city, &addr, &link, &website, &phone, &stage, &dnc, &followUp, &score); err != nil {
			return nil, 0, err
		}
		if cat.Valid {
			it.Category = cat.String
		}
		if city.Valid {
			it.City = city.String
		}
		if addr.Valid {
			it.Address = addr.String
		}
		if link.Valid {
			it.MapsLink = link.String
		}
		if website.Valid {
			it.Website = website.String
		}
		if phone.Valid {
			it.Phone = phone.String
			it.IsMobile = outreach.IsMobilePhone(phone.String)
		}
		it.Stage = stage
		it.DNC = dnc == 1
		if followUp.Valid {
			s := followUp.String
			it.FollowUpAt = &s
		}
		if score.Valid {
			v := int(score.Int64)
			it.Score = &v
		}
		out = append(out, it)
	}
	return out, total, rows.Err()
}

func (s *Service) Get(id string) (*Detail, error) {
	var d Detail
	var stage string
	var dnc int
	var followUp sql.NullString
	var score sql.NullInt64
	var city, website, phone, addr, link, lostReason sql.NullString
	var wonValue sql.NullFloat64
	err := s.db.QueryRow(`SELECT b.id,b.title,b.primary_category,b.city,b.website,b.phone,b.address_text,b.source_link,b.review_rating,b.review_count, ls.stage, ls.do_not_contact, ls.next_follow_up_at, ls.won_value, ls.lost_reason, os.overall_score FROM businesses b JOIN lead_states ls ON ls.business_id=b.id LEFT JOIN opportunity_scores os ON os.business_id=b.id AND os.id=(SELECT id FROM opportunity_scores WHERE business_id=b.id ORDER BY calculated_at DESC LIMIT 1) WHERE b.id=?`, id).Scan(&d.ID, &d.Title, &d.Category, &city, &website, &phone, &addr, &link, &d.ReviewRating, &d.ReviewCount, &stage, &dnc, &followUp, &wonValue, &lostReason, &score)
	if err != nil {
		return nil, err
	}
	if city.Valid {
		d.City = city.String
	}
	if website.Valid {
		d.Website = website.String
	}
	if phone.Valid {
		d.Phone = phone.String
	}
	if addr.Valid {
		d.Address = addr.String
	}
	if link.Valid {
		d.MapsLink = link.String
	}
	if wonValue.Valid {
		v := wonValue.Float64
		d.WonValue = &v
	}
	if lostReason.Valid {
		d.LostReason = lostReason.String
	}
	d.IsMobile = outreach.IsMobilePhone(d.Phone)
	d.Socials = []string{}
	if rows0, err := s.db.Query(`SELECT value FROM business_contacts WHERE business_id=? AND type='social' ORDER BY value`, id); err == nil && rows0 != nil {
		defer rows0.Close()
		for rows0.Next() {
			var v string
			_ = rows0.Scan(&v)
			d.Socials = append(d.Socials, v)
		}
	}
	d.Stage = stage
	d.DNC = dnc == 1
	if followUp.Valid {
		s := followUp.String
		d.FollowUpAt = &s
	}
	if score.Valid {
		v := int(score.Int64)
		d.Score = &v
	}
	// notes
	rows, _ := s.db.Query(`SELECT id, body, created_at FROM notes WHERE business_id=? ORDER BY created_at DESC`, id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var n Note
			_ = rows.Scan(&n.ID, &n.Body, &n.CreatedAt)
			d.Notes = append(d.Notes, n)
		}
	}
	// tags
	rows2, _ := s.db.Query(`SELECT t.name FROM tags t JOIN business_tags bt ON bt.tag_id=t.id WHERE bt.business_id=?`, id)
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var name string
			_ = rows2.Scan(&name)
			d.Tags = append(d.Tags, name)
		}
	}
	// activities
	rows3, _ := s.db.Query(`SELECT id, type, created_at FROM activities WHERE business_id=? ORDER BY created_at DESC LIMIT 50`, id)
	if rows3 != nil {
		defer rows3.Close()
		for rows3.Next() {
			var a Activity
			_ = rows3.Scan(&a.ID, &a.Type, &a.At)
			d.Activities = append(d.Activities, a)
		}
	}
	_ = json.Marshal // keep import used if needed
	return &d, nil
}

func (s *Service) UpdateStage(id, stage string) error {
	valid := map[string]bool{"new": true, "qualified": true, "shortlisted": true, "contacted": true, "replied": true, "interested": true, "meeting": true, "proposal": true, "won": true, "lost": true, "follow_up_later": true, "no_response": true, "bad_lead": true}
	if !valid[stage] {
		return fmt.Errorf("invalid stage %q", stage)
	}
	_, err := s.db.Exec(`UPDATE lead_states SET stage=?, updated_at=? WHERE business_id=?`, stage, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	_, _ = s.db.Exec(`INSERT INTO activities(id, business_id, type) VALUES(?,?, 'stage_changed')`, uuid.NewString(), id)
	return nil
}

func (s *Service) SetDNC(id string, dnc bool) error {
	v := 0
	if dnc {
		v = 1
	}
	_, err := s.db.Exec(`UPDATE lead_states SET do_not_contact=?, updated_at=? WHERE business_id=?`, v, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *Service) AddNote(id, body string) error {
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("empty note")
	}
	_, err := s.db.Exec(`INSERT INTO notes(id, business_id, body) VALUES(?,?,?)`, uuid.NewString(), id, body)
	if err != nil {
		return err
	}
	_, _ = s.db.Exec(`INSERT INTO activities(id, business_id, type) VALUES(?,?, 'note_added')`, uuid.NewString(), id)
	return nil
}

func (s *Service) SetFollowUp(id string, t *time.Time) error {
	var v sql.NullString
	if t != nil {
		v = sql.NullString{String: t.UTC().Format(time.RFC3339), Valid: true}
	}
	_, err := s.db.Exec(`UPDATE lead_states SET next_follow_up_at=?, updated_at=? WHERE business_id=?`, v, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *Service) SetDealOutcome(id, stage string, value float64, reason string) error {
	if stage != "won" && stage != "lost" {
		return fmt.Errorf("stage must be won or lost")
	}
	if value < 0 {
		return fmt.Errorf("deal value cannot be negative")
	}
	_, err := s.db.Exec(`UPDATE lead_states SET stage=?, won_value=?, lost_reason=?, updated_at=? WHERE business_id=?`,
		stage, value, strings.TrimSpace(reason), time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	_, _ = s.db.Exec(`INSERT INTO activities(id, business_id, type) VALUES(?,?, 'stage_changed')`, uuid.NewString(), id)
	return nil
}

func (s *Service) AddTag(id, tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return fmt.Errorf("empty tag")
	}
	var tagID string
	err := s.db.QueryRow(`SELECT id FROM tags WHERE name=?`, tag).Scan(&tagID)
	if err == sql.ErrNoRows {
		tagID = uuid.NewString()
		if _, err := s.db.Exec(`INSERT INTO tags(id, name) VALUES(?,?)`, tagID, tag); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT OR IGNORE INTO business_tags(business_id, tag_id) VALUES(?,?)`, id, tagID)
	return err
}
