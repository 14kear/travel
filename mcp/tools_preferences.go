package mcp

import (
	"fmt"

	"github.com/MikkoParkkola/trvl/internal/preferences"
)

// getPreferencesTool returns the MCP tool definition for get_preferences.
func getPreferencesTool() ToolDef {
	return ToolDef{
		Name:        "get_preferences",
		Title:       "Get User Preferences",
		Description: "Returns the user's personal travel preferences including home airports, accommodation requirements, currency, and loyalty programmes. Use this to personalise search results before calling search_hotels or search_flights.",
		InputSchema: InputSchema{
			Type:       "object",
			Properties: map[string]Property{},
			Required:   []string{},
		},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"home_airports":       map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				"home_cities":         map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				"carry_on_only":       map[string]interface{}{"type": "boolean"},
				"prefer_direct":       map[string]interface{}{"type": "boolean"},
				"no_dormitories":      map[string]interface{}{"type": "boolean"},
				"ensuite_only":        map[string]interface{}{"type": "boolean"},
				"fast_wifi_needed":    map[string]interface{}{"type": "boolean"},
				"min_hotel_stars":     map[string]interface{}{"type": "integer"},
				"min_hotel_rating":    map[string]interface{}{"type": "number"},
				"display_currency":    map[string]interface{}{"type": "string"},
				"locale":              map[string]interface{}{"type": "string"},
				"loyalty_airlines":    map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				"loyalty_hotels":      map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				"preferred_districts": map[string]interface{}{"type": "object"},
				"family_members": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name":         map[string]interface{}{"type": "string"},
							"relationship": map[string]interface{}{"type": "string"},
							"notes":        map[string]interface{}{"type": "string"},
						},
					},
				},
			},
		},
		Annotations: &ToolAnnotations{
			Title:          "Get User Preferences",
			ReadOnlyHint:   true,
			IdempotentHint: true,
		},
	}
}

// handleGetPreferences returns the user's preferences as structured data.
func handleGetPreferences(args map[string]any, _ ElicitFunc, _ SamplingFunc, progress ProgressFunc) ([]ContentBlock, interface{}, error) {
	p, err := preferences.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("load preferences: %w", err)
	}

	var summary string
	if len(p.HomeAirports) > 0 {
		summary = fmt.Sprintf("Home airports: %v. Display currency: %s.", p.HomeAirports, p.DisplayCurrency)
	} else {
		summary = fmt.Sprintf("No home airports set. Display currency: %s.", p.DisplayCurrency)
	}

	var filters []string
	if p.MinHotelRating > 0 {
		filters = append(filters, fmt.Sprintf("min rating %.1f", p.MinHotelRating))
	}
	if p.MinHotelStars > 0 {
		filters = append(filters, fmt.Sprintf("min %d stars", p.MinHotelStars))
	}
	if p.NoDormitories {
		filters = append(filters, "no dormitories")
	}
	if p.EnSuiteOnly {
		filters = append(filters, "en-suite only")
	}
	if len(filters) > 0 {
		summary += " Hotel filters: " + joinStrings(filters, ", ") + "."
	}

	content, err := buildAnnotatedContentBlocks(summary, p)
	if err != nil {
		return nil, nil, err
	}
	return content, p, nil
}

// joinStrings joins a slice with sep (avoids importing strings in this file).
func joinStrings(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}
