package acquisition

import (
	"time"

	"github.com/gosom/google-maps-scraper/gmaps"
	"github.com/gosom/google-maps-scraper/internal/msg/domain"
)

func MapEntry(e *gmaps.Entry) domain.RawPlaceV1 {
	if e == nil {
		return domain.RawPlaceV1{
			SourceVersion: domain.RawPlaceVersion,
			CapturedAt:    time.Now().UTC(),
		}
	}

	r := domain.RawPlaceV1{
		SourceVersion: domain.RawPlaceVersion,
		CapturedAt:    time.Now().UTC(),
		Link:          e.Link,
		CID:           e.Cid,
		PlaceID:       e.PlaceID,
		DataID:        e.DataID,
		Title:         e.Title,
		Categories:    append([]string(nil), e.Categories...),
		Category:      e.Category,
		Address:       e.Address,
		CompleteAddress: domain.CompleteAddress{
			Borough:    e.CompleteAddress.Borough,
			Street:     e.CompleteAddress.Street,
			City:       e.CompleteAddress.City,
			PostalCode: e.CompleteAddress.PostalCode,
			State:      e.CompleteAddress.State,
			Country:    e.CompleteAddress.Country,
		},
		Website:      e.WebSite,
		Phone:        e.Phone,
		Emails:       append([]string(nil), e.Emails...),
		ReviewCount:  e.ReviewCount,
		ReviewRating: e.ReviewRating,
		Status:       e.Status,
		Latitude:     e.Latitude,
		Longitude:    e.Longtitude,
		PriceRange:   e.PriceRange,
		Description:  e.Description,
		OpenHours:    e.OpenHours,
		Thumbnail:    e.Thumbnail,
		Timezone:     e.Timezone,
		PlusCode:     e.PlusCode,
		StreetViewURL: e.StreetViewURL,
		ReviewsLink:  e.ReviewsLink,
	}

	r.Reservations = make([]domain.LinkSource, len(e.Reservations))
	for i, v := range e.Reservations {
		r.Reservations[i] = domain.LinkSource{Link: v.Link, Source: v.Source}
	}
	r.OrderOnline = make([]domain.LinkSource, len(e.OrderOnline))
	for i, v := range e.OrderOnline {
		r.OrderOnline[i] = domain.LinkSource{Link: v.Link, Source: v.Source}
	}
	r.Menu = domain.LinkSource{Link: e.Menu.Link, Source: e.Menu.Source}

	if r.Categories == nil {
		r.Categories = []string{}
	}
	if r.Emails == nil {
		r.Emails = []string{}
	}

	return r
}
