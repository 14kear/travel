package mcp

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/MikkoParkkola/trvl/internal/profile"
)

func TestHandleBuildProfileEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.json")

	content, structured, err := handleBuildProfileWithPath(nil, path, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected content blocks")
	}
	if structured == nil {
		t.Error("expected structured output")
	}

	// Content should mention no bookings.
	found := false
	for _, c := range content {
		if c.Type == "text" && containsSubstr(c.Text, "No bookings") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'No bookings' message in content")
	}
}

func TestHandleBuildProfileWithBookings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.json")

	// Seed a profile.
	p := &profile.TravelProfile{
		Bookings: []profile.Booking{
			{Type: "flight", Provider: "KLM", From: "HEL", To: "AMS", Price: 189, Currency: "EUR", TravelDate: "2026-03-15"},
			{Type: "hotel", Provider: "Marriott", Price: 450, Nights: 3, TravelDate: "2026-03-15"},
		},
	}
	if err := profile.SaveTo(path, p); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	content, structured, err := handleBuildProfileWithPath(nil, path, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected content blocks")
	}
	if structured == nil {
		t.Error("expected structured output")
	}

	// Verify the structured output is a rebuilt profile.
	data, _ := json.Marshal(structured)
	var rebuilt profile.TravelProfile
	if err := json.Unmarshal(data, &rebuilt); err != nil {
		t.Fatalf("unmarshal structured: %v", err)
	}
	if rebuilt.TotalFlights != 1 {
		t.Errorf("TotalFlights = %d, want 1", rebuilt.TotalFlights)
	}
}

func TestHandleBuildProfileEmailSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.json")

	args := map[string]any{"source": "email"}
	content, _, err := handleBuildProfileWithPath(args, path, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return email instructions.
	found := false
	for _, c := range content {
		if c.Type == "text" && containsSubstr(c.Text, "Gmail") {
			found = true
		}
	}
	if !found {
		t.Error("expected Gmail instructions in email source response")
	}
}

func TestHandleBuildProfileInvalidSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.json")

	args := map[string]any{"source": "invalid"}
	_, _, err := handleBuildProfileWithPath(args, path, nil, nil)
	if err == nil {
		t.Error("expected error for invalid source")
	}
}

func TestHandleAddBooking(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.json")

	args := map[string]any{
		"type":        "flight",
		"provider":    "KLM",
		"from":        "HEL",
		"to":          "AMS",
		"price":       189.0,
		"currency":    "EUR",
		"travel_date": "2026-03-15",
	}

	content, structured, err := handleAddBookingWithPath(args, path, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected content blocks")
	}
	if structured == nil {
		t.Error("expected structured output")
	}

	// Verify the booking was saved.
	p, err := profile.LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if len(p.Bookings) != 1 {
		t.Fatalf("Bookings len = %d, want 1", len(p.Bookings))
	}
	if p.Bookings[0].Provider != "KLM" {
		t.Errorf("Provider = %q, want KLM", p.Bookings[0].Provider)
	}
}

func TestHandleAddBookingMissingType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.json")

	args := map[string]any{
		"provider": "KLM",
	}

	_, _, err := handleAddBookingWithPath(args, path, nil)
	if err == nil {
		t.Error("expected error for missing type")
	}
}

func TestHandleAddBookingMissingProvider(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.json")

	args := map[string]any{
		"type": "flight",
	}

	_, _, err := handleAddBookingWithPath(args, path, nil)
	if err == nil {
		t.Error("expected error for missing provider")
	}
}

func TestHandleAddBookingDefaultSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.json")

	args := map[string]any{
		"type":     "flight",
		"provider": "KLM",
	}

	_, _, err := handleAddBookingWithPath(args, path, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p, _ := profile.LoadFrom(path)
	if p.Bookings[0].Source != "manual" {
		t.Errorf("Source = %q, want manual", p.Bookings[0].Source)
	}
}

func TestHandleInterviewTrip(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "profile.json")
	prefsPath := filepath.Join(dir, "prefs.json")

	content, structured, err := handleInterviewTripWithPath(nil, profilePath, prefsPath, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected content blocks")
	}
	if structured == nil {
		t.Error("expected structured output")
	}

	// Verify questions are in the structured output.
	data, _ := json.Marshal(structured)
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	questions, ok := result["questions"].([]interface{})
	if !ok || len(questions) == 0 {
		t.Error("expected questions in result")
	}
}

func TestParseBookingFromJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		check   func(t *testing.T, b *profile.Booking)
	}{
		{
			name:  "flight booking",
			input: `{"type":"flight","travel_date":"2026-03-15","from":"HEL","to":"AMS","provider":"KLM","price":189,"currency":"EUR"}`,
			check: func(t *testing.T, b *profile.Booking) {
				if b.Type != "flight" {
					t.Errorf("Type = %q", b.Type)
				}
				if b.Provider != "KLM" {
					t.Errorf("Provider = %q", b.Provider)
				}
				if b.Price != 189 {
					t.Errorf("Price = %v", b.Price)
				}
			},
		},
		{
			name:  "hotel booking with checkin",
			input: `{"type":"hotel","checkin":"2026-03-15","hotel_name":"Marriott Prague","price":450,"currency":"EUR","nights":3}`,
			check: func(t *testing.T, b *profile.Booking) {
				if b.Type != "hotel" {
					t.Errorf("Type = %q", b.Type)
				}
				if b.Provider != "Marriott Prague" {
					t.Errorf("Provider = %q", b.Provider)
				}
				if b.TravelDate != "2026-03-15" {
					t.Errorf("TravelDate = %q", b.TravelDate)
				}
				if b.Nights != 3 {
					t.Errorf("Nights = %d", b.Nights)
				}
			},
		},
		{
			name:    "not a booking",
			input:   `{"type":"not_booking"}`,
			wantNil: true,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid`,
			wantNil: true, // error case
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := parseBookingFromJSON([]byte(tt.input))
			if tt.wantNil {
				if b != nil && err == nil {
					t.Errorf("expected nil booking, got %+v", b)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if b == nil {
				t.Fatal("expected non-nil booking")
			}
			if b.Source != "email" {
				t.Errorf("Source = %q, want email", b.Source)
			}
			if tt.check != nil {
				tt.check(t, b)
			}
		})
	}
}

func TestFormatProfileSummary(t *testing.T) {
	// Empty profile.
	s := formatProfileSummary(nil)
	if s != "No booking history." {
		t.Errorf("nil summary = %q", s)
	}

	s = formatProfileSummary(&profile.TravelProfile{})
	if s != "No booking history." {
		t.Errorf("empty summary = %q", s)
	}

	// Profile with data.
	p := &profile.TravelProfile{
		TotalTrips:   5,
		TotalFlights: 10,
		TopAirlines:  []profile.AirlineStats{{Code: "KL", Name: "KLM", Flights: 8}},
		HomeDetected: []string{"HEL"},
		BudgetTier:   "mid-range",
		Bookings:     make([]profile.Booking, 1), // non-empty
	}
	s = formatProfileSummary(p)
	if !containsSubstr(s, "5 trips") {
		t.Errorf("summary should contain trip count: %q", s)
	}
	if !containsSubstr(s, "KLM") {
		t.Errorf("summary should contain airline: %q", s)
	}
	if !containsSubstr(s, "HEL") {
		t.Errorf("summary should contain home airport: %q", s)
	}
}

func TestToolDefs(t *testing.T) {
	// Verify tool definitions have required fields.
	tools := []ToolDef{buildProfileTool(), addBookingTool(), interviewTripTool()}

	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("tool name is empty")
		}
		if tool.Description == "" {
			t.Errorf("tool %s has empty description", tool.Name)
		}
		if tool.InputSchema.Type != "object" {
			t.Errorf("tool %s InputSchema.Type = %q, want object", tool.Name, tool.InputSchema.Type)
		}
	}

	// add_booking should have required fields.
	addTool := addBookingTool()
	if len(addTool.InputSchema.Required) == 0 {
		t.Error("add_booking should have required fields")
	}
}

// containsSubstr is a test helper (containsSubstring is declared in tools_providers_test.go).
func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
