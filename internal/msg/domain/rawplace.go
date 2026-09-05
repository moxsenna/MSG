package domain

import "time"

const RawPlaceVersion = "v1"

type CompleteAddress struct {
	Borough    string `json:"borough"`
	Street     string `json:"street"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	State      string `json:"state"`
	Country    string `json:"country"`
}

type LinkSource struct {
	Link   string `json:"link"`
	Source string `json:"source"`
}

type RawPlaceV1 struct {
	SourceVersion string `json:"source_version"`
	CapturedAt    time.Time `json:"captured_at"`

	Link    string `json:"link"`
	CID     string `json:"cid"`
	PlaceID string `json:"place_id"`
	DataID  string `json:"data_id"`

	Title      string   `json:"title"`
	Categories []string `json:"categories"`
	Category   string   `json:"category"`

	Address         string          `json:"address"`
	CompleteAddress CompleteAddress `json:"complete_address"`

	Website string   `json:"website"`
	Phone   string   `json:"phone"`
	Emails  []string `json:"emails"`
	Socials []string `json:"socials,omitempty"`

	ReviewCount  int     `json:"review_count"`
	ReviewRating float64 `json:"review_rating"`
	Status       string  `json:"status"`

	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	PriceRange  string `json:"price_range"`
	Description string `json:"description"`

	OpenHours    map[string][]string `json:"open_hours"`
	Reservations []LinkSource        `json:"reservations"`
	OrderOnline  []LinkSource        `json:"order_online"`
	Menu         LinkSource          `json:"menu"`

	Thumbnail     string `json:"thumbnail"`
	Timezone      string `json:"timezone"`
	PlusCode      string `json:"plus_code"`
	StreetViewURL string `json:"street_view_url"`
	ReviewsLink   string `json:"reviews_link"`
}
