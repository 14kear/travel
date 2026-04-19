package main

import (
	"context"
	"testing"

	"github.com/MikkoParkkola/trvl/internal/models"
)

// ---------------------------------------------------------------------------
// convertRoundedDisplayAmounts (currency_display.go) — nil and zero branches
// ---------------------------------------------------------------------------

func TestConvertRoundedDisplayAmounts_NilAmount(t *testing.T) {
	got := convertRoundedDisplayAmounts(context.Background(), "EUR", "USD", 2, nil)
	if got != "EUR" {
		t.Errorf("expected EUR (no conversion), got %q", got)
	}
}

func TestConvertRoundedDisplayAmounts_ZeroAmount(t *testing.T) {
	zero := 0.0
	got := convertRoundedDisplayAmounts(context.Background(), "EUR", "USD", 2, &zero)
	if got != "EUR" {
		t.Errorf("expected EUR (zero amount skipped), got %q", got)
	}
}

// ---------------------------------------------------------------------------
// printDatesTable (dates.go) — branches not covered elsewhere
// ---------------------------------------------------------------------------

func TestPrintDatesTable_FailedResult(t *testing.T) {
	result := &models.DateSearchResult{
		Success: false,
		Error:   "search failed",
	}
	if err := printDatesTable(context.Background(), "", result); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrintDatesTable_ZeroCount(t *testing.T) {
	result := &models.DateSearchResult{
		Success: true,
		Count:   0,
		Dates:   nil,
	}
	if err := printDatesTable(context.Background(), "", result); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrintDatesTable_OneWayV7(t *testing.T) {
	result := &models.DateSearchResult{
		Success:   true,
		Count:     2,
		DateRange: "2026-07-01 to 2026-07-31",
		TripType:  "one_way",
		Dates: []models.DatePriceResult{
			{Date: "2026-07-05", Price: 89, Currency: "EUR"},
			{Date: "2026-07-12", Price: 75, Currency: "EUR"},
		},
	}
	if err := printDatesTable(context.Background(), "", result); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrintDatesTable_RoundTripV7(t *testing.T) {
	result := &models.DateSearchResult{
		Success:   true,
		Count:     1,
		DateRange: "2026-07-01 to 2026-07-31",
		TripType:  "round_trip",
		Dates: []models.DatePriceResult{
			{Date: "2026-07-05", Price: 299, Currency: "EUR", ReturnDate: "2026-07-12"},
		},
	}
	if err := printDatesTable(context.Background(), "", result); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// printGridTable (grid.go) — empty result
// ---------------------------------------------------------------------------

func TestPrintGridTable_EmptyResult(t *testing.T) {
	result := &models.PriceGrid{
		Success: true,
		Count:   0,
	}
	if err := printGridTable(context.Background(), "", result, "HEL", "BCN"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrintGridTable_FailedResult(t *testing.T) {
	result := &models.PriceGrid{
		Success: false,
		Count:   0,
	}
	if err := printGridTable(context.Background(), "", result, "HEL", "BCN"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// maybeShowFlightHackTips (flights.go) — json format branch (no output)
// ---------------------------------------------------------------------------

func TestMaybeShowFlightHackTips_JSONFormatV7(t *testing.T) {
	result := &models.FlightSearchResult{
		Success: true,
		Flights: []models.FlightResult{
			{Price: 199, Currency: "EUR"},
		},
	}
	// JSON format should suppress hack tips.
	maybeShowFlightHackTips(context.Background(), []string{"HEL"}, []string{"BCN"}, "2026-07-01", "", 1, result)
}

func TestMaybeShowFlightHackTips_EmptyFlightsV7(t *testing.T) {
	result := &models.FlightSearchResult{
		Success: true,
		Flights: nil,
	}
	// No flights — should be a no-op without panic.
	maybeShowFlightHackTips(context.Background(), []string{"HEL"}, []string{"BCN"}, "2026-07-01", "", 1, result)
}

// ---------------------------------------------------------------------------
// printSuggestTable — invalid IATA path (validation before network)
// ---------------------------------------------------------------------------

func TestSuggestCmd_InvalidOriginIATAV7(t *testing.T) {
	cmd := suggestCmd()
	cmd.SetArgs([]string{"12", "BCN", "--around", "2026-07-01"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid origin IATA")
	}
}

func TestSuggestCmd_InvalidDestIATAV7(t *testing.T) {
	cmd := suggestCmd()
	cmd.SetArgs([]string{"HEL", "12", "--around", "2026-07-01"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid dest IATA")
	}
}

// ---------------------------------------------------------------------------
// flightsCmd — home origin path
// ---------------------------------------------------------------------------

func TestFlightsCmd_HomeOriginResolves(t *testing.T) {
	// When "home" is passed as origin with no prefs, it stays unresolved.
	// Since "home" has 4 chars, ParseAirports might interpret it.
	// Just verify no panic.
	cmd := flightsCmd()
	cmd.SetArgs([]string{"home", "BCN", "2026-07-01"})
	_ = cmd.Execute()
}

// ---------------------------------------------------------------------------
// printExploreTable — currency conversion branch (no-op when currency empty)
// ---------------------------------------------------------------------------

func TestPrintExploreTable_WithCityIDOnly(t *testing.T) {
	result := &models.ExploreResult{
		Destinations: []models.ExploreDestination{
			{CityID: "city:BCN", CityName: "Barcelona", Country: "Spain", Price: 89, Stops: 0},
		},
		Count: 1,
	}
	if err := printExploreTable(context.Background(), "", result, "HEL"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
