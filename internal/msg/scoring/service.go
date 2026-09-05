package scoring

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const RuleSetVersion = "v2"

type Evidence struct {
	Code   string `json:"code"`
	Label  string `json:"label"`
	Points int    `json:"points"`
	Detail string `json:"detail,omitempty"`
}

type Result struct {
	ID             string     `json:"id"`
	BusinessID     string     `json:"business_id"`
	RuleSetVersion string     `json:"rule_set_version"`
	OverallScore   int        `json:"overall_score"`
	WebsiteScore   int        `json:"website_score"`
	AppScore       int        `json:"app_score"`
	Priority       string     `json:"priority"`
	Confidence     float64    `json:"confidence"`
	Evidence       []Evidence `json:"evidence"`
	CalculatedAt   time.Time  `json:"calculated_at"`
}

type Inputs struct {
	BusinessID   string
	Title        string
	Category     string
	Website      string
	Phone        string
	ReviewRating float64
	ReviewCount  int
	Status       string
	// audit-derived
	HasWebsite       bool
	IsHTTPS          *bool
	HasMeta          *bool
	HasContact       *bool
	IsResponsive     *bool
	HasBookingSignal *bool
}

// Service persists scores to SQLite opportunity_scores.
type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

// CalculateV2 computes deterministic 0-100 scores with evidence.
// Pure function — no DB.
func CalculateV2(in Inputs) Result {
	var evidence []Evidence
	websiteScore := 0
	appScore := 0

	// closed rejection
	isClosed := strings.Contains(strings.ToLower(in.Status), "closed")
	if isClosed {
		return Result{
			RuleSetVersion: RuleSetVersion,
			OverallScore:   0,
			WebsiteScore:   0,
			AppScore:       0,
			Priority:       "Low",
			Confidence:     0.9,
			Evidence:       []Evidence{{Code: "closed", Label: "Business is closed", Points: 0, Detail: in.Status}},
			CalculatedAt:   time.Now().UTC(),
		}
	}

	// 1. Website dimension (authoritative)
	if !in.HasWebsite && strings.TrimSpace(in.Website) == "" {
		websiteScore += 50
		evidence = append(evidence, Evidence{Code: "no_website", Label: "No website detected", Points: 50, Detail: "High website creation opportunity"})
		appScore += 15 // no website also implies digital system gap
		evidence = append(evidence, Evidence{Code: "no_website_app", Label: "No digital presence", Points: 15})
	} else {
		if in.IsHTTPS != nil && !*in.IsHTTPS {
			websiteScore += 10
			evidence = append(evidence, Evidence{Code: "no_https", Label: "Website without HTTPS", Points: 10})
		}
		if in.HasMeta != nil && !*in.HasMeta {
			websiteScore += 5
			evidence = append(evidence, Evidence{Code: "no_meta", Label: "Missing meta description", Points: 5})
		}
		if in.HasContact != nil && !*in.HasContact {
			websiteScore += 5
			evidence = append(evidence, Evidence{Code: "no_contact", Label: "No contact info on site", Points: 5})
		}
		if in.IsResponsive != nil && !*in.IsResponsive {
			websiteScore += 5
			evidence = append(evidence, Evidence{Code: "not_responsive", Label: "Not mobile responsive", Points: 5})
		}
	}

	// 2. Rating
	if in.ReviewRating > 4.5 {
		websiteScore += 15
		appScore += 10
		evidence = append(evidence, Evidence{Code: "rating_high", Label: "Rating > 4.5", Points: 15, Detail: fmt.Sprintf("%.1f", in.ReviewRating)})
	} else if in.ReviewRating > 4.0 {
		websiteScore += 5
		appScore += 5
		evidence = append(evidence, Evidence{Code: "rating_good", Label: "Rating > 4.0", Points: 5})
	}

	// 3. Reviews
	if in.ReviewCount > 100 {
		websiteScore += 15
		appScore += 10
		evidence = append(evidence, Evidence{Code: "reviews_high", Label: "Reviews > 100", Points: 15, Detail: fmt.Sprintf("%d", in.ReviewCount)})
	} else if in.ReviewCount > 50 {
		websiteScore += 5
		appScore += 5
		evidence = append(evidence, Evidence{Code: "reviews_mid", Label: "Reviews > 50", Points: 5})
	}

	// 4. Phone (Indonesian mobile)
	phone := strings.ReplaceAll(in.Phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	isMobile := false
	if phone != "" {
		digits := ""
		for _, r := range phone {
			if r >= '0' && r <= '9' {
				digits += string(r)
			}
		}
		// normalize +62 / 62 / 0
		m := digits
		if strings.HasPrefix(m, "0") {
			m = m[1:]
		}
		if strings.HasPrefix(m, "62") {
			m = m[2:]
		}
		// Indonesian mobile starts with 8
		if len(m) >= 9 && m[0] == '8' {
			isMobile = true
		}
	}
	if isMobile {
		websiteScore += 5
		appScore += 10
		evidence = append(evidence, Evidence{Code: "mobile", Label: "Indonesian mobile (WhatsApp)", Points: 10, Detail: in.Phone})
	}

	// 5. Active bonus
	active := !isClosed
	if active {
		websiteScore += 10
		evidence = append(evidence, Evidence{Code: "active", Label: "Business appears active", Points: 10})
	}

	// 6. App/system signals
	if in.HasBookingSignal != nil && *in.HasBookingSignal {
		appScore += 10
		evidence = append(evidence, Evidence{Code: "booking_signal", Label: "Booking/order signal present", Points: 10})
	} else {
		// lack of booking is an opportunity for booking/app
		if wantsBooking(in.Category) {
			appScore += 15
			evidence = append(evidence, Evidence{Code: "booking_opportunity", Label: "Category benefits from booking/app", Points: 15, Detail: in.Category})
		}
	}

	// clamp 0-100
	websiteScore = clamp(websiteScore, 0, 100)
	appScore = clamp(appScore, 0, 100)
	overall := websiteScore
	if appScore > overall {
		overall = appScore
	}
	// also blended: 60% max + 40% avg for overall nuance
	blended := int(float64(overall)*0.6 + float64((websiteScore+appScore)/2)*0.4)
	if blended > overall {
		overall = clamp(blended, 0, 100)
	}

	priority := "Low"
	if overall >= 80 {
		priority = "Hot Lead"
	} else if overall >= 50 {
		priority = "Warm Lead"
	}

	// confidence based on data completeness
	confidence := 0.7
	if in.ReviewRating > 0 && in.ReviewCount > 0 && in.Phone != "" {
		confidence = 0.85
	}
	if isClosed {
		confidence = 0.9
	}
	if len(evidence) == 0 {
		confidence = 0.5
	}

	return Result{
		RuleSetVersion: RuleSetVersion,
		OverallScore:   overall,
		WebsiteScore:   websiteScore,
		AppScore:       appScore,
		Priority:       priority,
		Confidence:     confidence,
		Evidence:       evidence,
		CalculatedAt:   time.Now().UTC(),
	}
}

func wantsBooking(category string) bool {
	c := strings.ToLower(category)
	keywords := []string{"restaurant", "cafe", "hotel", "spa", "clinic", "salon", "barber", "fitness", "gym", "rental"}
	for _, k := range keywords {
		if strings.Contains(c, k) {
			return true
		}
	}
	return false
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func InputsFromPlace(title, category, website, phone, status string, rating float64, reviews int) Inputs {
	return Inputs{
		Title: title, Category: category, Website: website, Phone: phone,
		ReviewRating: rating, ReviewCount: reviews, Status: status,
		HasWebsite: strings.TrimSpace(website) != "",
	}
}

// ScoreMissing calculates and persists scores for businesses that have none.
// Returns the number of businesses scored.
func (s *Service) ScoreMissing() (int, error) {
	rows, err := s.db.Query(`
		SELECT b.id, b.title, COALESCE(b.primary_category,''), COALESCE(b.website,''), COALESCE(b.phone,''),
		       COALESCE(b.status,''), COALESCE(b.review_rating,0), COALESCE(b.review_count,0)
		FROM businesses b LEFT JOIN opportunity_scores os ON os.business_id=b.id
		WHERE os.id IS NULL`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	type row struct {
		id, title, cat, web, phone, status string
		rating                              float64
		reviews                             int
	}
	var pending []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.title, &r.cat, &r.web, &r.phone, &r.status, &r.rating, &r.reviews); err != nil {
			return 0, err
		}
		pending = append(pending, r)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, r := range pending {
		res := CalculateV2(InputsFromPlace(r.title, r.cat, r.web, r.phone, r.status, r.rating, r.reviews))
		if _, err := s.Persist(res, r.id); err != nil {
			return 0, err
		}
	}
	return len(pending), nil
}

// Persist stores the calculated result for a business.
func (s *Service) Persist(ctxResult Result, businessID string) (*Result, error) {
	ctxResult.BusinessID = businessID
	ctxResult.ID = uuid.NewString()
	if ctxResult.CalculatedAt.IsZero() {
		ctxResult.CalculatedAt = time.Now().UTC()
	}
	evJSON, _ := json.Marshal(ctxResult.Evidence)
	_, err := s.db.Exec(`INSERT INTO opportunity_scores(id, business_id, rule_set_version, overall_score, website_score, app_score, priority, confidence, evidence_json, calculated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		ctxResult.ID, businessID, ctxResult.RuleSetVersion, ctxResult.OverallScore, ctxResult.WebsiteScore, ctxResult.AppScore, ctxResult.Priority, ctxResult.Confidence, string(evJSON), ctxResult.CalculatedAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	return &ctxResult, nil
}

func (s *Service) Latest(businessID string) (*Result, error) {
	var r Result
	var evJSON string
	var calculatedAt string
	err := s.db.QueryRow(`SELECT id, business_id, rule_set_version, overall_score, website_score, app_score, priority, confidence, evidence_json, calculated_at FROM opportunity_scores WHERE business_id=? ORDER BY calculated_at DESC LIMIT 1`, businessID).
		Scan(&r.ID, &r.BusinessID, &r.RuleSetVersion, &r.OverallScore, &r.WebsiteScore, &r.AppScore, &r.Priority, &r.Confidence, &evJSON, &calculatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(evJSON), &r.Evidence)
	r.CalculatedAt, _ = time.Parse(time.RFC3339, calculatedAt)
	return &r, nil
}
