package acquisition

import (
	"context"
	"fmt"

	"github.com/gosom/google-maps-scraper/internal/msg/domain"
)

// SearchRequest is the product-level DTO for a discovery run.
type SearchRequest struct {
	Query        string  `json:"query"`
	LocationText string  `json:"location_text"`
	RadiusMeters float64 `json:"radius_meters"`
	Language     string  `json:"language"`
	Goal         string  `json:"goal"`
	MinRating    float64 `json:"min_rating"`
	MinReviews   int     `json:"min_reviews"`
	ExtractEmail bool    `json:"extract_email"`
	ExtraReviews bool    `json:"extra_reviews"`
	FastMode     bool    `json:"fast_mode"`
	TargetCount  int     `json:"target_count"`
	// Advanced overrides; SpeedPreset maps to runner.Config preset when Advanced is nil.
	SpeedPreset string               `json:"speed_preset"` // "conservative"|"balanced"|"fast"|"custom"
	Advanced    *ScraperAdvancedConfig `json:"advanced,omitempty"`
}

type ScraperAdvancedConfig struct {
	Concurrency      *int    `json:"concurrency,omitempty"`
	MaxDepth         *int    `json:"max_depth,omitempty"`
	Zoom             *int    `json:"zoom,omitempty"`
	Proxies          []string `json:"proxies,omitempty"`
	BrowserPoolSize  *int    `json:"browser_pool_size,omitempty"`
	PagesPerBrowser  *int    `json:"pages_per_browser,omitempty"`
	DisablePageReuse bool    `json:"disable_page_reuse,omitempty"`
	GridBBox         string  `json:"grid_bbox,omitempty"`
	GridCellKm       *float64 `json:"grid_cell_km,omitempty"`
	GeoCoordinates   string  `json:"geo_coordinates,omitempty"`
}

// RawPlaceSink receives normalized places.
type RawPlaceSink interface {
	OnPlace(ctx context.Context, place domain.RawPlaceV1) error
}

// ProgressSink receives progress updates.
type ProgressSink interface {
	OnProgress(stage string, found, processed int)
	OnError(code, message string)
}

// Validate returns a stable error code if the request is invalid.
func (r SearchRequest) Validate() error {
	if r.Query == "" {
		return fmt.Errorf("%w: query required", ErrInvalidRequest)
	}
	if r.LocationText == "" && r.Advanced != nil && r.Advanced.GeoCoordinates == "" && r.Advanced.GridBBox == "" {
		// location may be embedded in query like "dentists in Berlin"
	}
	if r.Language == "" {
		// default en, not an error
	}
	return nil
}

// Stable error codes for UI mapping.
var (
	ErrInvalidRequest     = fmt.Errorf("INVALID_SEARCH_REQUEST")
	ErrBrowserMissing     = fmt.Errorf("SCRAPER_BROWSER_MISSING")
	ErrBlocked            = fmt.Errorf("SCRAPER_BLOCKED")
	ErrProxyFailure       = fmt.Errorf("SCRAPER_PROXY_FAILURE")
	ErrCancelled          = fmt.Errorf("SCRAPER_CANCELLED")
)
