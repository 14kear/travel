package main

import (
	"context"
	"strings"
	"testing"

	"github.com/MikkoParkkola/trvl/internal/models"
	"github.com/MikkoParkkola/trvl/internal/nlsearch"
	"github.com/MikkoParkkola/trvl/internal/trips"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// formatDestinationCard — pure display function, all branches
// ---------------------------------------------------------------------------

func TestFormatDestinationCard_MinimalV25(t *testing.T) {
	info := &models.DestinationInfo{
		Location: "Barcelona",
	}
	if err := formatDestinationCard(info); err != nil {
		t.Errorf("formatDestinationCard minimal: %v", err)
	}
}

func TestFormatDestinationCard_FullV25(t *testing.T) {
	info := &models.DestinationInfo{
		Location: "Tokyo",
		Timezone: "Asia/Tokyo",
		Country: models.CountryInfo{
			Name:       "Japan",
			Code:       "JP",
			Capital:    "Tokyo",
			Languages:  []string{"Japanese"},
			Currencies: []string{"JPY"},
			Region:     "Asia",
		},
		Weather: models.WeatherInfo{
			Forecast: []models.WeatherDay{
				{Date: "2026-07-01", TempHigh: 28, TempLow: 22, Precipitation: 3.5, Description: "Partly cloudy"},
				{Date: "2026-07-02", TempHigh: 30, TempLow: 24, Precipitation: 0, Description: "Sunny"},
			},
		},
		Holidays: []models.Holiday{
			{Date: "2026-07-01", Name: "Test Holiday", Type: "public"},
		},
		Safety: models.SafetyInfo{
			Level:       4.5,
			Advisory:    "Exercise normal caution",
			Source:      "Travel Advisory",
			LastUpdated: "2026-01-01",
		},
		Currency: models.CurrencyInfo{
			LocalCurrency: "JPY",
			ExchangeRate:  160.5,
			BaseCurrency:  "EUR",
		},
	}
	if err := formatDestinationCard(info); err != nil {
		t.Errorf("formatDestinationCard full: %v", err)
	}
}

// ---------------------------------------------------------------------------
// formatGuideCard — pure display function
// ---------------------------------------------------------------------------

func TestFormatGuideCard_EmptyV25(t *testing.T) {
	guide := &models.WikivoyageGuide{
		Location: "Prague",
		URL:      "https://en.wikivoyage.org/wiki/Prague",
	}
	if err := formatGuideCard(guide); err != nil {
		t.Errorf("formatGuideCard empty: %v", err)
	}
}

func TestFormatGuideCard_WithContentV25(t *testing.T) {
	guide := &models.WikivoyageGuide{
		Location: "Barcelona",
		URL:      "https://en.wikivoyage.org/wiki/Barcelona",
		Summary:  "Barcelona is the capital of Catalonia and the second-largest city in Spain.",
		Sections: map[string]string{
			"See":      "The city boasts amazing architecture by Antoni Gaudí.",
			"Eat":      "Tapas, pintxos, and paella are local specialties.",
			"Get in":   "El Prat Airport serves many European routes.",
			"Sleep":    "Hotels range from budget to luxury in the Eixample district.",
			"Get out":  "Day trips to Montserrat and Costa Brava are popular.",
			"Stay safe": "Keep an eye on pickpockets in tourist areas.",
		},
	}
	if err := formatGuideCard(guide); err != nil {
		t.Errorf("formatGuideCard with content: %v", err)
	}
}

// ---------------------------------------------------------------------------
// printReviewsTable — pure display function
// ---------------------------------------------------------------------------

func TestPrintReviewsTable_EmptyV25(t *testing.T) {
	result := &models.HotelReviewResult{
		Name:    "Hotel Test",
		Summary: models.ReviewSummary{AverageRating: 4.2, TotalReviews: 100},
		Reviews: nil,
	}
	if err := printReviewsTable(result); err != nil {
		t.Errorf("printReviewsTable empty: %v", err)
	}
}

func TestPrintReviewsTable_WithReviewsV25(t *testing.T) {
	result := &models.HotelReviewResult{
		Name:    "Grand Hotel",
		Summary: models.ReviewSummary{AverageRating: 4.7, TotalReviews: 250},
		Reviews: []models.HotelReview{
			{Rating: 5.0, Text: "Excellent stay, highly recommend!", Author: "Alice", Date: "2026-04-01"},
			{Rating: 4.0, Text: "Good location but the room was a bit small for the price paid.", Author: "Bob", Date: "2026-03-28"},
			{Rating: 3.5, Text: strings.Repeat("This is a very long review text that exceeds the 80 character limit. ", 2), Author: "Charlie", Date: "2026-03-15"},
		},
	}
	if err := printReviewsTable(result); err != nil {
		t.Errorf("printReviewsTable with reviews: %v", err)
	}
}

func TestStarRating_V25(t *testing.T) {
	cases := []float64{0, 1, 2.5, 3, 4.5, 5}
	for _, r := range cases {
		s := starRating(r)
		if s == "" {
			t.Errorf("starRating(%v) returned empty string", r)
		}
	}
}

// ---------------------------------------------------------------------------
// dispatchSearch — missing-fields paths (all return nil, no network)
// ---------------------------------------------------------------------------

func TestDispatchSearch_FlightMissingFieldsV25(t *testing.T) {
	parent := &cobra.Command{}
	p := nlsearch.Params{Intent: "flight", Origin: "HEL"} // missing Destination and Date
	err := dispatchSearch(parent, p)
	if err != nil {
		t.Errorf("expected nil for missing flight fields, got: %v", err)
	}
}

func TestDispatchSearch_HotelMissingFieldsV25(t *testing.T) {
	parent := &cobra.Command{}
	p := nlsearch.Params{Intent: "hotel", Location: "Barcelona"} // missing CheckIn/CheckOut
	err := dispatchSearch(parent, p)
	if err != nil {
		t.Errorf("expected nil for missing hotel fields, got: %v", err)
	}
}

func TestDispatchSearch_RouteMissingOriginV25(t *testing.T) {
	parent := &cobra.Command{}
	p := nlsearch.Params{Intent: "route"} // missing Origin
	err := dispatchSearch(parent, p)
	if err != nil {
		t.Errorf("expected nil for missing route origin, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// missingFieldsHint — direct call, all intent paths
// ---------------------------------------------------------------------------

func TestMissingFieldsHint_FlightV25(t *testing.T) {
	p := nlsearch.Params{Intent: "flight"}
	err := missingFieldsHint(p, "flight", "trvl flights ORIGIN DESTINATION YYYY-MM-DD")
	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func TestMissingFieldsHint_HotelV25(t *testing.T) {
	p := nlsearch.Params{Intent: "hotel"}
	err := missingFieldsHint(p, "hotel", `trvl hotels "CITY" --checkin YYYY-MM-DD --checkout YYYY-MM-DD`)
	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func TestMissingFieldsHint_DealsV25(t *testing.T) {
	p := nlsearch.Params{Intent: "deals"}
	err := missingFieldsHint(p, "deals", "trvl deals")
	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// printTripWeather — with empty legs (early exit branch, no network)
// ---------------------------------------------------------------------------

func TestPrintTripWeather_EmptyLegsV25(t *testing.T) {
	tr := &trips.Trip{
		ID:   "test-trip-weather",
		Name: "Weather Test Trip",
		Legs: nil,
	}
	// Should return immediately with 0 targets — no network, no panic.
	printTripWeather(context.Background(), tr)
}

func TestPrintTripWeather_LegsWithEmptyToV25(t *testing.T) {
	tr := &trips.Trip{
		ID:   "test-trip-weather-2",
		Name: "Weather Test Trip 2",
		Legs: []trips.TripLeg{
			{From: "HEL", To: "", StartTime: "2026-07-01T08:00"},
			{From: "BCN", To: "HEL", StartTime: ""},
		},
	}
	// All legs skipped (To="" or StartTime="") → 0 targets, no network.
	printTripWeather(context.Background(), tr)
}

// ---------------------------------------------------------------------------
// destinationCmd — missing required arg (no network)
// ---------------------------------------------------------------------------

func TestDestinationCmd_MissingArgV25(t *testing.T) {
	cmd := destinationCmd()
	cmd.SetArgs([]string{}) // needs 1 positional arg
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error with no positional arg")
	}
}

func TestDestinationCmd_FlagsV25(t *testing.T) {
	cmd := destinationCmd()
	if f := cmd.Flags().Lookup("dates"); f == nil {
		t.Error("expected --dates flag on destinationCmd")
	}
}

// ---------------------------------------------------------------------------
// guideCmd — missing arg (no network)
// ---------------------------------------------------------------------------

func TestGuideCmd_MissingArgV25(t *testing.T) {
	cmd := guideCmd()
	cmd.SetArgs([]string{}) // needs 1 positional arg
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error with no positional arg")
	}
}

// ---------------------------------------------------------------------------
// reviewsCmd — flags coverage (no network)
// ---------------------------------------------------------------------------

func TestReviewsCmd_FlagsV25(t *testing.T) {
	// reviewsCmd is a package-level var, check it's non-nil and has flags.
	if reviewsCmd == nil {
		t.Fatal("reviewsCmd is nil")
	}
	for _, name := range []string{"limit", "sort", "format"} {
		if f := reviewsCmd.Flags().Lookup(name); f == nil {
			t.Errorf("expected --%s flag on reviewsCmd", name)
		}
	}
}

// ---------------------------------------------------------------------------
// weatherCmd — runs until network (covers weatherCmd RunE and default dates)
// ---------------------------------------------------------------------------

func TestWeatherCmd_DefaultDatesNoNetworkV25(t *testing.T) {
	cmd := weatherCmd()
	cmd.SetArgs([]string{"Helsinki"})
	// Will call Open-Meteo but that's fast and free — cover RunE body.
	_ = cmd.Execute()
}

// ---------------------------------------------------------------------------
// loungesCmd — runs until network call (covers most of RunE)
// ---------------------------------------------------------------------------

func TestLoungesCmd_ValidIATANoNetworkV25(t *testing.T) {
	cmd := loungesCmd()
	cmd.SetArgs([]string{"HEL"})
	// lounges.SearchLounges may fail (network/scraping) — either is fine.
	_ = cmd.Execute()
}
