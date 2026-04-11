package lounges

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSearchLounges_StaticFallback verifies that known hub airports return results
// without any external API call.
func TestSearchLounges_StaticFallback(t *testing.T) {
	// Point Priority Pass API at an address that always fails so the static fallback runs.
	orig := priorityPassBaseURL
	priorityPassBaseURL = "http://127.0.0.1:0" // unreachable
	defer func() { priorityPassBaseURL = orig }()

	result, err := SearchLounges(context.Background(), "HEL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Source != "static" {
		t.Errorf("source: got %q, want static", result.Source)
	}
	if result.Airport != "HEL" {
		t.Errorf("airport: got %q, want HEL", result.Airport)
	}
	if result.Count == 0 {
		t.Error("expected at least 1 lounge for HEL")
	}
	if len(result.Lounges) != result.Count {
		t.Errorf("count mismatch: Count=%d len(Lounges)=%d", result.Count, len(result.Lounges))
	}
	// Verify all HEL lounges have required fields.
	for _, l := range result.Lounges {
		if l.Name == "" {
			t.Error("lounge has empty name")
		}
		if l.Airport != "HEL" {
			t.Errorf("lounge airport: got %q, want HEL", l.Airport)
		}
		if len(l.Cards) == 0 {
			t.Errorf("lounge %q has no access cards", l.Name)
		}
	}
}

// TestSearchLounges_UnknownAirport verifies that an unknown airport returns a
// successful empty result (graceful degradation, not an error).
func TestSearchLounges_UnknownAirport(t *testing.T) {
	orig := priorityPassBaseURL
	priorityPassBaseURL = "http://127.0.0.1:0"
	defer func() { priorityPassBaseURL = orig }()

	result, err := SearchLounges(context.Background(), "XYZ")
	if err != nil {
		t.Fatalf("unexpected error for unknown airport: %v", err)
	}
	if !result.Success {
		t.Error("expected success even for unknown airport")
	}
	if result.Count != 0 {
		t.Errorf("expected 0 lounges for XYZ, got %d", result.Count)
	}
}

// TestSearchLounges_InvalidIATA verifies that invalid codes are rejected.
func TestSearchLounges_InvalidIATA(t *testing.T) {
	for _, code := range []string{"", "HE", "HELL", "12", "123"} {
		_, err := SearchLounges(context.Background(), code)
		if err == nil {
			t.Errorf("expected error for code %q, got nil", code)
		}
	}
}

// TestSearchLounges_PriorityPassAPI tests the live-API path against a mock server.
func TestSearchLounges_PriorityPassAPI(t *testing.T) {
	// Mock the Priority Pass search API response for airport "HEL".
	mockResults := []ppSearchResult{
		{
			Heading:    "Helsinki Airport",
			Subheading: "HEL, Helsinki, Finland",
			LocationID: "HEL-Helsinki Airport",
			URL:        "/lounges/finland/helsinki-vantaa",
		},
	}
	body, _ := json.Marshal(mockResults)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("term") == "" {
			http.Error(w, "missing term", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	orig := priorityPassBaseURL
	priorityPassBaseURL = srv.URL
	defer func() { priorityPassBaseURL = orig }()

	result, err := SearchLounges(context.Background(), "HEL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	if result.Source != "prioritypass" {
		t.Errorf("source: got %q, want prioritypass", result.Source)
	}
	if result.Count == 0 {
		t.Fatal("expected at least 1 lounge for HEL")
	}
	if result.Airport != "HEL" {
		t.Errorf("airport: got %q, want HEL", result.Airport)
	}
}

// TestAnnotateAccess verifies that user cards are matched case-insensitively.
func TestAnnotateAccess(t *testing.T) {
	result := &SearchResult{
		Lounges: []Lounge{
			{Name: "Lounge A", Cards: []string{"Priority Pass", "LoungeKey"}},
			{Name: "Lounge B", Cards: []string{"Amex Platinum", "Diners Club"}},
			{Name: "Lounge C", Cards: []string{"Dragon Pass"}},
		},
	}

	// User has Priority Pass and Diners Club (different casing).
	AnnotateAccess(result, []string{"priority pass", "Diners Club"})

	// Lounge A: user has Priority Pass.
	if len(result.Lounges[0].AccessibleWith) != 1 || result.Lounges[0].AccessibleWith[0] != "priority pass" {
		t.Errorf("Lounge A AccessibleWith: got %v, want [priority pass]", result.Lounges[0].AccessibleWith)
	}
	// Lounge B: user has Diners Club.
	if len(result.Lounges[1].AccessibleWith) != 1 || result.Lounges[1].AccessibleWith[0] != "Diners Club" {
		t.Errorf("Lounge B AccessibleWith: got %v, want [Diners Club]", result.Lounges[1].AccessibleWith)
	}
	// Lounge C: user has no matching card.
	if len(result.Lounges[2].AccessibleWith) != 0 {
		t.Errorf("Lounge C AccessibleWith: got %v, want []", result.Lounges[2].AccessibleWith)
	}
}

// TestAnnotateAccess_Nil verifies nil safety.
func TestAnnotateAccess_Nil(t *testing.T) {
	// Should not panic.
	AnnotateAccess(nil, []string{"Priority Pass"})
	AnnotateAccess(&SearchResult{}, nil)
	AnnotateAccess(&SearchResult{Lounges: []Lounge{{Name: "X"}}}, nil)
}

// TestStaticFallback_AllAirports verifies that all airports in the static dataset
// return valid data (no empty names, all have access cards).
func TestStaticFallback_AllAirports(t *testing.T) {
	for airport, entries := range staticData {
		t.Run(airport, func(t *testing.T) {
			result := staticFallback(airport)
			if !result.Success {
				t.Fatal("expected success")
			}
			if result.Count != len(entries) {
				t.Errorf("count: got %d, want %d", result.Count, len(entries))
			}
			for _, l := range result.Lounges {
				if strings.TrimSpace(l.Name) == "" {
					t.Error("lounge has empty name")
				}
				if len(l.Cards) == 0 {
					t.Errorf("lounge %q has no access cards", l.Name)
				}
			}
		})
	}
}

// TestStaticFallback_Coverage verifies all 30 target airports are present in
// the static dataset.
func TestStaticFallback_Coverage(t *testing.T) {
	required := []string{
		"HEL", "LHR", "CDG", "FRA", "AMS",
		"JFK", "LAX", "SFO",
		"NRT", "HND", "SIN", "DXB", "DOH",
		"IST", "BKK", "ICN", "HKG", "PEK", "PVG",
		"SYD", "MEL", "FCO", "MXP", "MAD", "BCN",
		"MUC", "ZRH", "VIE", "CPH", "OSL",
	}
	for _, code := range required {
		t.Run(code, func(t *testing.T) {
			entries, ok := staticData[code]
			if !ok {
				t.Errorf("airport %q missing from static dataset", code)
				return
			}
			if len(entries) == 0 {
				t.Errorf("airport %q has no lounges", code)
			}
		})
	}
}

// TestStaticFallback_MinLounges verifies that major hubs have at least 2 lounges.
func TestStaticFallback_MinLounges(t *testing.T) {
	majorHubs := []string{
		"LHR", "CDG", "FRA", "AMS", "DXB", "DOH",
		"SIN", "NRT", "ICN", "HKG", "BKK", "JFK", "LAX",
	}
	for _, code := range majorHubs {
		t.Run(code, func(t *testing.T) {
			if len(staticData[code]) < 2 {
				t.Errorf("major hub %q should have >= 2 lounges, got %d", code, len(staticData[code]))
			}
		})
	}
}

// TestStaticFallback_PriorityPassCoverage verifies that each airport has at
// least one lounge accepting Priority Pass (the most widely held card).
func TestStaticFallback_PriorityPassCoverage(t *testing.T) {
	for airport, entries := range staticData {
		t.Run(airport, func(t *testing.T) {
			found := false
			for _, e := range entries {
				for _, c := range e.Cards {
					if c == "Priority Pass" {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				t.Errorf("airport %q has no Priority Pass lounge", airport)
			}
		})
	}
}

// TestStaticFallback_FieldIntegrity verifies every lounge has a terminal and
// opening hours set.
func TestStaticFallback_FieldIntegrity(t *testing.T) {
	for airport, entries := range staticData {
		for i, e := range entries {
			if strings.TrimSpace(e.Terminal) == "" {
				t.Errorf("%s[%d] %q: empty terminal", airport, i, e.Name)
			}
			if strings.TrimSpace(e.OpenHours) == "" {
				t.Errorf("%s[%d] %q: empty open_hours", airport, i, e.Name)
			}
			if len(e.Amenities) == 0 {
				t.Errorf("%s[%d] %q: no amenities listed", airport, i, e.Name)
			}
		}
	}
}

// TestSearchLounges_CaseNormalization verifies that lowercase IATA codes work.
func TestSearchLounges_CaseNormalization(t *testing.T) {
	orig := priorityPassBaseURL
	priorityPassBaseURL = "http://127.0.0.1:0"
	defer func() { priorityPassBaseURL = orig }()

	result, err := SearchLounges(context.Background(), "lhr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Airport != "LHR" {
		t.Errorf("airport: got %q, want LHR", result.Airport)
	}
	if result.Count == 0 {
		t.Error("expected lounges for lhr (normalised to LHR)")
	}
}

// TestSharedCardSlices verifies that ppDragon and ppLK shared slices contain
// the expected programs and are not accidentally mutated between calls.
func TestSharedCardSlices(t *testing.T) {
	if len(ppDragon) != 4 {
		t.Errorf("ppDragon len: got %d, want 4", len(ppDragon))
	}
	if len(ppLK) != 2 {
		t.Errorf("ppLK len: got %d, want 2", len(ppLK))
	}

	wantDragon := map[string]bool{
		"Priority Pass": true,
		"Diners Club":   true,
		"LoungeKey":     true,
		"Dragon Pass":   true,
	}
	for _, c := range ppDragon {
		if !wantDragon[c] {
			t.Errorf("unexpected card in ppDragon: %q", c)
		}
	}
}
