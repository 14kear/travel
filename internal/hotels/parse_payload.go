package hotels

import (
	"encoding/json"
	"fmt"

	"github.com/MikkoParkkola/trvl/internal/models"
)

// is preferred.
func ParseHotelSearchResponse(entries []any, currency string) ([]models.HotelResult, error) {
	// Try to extract the AtySUc payload first.
	payload, err := extractBatchPayload(entries, "AtySUc")
	if err != nil {
		return parseHotelsFromRaw(entries, currency)
	}

	return parseHotelsFromPayload(payload, currency)
}

// extractBatchPayload extracts the inner JSON payload from a batchexecute
// response entry.
func extractBatchPayload(entries []any, rpcid string) (any, error) {
	for _, entry := range entries {
		arr, ok := entry.([]any)
		if !ok {
			continue
		}

		for _, item := range arr {
			itemArr, ok := item.([]any)
			if !ok || len(itemArr) < 3 {
				continue
			}

			id, ok := itemArr[1].(string)
			if !ok || id != rpcid {
				continue
			}

			payloadStr, ok := itemArr[2].(string)
			if !ok {
				continue
			}

			var payload any
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				return nil, fmt.Errorf("parse %s payload: %w", rpcid, err)
			}
			return payload, nil
		}
	}

	// Fallback: try treating entries directly as the batch array.
	for _, entry := range entries {
		arr, ok := entry.([]any)
		if !ok || len(arr) < 3 {
			continue
		}
		id, ok := arr[1].(string)
		if !ok || id != rpcid {
			continue
		}
		payloadStr, ok := arr[2].(string)
		if !ok {
			continue
		}
		var payload any
		if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
			return nil, fmt.Errorf("parse %s payload: %w", rpcid, err)
		}
		return payload, nil
	}

	return nil, fmt.Errorf("no response found for rpcid %s", rpcid)
}

// parseHotelsFromPayload extracts hotels from the AtySUc response payload.
// It searches the nested map/array structure for hotel entries.
func parseHotelsFromPayload(payload any, currency string) ([]models.HotelResult, error) {
	var hotels []models.HotelResult

	// Search through the nested structure for hotel entries.
	found := findHotelEntries(payload, 0)
	for _, h := range found {
		hotel := parseHotelFromMapEntry(h, currency)
		if hotel.Name != "" {
			hotels = append(hotels, hotel)
		}
	}

	if len(hotels) == 0 {
		return nil, fmt.Errorf("no hotels found in response payload")
	}

	return hotels, nil
}

// findHotelEntries recursively searches for arrays that look like organic
// hotel entries (27-element arrays with name at [1] and coordinates at [2]).
func findHotelEntries(v any, depth int) [][]any {
	if depth > 10 {
		return nil
	}

	switch val := v.(type) {
	case []any:
		// Check if this looks like a hotel entry (name at [1], coords at [2]).
		if len(val) > 10 && val[0] == nil {
			if name, ok := val[1].(string); ok && len(name) > 2 {
				if locArr, ok := val[2].([]any); ok && len(locArr) > 0 {
					if coords, ok := locArr[0].([]any); ok && len(coords) == 2 {
						if _, ok := coords[0].(float64); ok {
							return [][]any{val}
						}
					}
				}
			}
		}
		// Recurse into sub-arrays.
		var results [][]any
		for _, item := range val {
			found := findHotelEntries(item, depth+1)
			results = append(results, found...)
		}
		return results

	case map[string]any:
		var results [][]any
		for _, mv := range val {
			found := findHotelEntries(mv, depth+1)
			results = append(results, found...)
		}
		return results
	}

	return nil
}

// parseHotelFromMapEntry parses a hotel from the organic hotel array format.
func parseHotelFromMapEntry(entry []any, currency string) models.HotelResult {
	return parseOrganicHotel(entry, currency)
}

// parseHotelsFromRaw tries to extract hotels from raw decoded entries.
func parseHotelsFromRaw(entries []any, currency string) ([]models.HotelResult, error) {
	var hotels []models.HotelResult
	for _, entry := range entries {
		found := findHotelEntries(entry, 0)
		for _, h := range found {
			hotel := parseHotelFromMapEntry(h, currency)
			if hotel.Name != "" {
				hotels = append(hotels, hotel)
			}
		}
	}
	if len(hotels) == 0 {
		return nil, fmt.Errorf("no hotels found in raw response")
	}
	return hotels, nil
}
