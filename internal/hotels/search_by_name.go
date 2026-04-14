package hotels

import (
	"context"
	"fmt"
	"sync"
	"strings"

	"github.com/MikkoParkkola/trvl/internal/models"
)

// word(s)), search that area, then fuzzy-match the hotel name in results. If that
// fails we fall back to searching the full query as the location.
func SearchHotelByName(ctx context.Context, query string, checkIn, checkOut string) (*models.HotelResult, error) {
	if query == "" {
		return nil, fmt.Errorf("hotel name query is required")
	}
	if checkIn == "" || checkOut == "" {
		return nil, fmt.Errorf("check-in and check-out dates are required")
	}

	opts := HotelSearchOptions{
		CheckIn:  checkIn,
		CheckOut: checkOut,
		Guests:   2,
		Currency: "USD",
	}

	// Build search location candidates: prefer context after comma, then last word.
	candidates := buildLocationCandidates(query)

	var lastErr error
	for _, loc := range candidates {
		result, err := SearchHotels(ctx, loc, opts)
		if err != nil {
			lastErr = err
			continue
		}
		if len(result.Hotels) == 0 {
			continue
		}

		match := findBestNameMatch(result.Hotels, query)
		if match != nil {
			return match, nil
		}

		// Area search succeeded but no name match — return first result with a note.
		first := result.Hotels[0]
		return &first, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("hotel name search: %w", lastErr)
	}
	return nil, fmt.Errorf("no hotels found for %q", query)
}

// buildLocationCandidates generates location search strings from a hotel name query.
// E.g. "Beverly Hills Heights, Tenerife" -> ["Tenerife", "Beverly Hills Heights Tenerife"]
func buildLocationCandidates(query string) []string {
	var candidates []string

	// If comma-separated, use the part after the last comma as primary location.
	if idx := strings.LastIndex(query, ","); idx >= 0 {
		after := strings.TrimSpace(query[idx+1:])
		before := strings.TrimSpace(query[:idx])
		if after != "" {
			candidates = append(candidates, after)
		}
		// Also try "before after" as the full query.
		if before != "" && after != "" {
			candidates = append(candidates, before+" "+after)
		}
	}

	// Try the full query as location (works when it contains a city).
	candidates = append(candidates, query)

	return candidates
}

// findBestNameMatch searches hotels for the best fuzzy match to the query.
func findBestNameMatch(hotels []models.HotelResult, query string) *models.HotelResult {
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	var best *models.HotelResult
	bestScore := 0

	for i := range hotels {
		h := &hotels[i]
		nameLower := strings.ToLower(h.Name)

		score := 0
		// Exact contains match scores highest.
		if strings.Contains(nameLower, queryLower) {
			score = 100
		} else {
			// Count how many query words (≥3 chars) appear in the hotel name.
			for _, w := range queryWords {
				if len(w) >= 3 && strings.Contains(nameLower, w) {
					score += 10
				}
			}
		}

		if score > bestScore {
			bestScore = score
			best = h
		}
	}

	if bestScore == 0 {
		return nil
	}
	return best
}

// enrichHotelAmenities fetches detail pages for the top N hotels to get full
// amenity lists. Runs up to 3 concurrent fetches. Hotels without a HotelID
// are skipped. Failures are silently ignored (search results still have
// partial amenities from the search page).
func enrichHotelAmenities(ctx context.Context, hotels []models.HotelResult, limit int) []models.HotelResult {
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	// Collect indices of hotels eligible for enrichment.
	var indices []int
	for i := range hotels {
		if hotels[i].HotelID != "" && len(indices) < limit {
			indices = append(indices, i)
		}
	}
	if len(indices) == 0 {
		return hotels
	}

	// Fetch detail pages in parallel with concurrency limit of 3.
	const concurrency = 3
	type result struct {
		index     int
		amenities []string
	}

	results := make(chan result, len(indices))
	sem := make(chan struct{}, concurrency)

	var wg sync.WaitGroup
	for _, idx := range indices {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			amenities, err := FetchHotelAmenities(ctx, hotels[i].HotelID)
			if err != nil || len(amenities) == 0 {
				return
			}
			results <- result{index: i, amenities: amenities}
		}(idx)
	}

	// Close results channel when all goroutines complete.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Apply enriched amenities back to hotels.
	for r := range results {
		hotels[r.index].Amenities = mergeAmenities(hotels[r.index].Amenities, r.amenities)
	}

	return hotels
}

// propertyTypeCode converts a human-readable property type string to the
// Google Hotels &ptype= parameter value. Returns "" if the type is unknown
// or empty (meaning: no filter applied).
//
// Known Google Hotels ptype values (reverse-engineered):
//
//	2 = hotel, 3 = apartment, 4 = hostel, 5 = resort, 7 = bnb, 8 = villa
func propertyTypeCode(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "hotel":
		return "2"
	case "apartment":
		return "3"
	case "hostel":
		return "4"
	case "resort":
		return "5"
	case "bnb", "bed_and_breakfast", "bed and breakfast":
		return "7"
	case "villa":
		return "8"
	default:
		return ""
	}
}

// mergeAmenities combines two amenity lists, deduplicating by lowercase name.
// The first list's items take priority in ordering.
// tagHotelSource stamps each hotel with a PriceSource for the given provider
// so that MergeHotelResults can track per-provider prices. Hotels that already
// carry Sources (e.g. from a previous enrichment pass) are left unchanged.
func tagHotelSource(hotels []models.HotelResult, provider string) []models.HotelResult {
	tagged := make([]models.HotelResult, len(hotels))
	copy(tagged, hotels)
	for i := range tagged {
		if len(tagged[i].Sources) == 0 {
			tagged[i].Sources = []models.PriceSource{{
				Provider:   provider,
				Price:      tagged[i].Price,
				Currency:   tagged[i].Currency,
				BookingURL: tagged[i].BookingURL,
			}}
		}
	}
	return tagged
}

func mergeAmenities(existing, additional []string) []string {
	seen := make(map[string]bool, len(existing)+len(additional))
	var merged []string

	for _, a := range existing {
		lower := strings.ToLower(a)
		if !seen[lower] {
			seen[lower] = true
			merged = append(merged, a)
		}
	}
	for _, a := range additional {
		lower := strings.ToLower(a)
		if !seen[lower] {
			seen[lower] = true
			merged = append(merged, a)
		}
	}

	return merged
}
