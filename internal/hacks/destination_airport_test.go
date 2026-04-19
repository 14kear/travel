package hacks

import (
	"context"
	"strings"
	"testing"
)

// TestDestinationAlternatives_populated verifies the static map is non-empty
// and all entries have required fields.
func TestDestinationAlternatives_populated(t *testing.T) {
	if len(destinationAlternatives) == 0 {
		t.Fatal("destinationAlternatives is empty")
	}
	for dest, alts := range destinationAlternatives {
		if len(alts) == 0 {
			t.Errorf("[%s] has zero alternatives", dest)
		}
		for i, alt := range alts {
			if alt.IATA == "" {
				t.Errorf("[%s][%d] IATA is empty", dest, i)
			}
			if len(alt.IATA) != 3 {
				t.Errorf("[%s][%d] IATA %q is not 3 chars", dest, i, alt.IATA)
			}
			if alt.City == "" {
				t.Errorf("[%s][%d] City is empty", dest, i)
			}
			if alt.TransportCost < 0 {
				t.Errorf("[%s][%d] negative TransportCost: %.0f", dest, i, alt.TransportCost)
			}
			if alt.TransportMin <= 0 {
				t.Errorf("[%s][%d] TransportMin must be > 0, got %d", dest, i, alt.TransportMin)
			}
			if alt.TransportMode == "" {
				t.Errorf("[%s][%d] TransportMode is empty", dest, i)
			}
			if alt.Notes == "" {
				t.Errorf("[%s][%d] Notes is empty", dest, i)
			}
		}
	}
}

// TestDetectDestinationAirport_emptyInput verifies no panic on empty input.
func TestDetectDestinationAirport_emptyInput(t *testing.T) {
	hacks := detectDestinationAirport(context.Background(), DetectorInput{})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for empty input, got %d", len(hacks))
	}
}

// TestDetectDestinationAirport_noDestination verifies early return when
// destination is empty.
func TestDetectDestinationAirport_noDestination(t *testing.T) {
	hacks := detectDestinationAirport(context.Background(), DetectorInput{
		Origin: "HEL",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks, got %d", len(hacks))
	}
}

// TestDetectDestinationAirport_noOrigin verifies early return when origin
// is empty.
func TestDetectDestinationAirport_noOrigin(t *testing.T) {
	hacks := detectDestinationAirport(context.Background(), DetectorInput{
		Destination: "BCN",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks, got %d", len(hacks))
	}
}

// TestDetectDestinationAirport_unknownDestination verifies nil result for a
// destination not in the alternatives map.
func TestDetectDestinationAirport_unknownDestination(t *testing.T) {
	hacks := detectDestinationAirport(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "JFK",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for JFK, got %d", len(hacks))
	}
}

// TestDetectDestinationAirport_knownDestination verifies that a known
// destination produces hack suggestions.
func TestDetectDestinationAirport_knownDestination(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		dest     string
		date     string
		wantMin  int    // minimum expected hacks
		wantType string // expected hack type
	}{
		{
			name:     "BCN has Girona alternative",
			origin:   "HEL",
			dest:     "BCN",
			wantMin:  1,
			wantType: "destination_airport",
		},
		{
			name:     "LHR has Stansted and Luton",
			origin:   "HEL",
			dest:     "LHR",
			wantMin:  2,
			wantType: "destination_airport",
		},
		{
			name:     "CDG has Beauvais and Orly",
			origin:   "HEL",
			dest:     "CDG",
			wantMin:  2,
			wantType: "destination_airport",
		},
		{
			name:     "ARN has Skavsta and Västerås",
			origin:   "HEL",
			dest:     "ARN",
			wantMin:  2,
			wantType: "destination_airport",
		},
		{
			name:     "FCO has Ciampino",
			origin:   "HEL",
			dest:     "FCO",
			date:     "2026-06-15",
			wantMin:  1,
			wantType: "destination_airport",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := DetectorInput{
				Origin:      tc.origin,
				Destination: tc.dest,
				Date:        tc.date,
			}
			hacks := detectDestinationAirport(context.Background(), in)
			if len(hacks) < tc.wantMin {
				t.Errorf("expected >= %d hacks, got %d", tc.wantMin, len(hacks))
			}
			for _, h := range hacks {
				if h.Type != tc.wantType {
					t.Errorf("hack type = %q, want %q", h.Type, tc.wantType)
				}
				if h.Title == "" {
					t.Error("hack title is empty")
				}
				if h.Description == "" {
					t.Error("hack description is empty")
				}
				if len(h.Steps) == 0 {
					t.Error("hack steps are empty")
				}
				if len(h.Risks) == 0 {
					t.Error("hack risks are empty")
				}
				if len(h.Citations) == 0 {
					t.Error("hack citations are empty")
				}
			}
		})
	}
}

// TestDetectDestinationAirport_skipsOriginAsAlternative verifies that if the
// origin airport IS the alternative, it is excluded (nonsensical to suggest
// "fly from EIN to EIN").
func TestDetectDestinationAirport_skipsOriginAsAlternative(t *testing.T) {
	// AMS destination has EIN as alternative. If origin is EIN, skip it.
	hacks := detectDestinationAirport(context.Background(), DetectorInput{
		Origin:      "EIN",
		Destination: "AMS",
	})
	for _, h := range hacks {
		if h.Title != "" && h.Type == "destination_airport" {
			// Should not contain EIN as suggested alternative.
			for _, s := range h.Steps {
				if s != "" && len(s) > 0 {
					// Just ensure we don't get a hack suggesting EIN→EIN.
					continue
				}
			}
		}
	}
	// The key check: EIN→AMS should produce 0 hacks since the only
	// alternative (EIN) is the origin itself.
	if len(hacks) != 0 {
		t.Errorf("expected 0 hacks when origin==alternative, got %d", len(hacks))
	}
}

// TestDetectDestinationAirport_currencyFallback verifies EUR is used when
// no currency is specified.
func TestDetectDestinationAirport_currencyFallback(t *testing.T) {
	hacks := detectDestinationAirport(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack")
	}
	if hacks[0].Currency != "EUR" {
		t.Errorf("currency = %q, want EUR", hacks[0].Currency)
	}
}

// TestDetectDestinationAirport_customCurrency verifies custom currency propagates.
func TestDetectDestinationAirport_customCurrency(t *testing.T) {
	hacks := detectDestinationAirport(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		Currency:    "USD",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack")
	}
	if hacks[0].Currency != "USD" {
		t.Errorf("currency = %q, want USD", hacks[0].Currency)
	}
}

// TestDetectDestinationAirport_withDate verifies that a date in input
// appears in the citation URL.
func TestDetectDestinationAirport_withDate(t *testing.T) {
	hacks := detectDestinationAirport(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "OSL",
		Date:        "2026-07-01",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack for OSL")
	}
	for _, h := range hacks {
		if len(h.Citations) == 0 {
			t.Error("expected at least one citation")
			continue
		}
		found := false
		for _, c := range h.Citations {
			if strings.Contains(c, "2026-07-01") {
				found = true
			}
		}
		if !found {
			t.Errorf("date not found in citations: %v", h.Citations)
		}
	}
}

// TestDestinationCity verifies the helper returns correct names.
func TestDestinationCity(t *testing.T) {
	tests := []struct {
		iata string
		want string
	}{
		{"MXP", "Milan"},
		{"BGY", "Bergamo"},
		{"GRO", "Girona"},
		{"BVA", "Beauvais"},
		{"XYZ", "XYZ"}, // unknown falls back to code
	}
	for _, tc := range tests {
		got := destinationCity(tc.iata)
		if got != tc.want {
			t.Errorf("destinationCity(%q) = %q, want %q", tc.iata, got, tc.want)
		}
	}
}

// TestDestinationAlternativesForDisplay verifies the summary is non-empty.
func TestDestinationAlternativesForDisplay(t *testing.T) {
	summary := destinationAlternativesForDisplay()
	if summary == "" {
		t.Fatal("destinationAlternativesForDisplay returned empty string")
	}
}
