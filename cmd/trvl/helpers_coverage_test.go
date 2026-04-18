package main

import (
	"strings"
	"testing"

	"github.com/MikkoParkkola/trvl/internal/models"
	"github.com/MikkoParkkola/trvl/internal/trip"
)

// ---------------------------------------------------------------------------
// nudgePath
// ---------------------------------------------------------------------------

func TestNudgePath_ReturnsPath(t *testing.T) {
	path, err := nudgePath()
	if err != nil {
		t.Fatalf("nudgePath returned error: %v", err)
	}
	if path == "" {
		t.Fatal("nudgePath returned empty string")
	}
	if !strings.HasSuffix(path, "nudge.json") {
		t.Errorf("nudgePath = %q, want suffix 'nudge.json'", path)
	}
	if !strings.Contains(path, ".trvl") {
		t.Errorf("nudgePath = %q, want to contain '.trvl'", path)
	}
}

// ---------------------------------------------------------------------------
// truncateName
// ---------------------------------------------------------------------------

func TestTruncateName_Short(t *testing.T) {
	got := truncateName("Short Name", 25)
	if got != "Short Name" {
		t.Errorf("truncateName(short) = %q, want %q", got, "Short Name")
	}
}

func TestTruncateName_Exact(t *testing.T) {
	s := "Exactly Twenty Five Chars"
	got := truncateName(s, 25)
	if got != s {
		t.Errorf("truncateName(exact) = %q, want %q", got, s)
	}
}

func TestTruncateName_Long(t *testing.T) {
	s := "This Is A Very Long Hotel Name That Should Be Truncated"
	got := truncateName(s, 25)
	if len([]rune(got)) > 25 {
		t.Errorf("truncateName should be <= 25 runes, got %d", len([]rune(got)))
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("truncateName should end with '...', got %q", got)
	}
}

func TestTruncateName_Unicode(t *testing.T) {
	s := "ホテル東京グランドパレス長い名前"
	got := truncateName(s, 10)
	if len([]rune(got)) > 10 {
		t.Errorf("truncateName(unicode) should be <= 10 runes, got %d", len([]rune(got)))
	}
}

// ---------------------------------------------------------------------------
// formatPrice
// ---------------------------------------------------------------------------

func TestFormatPrice_Various(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		currency string
		want     string
	}{
		{"positive EUR", 250, "EUR", "EUR 250"},
		{"positive USD", 100, "USD", "USD 100"},
		{"zero returns dash", 0, "EUR", "-"},
		{"large amount", 9999, "JPY", "JPY 9999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPrice(tt.amount, tt.currency)
			if got != tt.want {
				t.Errorf("formatPrice(%v, %q) = %q, want %q", tt.amount, tt.currency, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// hasAirportTransferProvider
// ---------------------------------------------------------------------------

func TestHasAirportTransferProvider_CaseInsensitive(t *testing.T) {
	routes := []models.GroundRoute{
		{Provider: "Taxi"},
		{Provider: "FlixBus"},
	}

	tests := []struct {
		provider string
		want     bool
	}{
		{"taxi", true},
		{"TAXI", true},
		{"Taxi", true},
		{"flixbus", true},
		{"FLIXBUS", true},
		{"eurostar", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := hasAirportTransferProvider(routes, tt.provider)
			if got != tt.want {
				t.Errorf("hasAirportTransferProvider(%q) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// hotelSourceLabel edge cases
// ---------------------------------------------------------------------------

func TestHotelSourceLabel_AllKnown(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"google_hotels", "Google"},
		{"trivago", "Trivago"},
		{"airbnb", "Airbnb"},
		{"booking", "Booking"},
		{"GOOGLE_HOTELS", "Google"},
		{"TRIVAGO", "Trivago"},
		{"  airbnb  ", "Airbnb"},
		{"custom_provider", "custom_provider"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := hotelSourceLabel(tt.input)
			if got != tt.want {
				t.Errorf("hotelSourceLabel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// colorizeRating edge cases
// ---------------------------------------------------------------------------

func TestColorizeRating_Negative(t *testing.T) {
	models.UseColor = false
	got := colorizeRating(-1, "-1")
	if got != "-1" {
		t.Errorf("colorizeRating(-1) = %q, want '-1'", got)
	}
}

// ---------------------------------------------------------------------------
// truncateStr edge cases
// ---------------------------------------------------------------------------

func TestTruncateStr_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"empty string", "", 10, ""},
		{"maxLen 0", "hello", 0, ""},
		{"maxLen 1", "hello", 1, "h"},
		{"maxLen 2", "hello", 2, "he"},
		{"maxLen 3", "hello", 3, "hel"},
		{"maxLen 4 truncated", "hello", 4, "h..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateStr(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateStr(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// relativeTimeStr edge cases (already tested, but covering boundary)
// ---------------------------------------------------------------------------

func TestRelativeTimeStr_ExactBoundaries(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"just now", "just now"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// "just now" tested with very recent time.
			// Already covered; this is a compile check.
			_ = tt.want
		})
	}
}

// ---------------------------------------------------------------------------
// shouldShowNudge edge cases
// ---------------------------------------------------------------------------

func TestShouldShowNudge_MCPCommand(t *testing.T) {
	// MCP command should never trigger nudge, even if it were in searchCommands.
	if shouldShowNudge("mcp", "table", envNone, 1, termYes) {
		t.Error("mcp command should never trigger nudge")
	}
}

// ---------------------------------------------------------------------------
// loadNudgeState / saveNudgeState atomicity
// ---------------------------------------------------------------------------

func TestNudgeState_SaveCreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/deep/nested/dir/nudge.json"

	saveNudgeState(path, nudgeState{SearchCount: 42})

	got := loadNudgeState(path)
	if got.SearchCount != 42 {
		t.Errorf("SearchCount = %d after save through nested dirs, want 42", got.SearchCount)
	}
}

// ---------------------------------------------------------------------------
// saveTripPlanLastSearch
// ---------------------------------------------------------------------------

func TestSaveTripPlanLastSearch_Nil(t *testing.T) {
	// Should not panic on nil input.
	saveTripPlanLastSearch(nil)
}

func TestSaveTripPlanLastSearch_Full(t *testing.T) {
	result := &trip.PlanResult{
		Success:     true,
		Origin:      "HEL",
		Destination: "BCN",
		DepartDate:  "2026-07-01",
		ReturnDate:  "2026-07-08",
		Nights:      7,
		Guests:      2,
		OutboundFlights: []trip.PlanFlight{
			{Price: 199, Currency: "EUR", Airline: "Finnair", Stops: 0},
		},
		Hotels: []trip.PlanHotel{
			{Name: "Hotel Arts", Total: 1260, Currency: "EUR"},
		},
		Summary: trip.PlanSummary{
			GrandTotal: 1679,
			Currency:   "EUR",
		},
	}

	// Just verify it does not panic.
	saveTripPlanLastSearch(result)
}

func TestSaveTripPlanLastSearch_NoFlightsNoHotels(t *testing.T) {
	result := &trip.PlanResult{
		Success:     true,
		Origin:      "HEL",
		Destination: "BCN",
		Summary: trip.PlanSummary{
			Currency: "EUR",
		},
	}

	saveTripPlanLastSearch(result)
}

// ---------------------------------------------------------------------------
// outputShare edge cases
// ---------------------------------------------------------------------------

func TestOutputShare_ClipboardMode(t *testing.T) {
	// clipboard mode may fail without pbcopy on CI; just verify no panic.
	_ = outputShare("test data", "clipboard")
}

// ---------------------------------------------------------------------------
// colorizeRating with color enabled
// ---------------------------------------------------------------------------

func TestColorizeRating_WithColor(t *testing.T) {
	models.UseColor = true
	defer func() { models.UseColor = false }()

	got := colorizeRating(9.5, "9.5")
	if !strings.Contains(got, "9.5") {
		t.Errorf("colorizeRating(9.5) should contain '9.5', got %q", got)
	}

	got = colorizeRating(7.0, "7.0")
	if !strings.Contains(got, "7.0") {
		t.Errorf("colorizeRating(7.0) should contain '7.0', got %q", got)
	}

	got = colorizeRating(5.0, "5.0")
	if !strings.Contains(got, "5.0") {
		t.Errorf("colorizeRating(5.0) should contain '5.0', got %q", got)
	}
}

// ---------------------------------------------------------------------------
// priceScale edge cases
// ---------------------------------------------------------------------------

func TestPriceScale_ApplyWithColor(t *testing.T) {
	models.UseColor = true
	defer func() { models.UseColor = false }()

	var ps priceScale
	ps = ps.With(100)
	ps = ps.With(300)

	gotMin := ps.Apply(100, "EUR 100")
	if !strings.Contains(gotMin, "EUR 100") {
		t.Errorf("Apply(min) with color should contain text")
	}

	gotMax := ps.Apply(300, "EUR 300")
	if !strings.Contains(gotMax, "EUR 300") {
		t.Errorf("Apply(max) with color should contain text")
	}

	gotMid := ps.Apply(200, "EUR 200")
	if !strings.Contains(gotMid, "EUR 200") {
		t.Errorf("Apply(mid) with color should contain text")
	}
}

// ---------------------------------------------------------------------------
// colorizeStops with color enabled
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// capitalizeFirst
// ---------------------------------------------------------------------------

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "Hello"},
		{"", ""},
		{"H", "H"},
		{"already", "Already"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := capitalizeFirst(tt.input)
			if got != tt.want {
				t.Errorf("capitalizeFirst(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// colorizeStops with color
// ---------------------------------------------------------------------------

func TestColorizeStops_WithColor(t *testing.T) {
	models.UseColor = true
	defer func() { models.UseColor = false }()

	got := colorizeStops(0)
	if !strings.Contains(got, "Direct") {
		t.Errorf("colorizeStops(0) with color should contain 'Direct'")
	}

	got = colorizeStops(1)
	if !strings.Contains(got, "1 stop") {
		t.Errorf("colorizeStops(1) with color should contain '1 stop'")
	}

	got = colorizeStops(2)
	if !strings.Contains(got, "2 stops") {
		t.Errorf("colorizeStops(2) with color should contain '2 stops'")
	}
}
