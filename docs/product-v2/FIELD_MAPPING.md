# RawPlaceV1 Field Mapping — gmaps.Entry → domain.RawPlaceV1

| RawPlaceV1 field | gmaps.Entry source | Transform |
|---|---|---|
| SourceVersion | const `v1` | hardcode |
| CapturedAt | `time.Now().UTC()` | generated at map time |
| Link | `e.Link` | direct |
| CID | `e.Cid` | direct (note Entry uses `Cid`) |
| PlaceID | `e.PlaceID` | direct |
| DataID | `e.DataID` | direct |
| Title | `e.Title` | direct |
| Categories | `e.Categories` | copy slice |
| Category | `e.Category` | direct |
| Address | `e.Address` | direct |
| CompleteAddress | `e.CompleteAddress` fields | struct copy |
| Website | `e.WebSite` | direct |
| Phone | `e.Phone` | direct |
| Emails | `e.Emails` | copy slice, nil→[] |
| ReviewCount | `e.ReviewCount` | direct |
| ReviewRating | `e.ReviewRating` | direct |
| Status | `e.Status` | direct |
| Latitude | `e.Latitude` | direct |
| Longitude | `e.Longtitude` | correct spelling, typo fixed at boundary |
| PriceRange | `e.PriceRange` | direct |
| Description | `e.Description` | direct |
| OpenHours | `e.OpenHours` | direct reference (immutable after scrape) |
| Reservations | `e.Reservations` | slice copy LinkSource |
| OrderOnline | `e.OrderOnline` | slice copy LinkSource |
| Menu | `e.Menu` | struct copy |
| Thumbnail | `e.Thumbnail` | direct |
| Timezone | `e.Timezone` | direct |
| PlusCode | `e.PlusCode` | direct |
| StreetViewURL | `e.StreetViewURL` | direct |
| ReviewsLink | `e.ReviewsLink` | direct |

No CRM, pipeline, or scoring fields are stored in RawPlaceV1.
