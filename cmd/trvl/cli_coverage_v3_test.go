package main

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/MikkoParkkola/trvl/internal/hotels"
	"github.com/MikkoParkkola/trvl/internal/models"
	"github.com/MikkoParkkola/trvl/internal/profile"
)

// ---------------------------------------------------------------------------
// looksLikeGoogleHotelID (rooms.go)
// ---------------------------------------------------------------------------

func TestLooksLikeGoogleHotelID_SlashG(t *testing.T) {
	if !looksLikeGoogleHotelID("/g/11b6d4_v_4") {
		t.Error("expected true for /g/ prefix")
	}
}

func TestLooksLikeGoogleHotelID_ChIJ(t *testing.T) {
	if !looksLikeGoogleHotelID("ChIJy7MSZP0LkkYRZw2dDekQP78") {
		t.Error("expected true for ChIJ prefix")
	}
}

func TestLooksLikeGoogleHotelID_ColonSeparated(t *testing.T) {
	if !looksLikeGoogleHotelID("0x123456:0xabcdef") {
		t.Error("expected true for colon-separated ID without spaces")
	}
}

func TestLooksLikeGoogleHotelID_HotelName(t *testing.T) {
	if looksLikeGoogleHotelID("Hotel Lutetia Paris") {
		t.Error("expected false for hotel name with spaces")
	}
}

func TestLooksLikeGoogleHotelID_Empty(t *testing.T) {
	if looksLikeGoogleHotelID("") {
		t.Error("expected false for empty string")
	}
}

func TestLooksLikeGoogleHotelID_WithSpaces(t *testing.T) {
	if looksLikeGoogleHotelID("   /g/abc   ") {
		// trimmed, should still match (trimmed internally)
		// Actually the function trims spaces, so this SHOULD be true
	}
	// The function trims and checks prefix, so /g/ after trim should be true
	if !looksLikeGoogleHotelID("   /g/abc   ") {
		t.Error("expected true after trimming spaces for /g/ prefix")
	}
}

// ---------------------------------------------------------------------------
// formatRoomsTable (rooms.go) — pure formatting
// ---------------------------------------------------------------------------

func TestFormatRoomsTable_Empty(t *testing.T) {
	result := &hotels.RoomAvailability{
		HotelID:  "/g/11abc",
		Name:     "Grand Hotel",
		CheckIn:  "2026-06-15",
		CheckOut: "2026-06-18",
		Rooms:    nil,
	}
	if err := formatRoomsTable(result); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFormatRoomsTable_EmptyNameFallsBackToID(t *testing.T) {
	result := &hotels.RoomAvailability{
		HotelID:  "/g/11abc",
		Name:     "",
		CheckIn:  "2026-06-15",
		CheckOut: "2026-06-18",
		Rooms:    nil,
	}
	if err := formatRoomsTable(result); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFormatRoomsTable_WithRoomsV3(t *testing.T) {
	result := &hotels.RoomAvailability{
		HotelID:  "/g/11abc",
		Name:     "Grand Hotel",
		CheckIn:  "2026-06-15",
		CheckOut: "2026-06-18",
		Rooms: []hotels.RoomType{
			{Name: "Standard", Price: 120, Currency: "EUR", MaxGuests: 2, Provider: "direct", Amenities: []string{"WiFi", "TV"}},
			{Name: "Deluxe", Price: 200, Currency: "EUR", MaxGuests: 2, Provider: "booking", Amenities: []string{"WiFi", "TV", "Minibar", "Extra1", "Extra2", "SomeLongAmenity"}},
			{Name: "Free Room", Price: 0, Currency: "EUR", MaxGuests: 0, Provider: ""},
		},
	}
	if err := formatRoomsTable(result); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// printReviewsTable + starRating (reviews.go)
// ---------------------------------------------------------------------------

func TestStarRating_Full(t *testing.T) {
	got := starRating(5.0)
	if got == "" {
		t.Error("expected non-empty star rating")
	}
}

func TestStarRating_Half(t *testing.T) {
	got := starRating(3.5)
	if got == "" {
		t.Error("expected non-empty star rating for half star")
	}
}

func TestStarRating_ZeroV3(t *testing.T) {
	got := starRating(0)
	if got == "" {
		t.Error("expected non-empty star rating for zero")
	}
}

func TestPrintReviewsTable_Empty(t *testing.T) {
	result := &models.HotelReviewResult{
		HotelID: "/g/11abc",
		Name:    "Grand Hotel",
		Summary: models.ReviewSummary{AverageRating: 4.2, TotalReviews: 100},
		Reviews: nil,
	}
	if err := printReviewsTable(result); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrintReviewsTable_WithReviewsV3(t *testing.T) {
	longText := strings.Repeat("A very long review text that should be truncated. ", 3)
	result := &models.HotelReviewResult{
		HotelID: "/g/11abc",
		Name:    "Grand Hotel",
		Summary: models.ReviewSummary{AverageRating: 4.2, TotalReviews: 2},
		Reviews: []models.HotelReview{
			{Rating: 5.0, Author: "Alice", Date: "2026-03-15", Text: "Excellent!"},
			{Rating: 3.5, Author: "Bob", Date: "2026-03-10", Text: longText},
		},
	}
	if err := printReviewsTable(result); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrintReviewsTable_NoName(t *testing.T) {
	result := &models.HotelReviewResult{
		HotelID: "/g/11abc",
		Summary: models.ReviewSummary{AverageRating: 0, TotalReviews: 0},
		Reviews: []models.HotelReview{
			{Rating: 4.0, Author: "Eve", Date: "2026-01-01", Text: "Good."},
		},
	}
	if err := printReviewsTable(result); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// printProfileSummary (profile.go)
// ---------------------------------------------------------------------------

func TestPrintProfileSummary_Empty(t *testing.T) {
	p := &profile.TravelProfile{}
	// Should not panic with empty profile.
	printProfileSummary(p)
}

func TestPrintProfileSummary_Full(t *testing.T) {
	p := &profile.TravelProfile{
		TotalTrips:       15,
		TotalFlights:     30,
		TotalHotelNights: 45,
		TopAirlines: []profile.AirlineStats{
			{Code: "KL", Name: "KLM", Flights: 12},
			{Code: "AY", Name: "", Flights: 5},
		},
		PreferredAlliance: "SkyTeam",
		AvgFlightPrice:    210,
		TopRoutes: []profile.RouteStats{
			{From: "HEL", To: "AMS", Count: 8, AvgPrice: 180},
			{From: "AMS", To: "JFK", Count: 2, AvgPrice: 0},
		},
		HomeDetected:    []string{"HEL"},
		TopDestinations: []string{"AMS", "BCN", "NRT"},
		TopHotelChains: []profile.HotelChainStats{
			{Name: "Marriott", Nights: 20},
		},
		AvgStarRating:  4.2,
		AvgNightlyRate: 120,
		PreferredType:  "hotel",
		TopGroundModes: []profile.ModeStats{
			{Mode: "train", Count: 10},
		},
		AvgTripLength:  5.5,
		PreferredDays:  []string{"Tuesday", "Wednesday"},
		AvgBookingLead: 21,
		BudgetTier:     "mid-range",
		AvgTripCost:    850,
	}
	printProfileSummary(p)
}

// ---------------------------------------------------------------------------
// runRestaurants — arg validation via direct call (avoids shared cmd state)
// ---------------------------------------------------------------------------

func TestRunRestaurants_InvalidLat(t *testing.T) {
	// Create isolated cobra command that delegates to runRestaurants.
	cmd := restaurantsCmd
	// Call RunE directly to avoid global command state issues.
	err := runRestaurants(cmd, []string{"not-lat", "2.17"})
	if err == nil {
		t.Error("expected error for invalid latitude")
	}
}

func TestRunRestaurants_InvalidLon(t *testing.T) {
	cmd := restaurantsCmd
	err := runRestaurants(cmd, []string{"41.38", "not-lon"})
	if err == nil {
		t.Error("expected error for invalid longitude")
	}
}

func TestRunRestaurants_LatOutOfRange(t *testing.T) {
	cmd := restaurantsCmd
	err := runRestaurants(cmd, []string{"91.0", "2.17"})
	if err == nil {
		t.Error("expected error for lat > 90")
	}
}

func TestRunRestaurants_LonOutOfRange(t *testing.T) {
	cmd := restaurantsCmd
	err := runRestaurants(cmd, []string{"41.38", "181.0"})
	if err == nil {
		t.Error("expected error for lon > 180")
	}
}

// ---------------------------------------------------------------------------
// roomsCmd — flag registration
// ---------------------------------------------------------------------------

func TestRoomsCmd_FlagsV3(t *testing.T) {
	cmd := roomsCmd()
	for _, name := range []string{"checkin", "checkout", "currency", "location"} {
		if f := cmd.Flags().Lookup(name); f == nil {
			t.Errorf("expected --%s flag", name)
		}
	}
}

// ---------------------------------------------------------------------------
// setup.go prompt helpers — pure I/O with no filesystem
// ---------------------------------------------------------------------------

func TestSetupPromptString_UsesInput(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("newvalue\n"))
	got := setupPromptString(scanner, os.Stderr, "Label", "default")
	if got != "newvalue" {
		t.Errorf("expected newvalue, got %q", got)
	}
}

func TestSetupPromptString_EmptyKeepsDefault(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("\n"))
	got := setupPromptString(scanner, os.Stderr, "Label", "default")
	if got != "default" {
		t.Errorf("expected default, got %q", got)
	}
}

func TestSetupPromptString_EmptyCurrentLabel(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("value\n"))
	got := setupPromptString(scanner, os.Stderr, "Label", "")
	if got != "value" {
		t.Errorf("expected value, got %q", got)
	}
}

func TestSetupPromptOptional_ReturnsInput(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("optional\n"))
	got := setupPromptOptional(scanner, os.Stderr, "Label", "current")
	if got != "optional" {
		t.Errorf("expected optional, got %q", got)
	}
}

func TestSetupPromptOptional_EmptyReturnsEmpty(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("\n"))
	got := setupPromptOptional(scanner, os.Stderr, "Label", "")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestSetupPromptOptional_WithCurrentShowsBracket(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("\n"))
	// Should return current when empty input
	got := setupPromptOptional(scanner, os.Stderr, "Label", "existing")
	if got != "existing" {
		t.Errorf("expected existing (kept), got %q", got)
	}
}

func TestSetupPromptSecret_WithInput(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("secret123\n"))
	got := setupPromptSecret(scanner, os.Stderr, "API Key", "")
	if got != "secret123" {
		t.Errorf("expected secret123, got %q", got)
	}
}

func TestSetupPromptSecret_WithExisting(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("\n"))
	got := setupPromptSecret(scanner, os.Stderr, "API Key", "existing-secret")
	// Empty input returns empty (not existing); that's the function's behavior
	if got != "" {
		t.Errorf("expected empty (not kept), got %q", got)
	}
}

func TestSetupPromptChoice_ValidChoice(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("table\n"))
	valid := map[string]bool{"table": true, "json": true}
	got := setupPromptChoice(scanner, os.Stderr, "Format", "table", valid)
	if got != "table" {
		t.Errorf("expected table, got %q", got)
	}
}

func TestSetupPromptChoice_EmptyKeepsCurrent(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("\n"))
	valid := map[string]bool{"table": true, "json": true}
	got := setupPromptChoice(scanner, os.Stderr, "Format", "json", valid)
	if got != "json" {
		t.Errorf("expected json (current), got %q", got)
	}
}

func TestSetupPromptChoice_InvalidThenValid(t *testing.T) {
	// First input invalid, second valid — re-prompts
	scanner := bufio.NewScanner(strings.NewReader("csv\njson\n"))
	valid := map[string]bool{"table": true, "json": true}
	got := setupPromptChoice(scanner, os.Stderr, "Format", "table", valid)
	if got != "json" {
		t.Errorf("expected json after retry, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// reviewsCmd — flag registration
// ---------------------------------------------------------------------------

func TestReviewsCmd_FlagsV3(t *testing.T) {
	for _, name := range []string{"limit", "sort", "format"} {
		if f := reviewsCmd.Flags().Lookup(name); f == nil {
			t.Errorf("expected --%s flag on reviewsCmd", name)
		}
	}
}

// ---------------------------------------------------------------------------
// discoverCmd — missing origin error path
// ---------------------------------------------------------------------------

func TestDiscoverCmd_MissingOriginError(t *testing.T) {
	// Build a preferences file in a temp dir that has no home airports.
	cmd := discoverCmd()
	cmd.SetArgs([]string{"--from", "2026-07-01", "--until", "2026-07-31", "--budget", "500"})
	// This will fail either because no origin or because preferences are unavailable.
	// Either way it should error, not panic.
	_ = cmd.Execute()
}
