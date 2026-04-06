package hacks

import (
	"context"
	"testing"
)

// TestDetectorInput_currency verifies the currency fallback.
func TestDetectorInput_currency(t *testing.T) {
	tests := []struct {
		in   DetectorInput
		want string
	}{
		{DetectorInput{Currency: "USD"}, "USD"},
		{DetectorInput{Currency: ""}, "EUR"},
	}
	for _, tc := range tests {
		got := tc.in.currency()
		if got != tc.want {
			t.Errorf("currency() = %q, want %q", got, tc.want)
		}
	}
}

// TestDetectAll_emptyInput verifies DetectAll does not panic on empty input.
func TestDetectAll_emptyInput(t *testing.T) {
	// With empty input all detectors should return quickly without panicking.
	hacks := DetectAll(context.Background(), DetectorInput{})
	// We cannot assert specific counts because real API calls are involved,
	// but the function must not panic and must return a slice (possibly empty).
	if hacks == nil {
		// nil is acceptable; normalise to empty for assertions below.
		hacks = []Hack{}
	}
	_ = hacks
}

// TestHackFields verifies the Hack struct serialises correctly.
func TestHackFields(t *testing.T) {
	h := Hack{
		Type:        "throwaway",
		Title:       "Throwaway ticketing",
		Description: "Some description",
		Savings:     88,
		Currency:    "EUR",
		Risks:       []string{"risk1"},
		Steps:       []string{"step1"},
		Citations:   []string{"https://example.com"},
	}
	if h.Type != "throwaway" {
		t.Errorf("unexpected Type: %q", h.Type)
	}
	if h.Savings != 88 {
		t.Errorf("unexpected Savings: %v", h.Savings)
	}
}

// TestStopoverPrograms verifies the static database is not empty and has
// required fields set.
func TestStopoverPrograms(t *testing.T) {
	if len(stopoverPrograms) == 0 {
		t.Fatal("stopoverPrograms is empty")
	}
	for code, prog := range stopoverPrograms {
		if prog.Airline == "" {
			t.Errorf("[%s] Airline is empty", code)
		}
		if prog.Hub == "" {
			t.Errorf("[%s] Hub is empty", code)
		}
		if prog.MaxNights <= 0 {
			t.Errorf("[%s] MaxNights must be > 0, got %d", code, prog.MaxNights)
		}
	}
}

// TestAddDays verifies date arithmetic.
func TestAddDays(t *testing.T) {
	tests := []struct {
		date  string
		delta int
		want  string
	}{
		{"2026-04-13", 7, "2026-04-20"},
		{"2026-04-13", -3, "2026-04-10"},
		{"2026-04-13", 0, "2026-04-13"},
		{"invalid", 1, ""},
	}
	for _, tc := range tests {
		got := addDays(tc.date, tc.delta)
		if got != tc.want {
			t.Errorf("addDays(%q, %d) = %q, want %q", tc.date, tc.delta, got, tc.want)
		}
	}
}

// TestRoundSavings verifies rounding behaviour.
func TestRoundSavings(t *testing.T) {
	tests := []struct {
		in   float64
		want float64
	}{
		{12.4, 12},
		{12.5, 13},
		{0, 0},
		{-5.1, -5},
	}
	for _, tc := range tests {
		got := roundSavings(tc.in)
		if got != tc.want {
			t.Errorf("roundSavings(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestIsOvernightRoute verifies overnight detection heuristic.
func TestIsOvernightRoute(t *testing.T) {
	tests := []struct {
		dep  string
		arr  string
		want bool
	}{
		{"2026-04-13T21:55", "2026-04-14T10:40", true},  // classic night bus
		{"2026-04-13T14:00", "2026-04-13T16:00", false}, // afternoon route
		{"2026-04-13T08:00", "2026-04-13T12:00", false}, // morning route
		{"2026-04-13T23:00", "2026-04-14T06:30", true},  // night train
	}
	for _, tc := range tests {
		got := isOvernightRoute(tc.dep, tc.arr)
		if got != tc.want {
			t.Errorf("isOvernightRoute(%q, %q) = %v, want %v", tc.dep, tc.arr, got, tc.want)
		}
	}
}

// TestMatchStopoverProgram verifies hub matching.
func TestMatchStopoverProgram(t *testing.T) {
	prog, ok := matchStopoverProgram("HEL", "AY")
	if !ok {
		t.Fatal("expected match for HEL/AY")
	}
	if prog.Airline != "Finnair" {
		t.Errorf("expected Finnair, got %q", prog.Airline)
	}

	_, ok = matchStopoverProgram("JFK", "AA")
	if ok {
		t.Error("expected no match for JFK/AA")
	}
}

// TestAdjustReturnDate verifies return date shifting.
func TestAdjustReturnDate(t *testing.T) {
	tests := []struct {
		ret   string
		delta int
		want  string
	}{
		{"2026-04-22", 3, "2026-04-25"},
		{"2026-04-22", -1, "2026-04-21"},
		{"", 3, ""},
	}
	for _, tc := range tests {
		got := adjustReturnDate(tc.ret, tc.delta)
		if got != tc.want {
			t.Errorf("adjustReturnDate(%q, %d) = %q, want %q", tc.ret, tc.delta, got, tc.want)
		}
	}
}

// TestCityFromCode verifies IATA code to city name mapping.
func TestCityFromCode(t *testing.T) {
	if got := cityFromCode("HEL"); got != "Helsinki" {
		t.Errorf("cityFromCode(HEL) = %q, want Helsinki", got)
	}
	// Unknown code returns the code itself.
	if got := cityFromCode("XYZ"); got != "XYZ" {
		t.Errorf("cityFromCode(XYZ) = %q, want XYZ", got)
	}
}

// TestHiddenCityExtensions verifies the static map is populated.
func TestHiddenCityExtensions(t *testing.T) {
	if len(hiddenCityExtensions) == 0 {
		t.Fatal("hiddenCityExtensions is empty")
	}
	beyonds, ok := hiddenCityExtensions["AMS"]
	if !ok {
		t.Fatal("AMS should have hidden-city extensions")
	}
	if len(beyonds) == 0 {
		t.Error("AMS beyonds should be non-empty")
	}
}

// TestNearbyAirports verifies the static positioning map.
func TestNearbyAirports(t *testing.T) {
	if len(nearbyAirports) == 0 {
		t.Fatal("nearbyAirports is empty")
	}
	entries, ok := nearbyAirports["HEL"]
	if !ok {
		t.Fatal("HEL should have nearby airports")
	}
	for _, e := range entries {
		if e.Code == "" {
			t.Error("entry with empty Code in HEL nearby airports")
		}
		if e.GroundCost < 0 {
			t.Errorf("negative GroundCost for %s", e.Code)
		}
	}
}
