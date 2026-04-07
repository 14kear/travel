package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// searchNaturalTool returns the MCP tool definition for natural-language search.
func searchNaturalTool() ToolDef {
	return ToolDef{
		Name:  "search_natural",
		Title: "Natural Language Travel Search",
		Description: "Accept a free-form travel query in plain language and dispatch to the " +
			"appropriate search tool. Examples: " +
			"\"cheapest way from Helsinki to Dubrovnik next weekend\", " +
			"\"hotels in Prague for 3 nights in July under EUR 120\", " +
			"\"I want to explore a Croatian island, budget EUR 500, long weekend next month\". " +
			"Requires the AI client to support the MCP sampling capability; " +
			"falls back to a best-effort parse if sampling is unavailable.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"query": {
					Type:        "string",
					Description: "Natural language travel request",
				},
			},
			Required: []string{"query"},
		},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"intent":      map[string]interface{}{"type": "string"},
				"result":      map[string]interface{}{"type": "object"},
				"query":       map[string]interface{}{"type": "string"},
				"dispatched_to": map[string]interface{}{"type": "string"},
			},
		},
		Annotations: &ToolAnnotations{
			Title:          "Natural Language Travel Search",
			ReadOnlyHint:   true,
			OpenWorldHint:  true,
			IdempotentHint: true,
		},
	}
}

// naturalSearchParams holds the structured parameters extracted from a free-form query.
type naturalSearchParams struct {
	Intent        string   `json:"intent"`         // "route", "flight", "hotel", "deals"
	Origin        string   `json:"origin"`         // IATA or city; empty if not mentioned
	Destination   string   `json:"destination"`    // IATA or city
	Date          string   `json:"date"`           // YYYY-MM-DD or empty
	ReturnDate    string   `json:"return_date"`    // YYYY-MM-DD or empty
	CheckIn       string   `json:"check_in"`       // YYYY-MM-DD (hotels)
	CheckOut      string   `json:"check_out"`      // YYYY-MM-DD (hotels)
	MaxBudget     float64  `json:"max_budget"`     // 0 = unlimited
	TravelerCount int      `json:"traveler_count"` // 0 = unspecified (default 1 or 2)
	Modes         []string `json:"transport_modes"`// "flight", "train", "bus", "ferry"
	Location      string   `json:"location"`       // hotel location when intent=hotel
}

// extractionPrompt builds the LLM prompt used to parse a free-form query.
func extractionPrompt(query string, today string) string {
	return fmt.Sprintf(`Extract travel search parameters from this query: %q

Today's date is %s.

Return ONLY a JSON object with these fields (omit fields you cannot determine):
{
  "intent": "route"|"flight"|"hotel"|"deals",
  "origin": "IATA code or city name or null",
  "destination": "IATA code or city name or null",
  "date": "YYYY-MM-DD or null",
  "return_date": "YYYY-MM-DD or null",
  "check_in": "YYYY-MM-DD or null",
  "check_out": "YYYY-MM-DD or null",
  "max_budget": number or null,
  "traveler_count": integer or null,
  "transport_modes": ["flight","train","bus","ferry"] or null,
  "location": "city or neighborhood for hotel search or null"
}

Rules:
- Resolve relative dates ("next weekend", "next month", "this Friday") to ISO dates using today's date.
- "next weekend" = the coming Saturday.
- "long weekend" = 3 nights (Friday to Monday).
- If origin is not mentioned, use null.
- If intent is hotel, set check_in/check_out; if it is route or flight, set date.
- intent=route when query mentions trains, buses, ferries, or multi-modal travel.
- intent=flight when query mentions flying or airports explicitly.
- intent=hotel when query asks about accommodation, hotels, or places to stay.
- intent=deals when query asks for general deals or inspiration without a fixed destination.
- Return only the JSON object, no explanation.`, query, today)
}

// handleSearchNatural handles the search_natural tool.
func handleSearchNatural(args map[string]any, elicit ElicitFunc, sampling SamplingFunc, progress ProgressFunc) ([]ContentBlock, interface{}, error) {
	query := strings.TrimSpace(argString(args, "query"))
	if query == "" {
		return nil, nil, fmt.Errorf("query is required")
	}

	sendProgress(progress, 0, 100, "Parsing travel query...")

	today := time.Now().Format("2006-01-02")
	var params naturalSearchParams

	if sampling != nil {
		// Use AI sampling to extract structured parameters from the free-form query.
		prompt := extractionPrompt(query, today)
		response, err := sampling([]SamplingMessage{
			{Role: "user", Content: SamplingContent{Type: "text", Text: prompt}},
		}, 500)
		if err == nil && response != "" {
			// Strip markdown code fences if the model adds them.
			cleaned := strings.TrimSpace(response)
			if idx := strings.Index(cleaned, "{"); idx > 0 {
				cleaned = cleaned[idx:]
			}
			if idx := strings.LastIndex(cleaned, "}"); idx >= 0 && idx < len(cleaned)-1 {
				cleaned = cleaned[:idx+1]
			}
			_ = json.Unmarshal([]byte(cleaned), &params)
		}
	}

	// Fallback: best-effort heuristic parse when sampling is nil or failed.
	if params.Destination == "" && params.Location == "" {
		params = heuristicParse(query, today)
	}

	sendProgress(progress, 30, 100, fmt.Sprintf("Dispatching %s search...", params.Intent))

	// Dispatch to the appropriate handler.
	return dispatchNatural(params, query, elicit, sampling, progress)
}

// dispatchNatural routes parsed params to the right tool handler.
func dispatchNatural(p naturalSearchParams, originalQuery string, elicit ElicitFunc, sampling SamplingFunc, progress ProgressFunc) ([]ContentBlock, interface{}, error) {
	switch p.Intent {
	case "hotel":
		if p.Location == "" && p.Destination != "" {
			p.Location = p.Destination
		}
		if p.Location == "" {
			return []ContentBlock{{Type: "text", Text: "Could not determine hotel location from your query. Please specify a city."}}, nil, nil
		}
		if p.CheckIn == "" || p.CheckOut == "" {
			return []ContentBlock{{Type: "text", Text: "Could not determine check-in or check-out dates from your query. Please specify dates."}}, nil, nil
		}
		hotelArgs := map[string]any{
			"location":  p.Location,
			"check_in":  p.CheckIn,
			"check_out": p.CheckOut,
		}
		if p.TravelerCount > 0 {
			hotelArgs["guests"] = p.TravelerCount
		}
		if p.MaxBudget > 0 {
			hotelArgs["max_price"] = p.MaxBudget
		}
		return handleSearchHotels(hotelArgs, elicit, sampling, progress)

	case "flight":
		if p.Origin == "" || p.Destination == "" || p.Date == "" {
			return []ContentBlock{{Type: "text", Text: "Could not determine origin, destination, or date for the flight search. Please specify them."}}, nil, nil
		}
		flightArgs := map[string]any{
			"origin":         p.Origin,
			"destination":    p.Destination,
			"departure_date": p.Date,
		}
		if p.ReturnDate != "" {
			flightArgs["return_date"] = p.ReturnDate
		}
		return handleSearchFlights(flightArgs, elicit, sampling, progress)

	case "route":
		if p.Origin == "" || p.Destination == "" || p.Date == "" {
			return []ContentBlock{{Type: "text", Text: "Could not determine origin, destination, or date for the route search. Please specify them."}}, nil, nil
		}
		routeArgs := map[string]any{
			"origin":      p.Origin,
			"destination": p.Destination,
			"date":        p.Date,
		}
		if p.MaxBudget > 0 {
			routeArgs["max_price"] = p.MaxBudget
		}
		if len(p.Modes) > 0 {
			// If user only wants trains/buses, avoid flights.
			wantsFlights := false
			for _, m := range p.Modes {
				if m == "flight" {
					wantsFlights = true
				}
			}
			if !wantsFlights {
				routeArgs["avoid"] = "flight"
			}
		}
		return handleSearchRoute(routeArgs, elicit, sampling, progress)

	default:
		// Fallback: return a helpful message describing what we parsed.
		msg := fmt.Sprintf("Interpreted your query as: %q\n\n", originalQuery)
		if p.Origin != "" {
			msg += fmt.Sprintf("From: %s\n", p.Origin)
		}
		if p.Destination != "" {
			msg += fmt.Sprintf("To: %s\n", p.Destination)
		}
		if p.Date != "" {
			msg += fmt.Sprintf("Date: %s\n", p.Date)
		}
		msg += "\nCould not determine the search intent. Try search_flights, search_route, or search_hotels with explicit parameters."
		return []ContentBlock{{Type: "text", Text: msg}}, nil, nil
	}
}

// heuristicParse provides a minimal keyword-based fallback when sampling is
// unavailable. It is intentionally simple — the sampling path is strongly
// preferred.
func heuristicParse(query, today string) naturalSearchParams {
	lower := strings.ToLower(query)
	p := naturalSearchParams{Intent: "route"}

	// Detect intent using explicit keyword checks (not ContainsAny which is char-based).
	switch {
	case strings.Contains(lower, "hotel") || strings.Contains(lower, "hostel") ||
		strings.Contains(lower, "accommodation") || strings.Contains(lower, "stay") ||
		strings.Contains(lower, "sleep") || strings.Contains(lower, "room") ||
		strings.Contains(lower, "check-in") || strings.Contains(lower, "check in"):
		p.Intent = "hotel"
	case strings.Contains(lower, "fly ") || strings.Contains(lower, "flying") ||
		strings.Contains(lower, "flight") || strings.Contains(lower, "airport"):
		p.Intent = "flight"
	case strings.Contains(lower, "deal") || strings.Contains(lower, "inspiration"):
		p.Intent = "deals"
	}

	// Resolve "next weekend" — the simplest relative date.
	if strings.Contains(lower, "next weekend") || strings.Contains(lower, "this weekend") {
		t, _ := time.Parse("2006-01-02", today)
		// Advance to next Saturday.
		daysUntilSat := (6 - int(t.Weekday()) + 7) % 7
		if daysUntilSat == 0 {
			daysUntilSat = 7
		}
		sat := t.AddDate(0, 0, daysUntilSat)
		mon := sat.AddDate(0, 0, 2)
		p.Date = sat.Format("2006-01-02")
		p.CheckIn = p.Date
		p.CheckOut = mon.Format("2006-01-02")
	}

	return p
}
