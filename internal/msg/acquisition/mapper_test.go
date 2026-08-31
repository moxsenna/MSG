package acquisition

import (
	"encoding/json"
	"testing"

	"github.com/gosom/google-maps-scraper/gmaps"
	"github.com/gosom/google-maps-scraper/internal/msg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapEntry_NormalComplete(t *testing.T) {
	e := &gmaps.Entry{
		Link:        "https://www.google.com/maps/place/Test/@1,2",
		Cid:         "12345",
		PlaceID:     "ChIJ123",
		DataID:      "0x123:0x456",
		Title:       "Test Coffee",
		Categories:  []string{"Coffee shop", "Cafe"},
		Category:    "Coffee shop",
		Address:     "Jl Test 1",
		CompleteAddress: gmaps.Address{City: "Jakarta", Country: "Indonesia"},
		WebSite:     "https://testcoffee.com",
		Phone:       "+621234567",
		Emails:      []string{"a@b.com"},
		ReviewCount: 120,
		ReviewRating: 4.5,
		Status:      "OPEN",
		Latitude:    -6.2,
		Longtitude:  106.8,
		PriceRange:  "$$",
		Description: "Best coffee",
		OpenHours:   map[string][]string{"Monday": {"08:00-22:00"}},
		Thumbnail:   "https://thumb",
		Timezone:    "Asia/Jakarta",
		PlusCode:    "6P58+",
		StreetViewURL: "https://streetview",
		ReviewsLink: "https://reviews",
	}

	r := MapEntry(e)
	assert.Equal(t, domain.RawPlaceVersion, r.SourceVersion)
	assert.Equal(t, e.Link, r.Link)
	assert.Equal(t, e.Cid, r.CID)
	assert.Equal(t, e.PlaceID, r.PlaceID)
	assert.Equal(t, e.DataID, r.DataID)
	assert.Equal(t, e.Title, r.Title)
	assert.Equal(t, e.Categories, r.Categories)
	assert.Equal(t, e.Address, r.Address)
	assert.Equal(t, e.WebSite, r.Website)
	assert.Equal(t, e.Phone, r.Phone)
	assert.Equal(t, e.Emails, r.Emails)
	assert.Equal(t, e.ReviewCount, r.ReviewCount)
	assert.Equal(t, e.ReviewRating, r.ReviewRating)
	assert.Equal(t, e.Latitude, r.Latitude)
	assert.Equal(t, e.Longtitude, r.Longitude)
	assert.False(t, r.CapturedAt.IsZero())
}

func TestMapEntry_MissingOptional(t *testing.T) {
	e := &gmaps.Entry{Title: "Only Title"}
	r := MapEntry(e)
	assert.Equal(t, "Only Title", r.Title)
	assert.Empty(t, r.Website)
	assert.Empty(t, r.Emails)
	assert.Empty(t, r.Phone)
	assert.NotNil(t, r.Categories)
	assert.NotNil(t, r.Emails)
}

func TestMapEntry_MultipleEmails(t *testing.T) {
	e := &gmaps.Entry{Title: "X", Emails: []string{"a@b.com", "c@d.com", "e@f.com"}}
	r := MapEntry(e)
	require.Len(t, r.Emails, 3)
	assert.Equal(t, []string{"a@b.com", "c@d.com", "e@f.com"}, r.Emails)
	// ensure copy not alias
	e.Emails[0] = "changed"
	assert.Equal(t, "a@b.com", r.Emails[0])
}

func TestMapEntry_WebsiteNoWebsite(t *testing.T) {
	with := MapEntry(&gmaps.Entry{Title: "With", WebSite: "https://example.com"})
	without := MapEntry(&gmaps.Entry{Title: "Without"})
	assert.Equal(t, "https://example.com", with.Website)
	assert.Empty(t, without.Website)
}

func TestMapEntry_Coordinates(t *testing.T) {
	e := &gmaps.Entry{Title: "Geo", Latitude: 1.234, Longtitude: 5.678}
	r := MapEntry(e)
	assert.InDelta(t, 1.234, r.Latitude, 0.0001)
	assert.InDelta(t, 5.678, r.Longitude, 0.0001)
}

func TestMapEntry_ClosedBusiness(t *testing.T) {
	e := &gmaps.Entry{Title: "Closed", Status: "CLOSED_PERMANENTLY"}
	r := MapEntry(e)
	assert.Equal(t, "CLOSED_PERMANENTLY", r.Status)
}

func TestMapEntry_LegacyLongitude(t *testing.T) {
	// gmaps.Entry uses misspelled Longtitude, RawPlaceV1 uses correct Longitude
	e := &gmaps.Entry{Title: "Legacy", Longtitude: 106.816666}
	r := MapEntry(e)
	assert.Equal(t, 106.816666, r.Longitude)
	// JSON round-trip must emit longitude correctly
	b, err := json.Marshal(r)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Equal(t, 106.816666, m["longitude"])
	assert.NotContains(t, m, "longtitude")
}

func TestMapEntry_Nil(t *testing.T) {
	r := MapEntry(nil)
	assert.Equal(t, domain.RawPlaceVersion, r.SourceVersion)
	assert.Empty(t, r.Title)
}

func TestMapEntry_LinkAndIDs(t *testing.T) {
	e := &gmaps.Entry{Link: "https://maps/place/abc", Cid: "cid123", PlaceID: "place123", DataID: "data123"}
	r := MapEntry(e)
	assert.Equal(t, "https://maps/place/abc", r.Link)
	assert.Equal(t, "cid123", r.CID)
	assert.Equal(t, "place123", r.PlaceID)
	assert.Equal(t, "data123", r.DataID)
}
