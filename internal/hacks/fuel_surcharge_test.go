package hacks

import (
	"testing"
)

func TestDetectFuelSurcharge_emptyInput(t *testing.T) {
	hacks := DetectFuelSurcharge("", "", nil)
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for empty input, got %d", len(hacks))
	}
}

func TestDetectFuelSurcharge_noAirlines(t *testing.T) {
	hacks := DetectFuelSurcharge("HEL", "BKK", nil)
	if len(hacks) != 0 {
		t.Errorf("expected no hacks with no airlines, got %d", len(hacks))
	}
}

func TestDetectFuelSurcharge_emptyAirlines(t *testing.T) {
	hacks := DetectFuelSurcharge("HEL", "BKK", []string{})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks with empty airlines, got %d", len(hacks))
	}
}

func TestDetectFuelSurcharge_shortHaulNoDetection(t *testing.T) {
	// HEL->PRG is ~1320km, well under 3000km threshold.
	hacks := DetectFuelSurcharge("HEL", "PRG", []string{"LH", "BA"})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for short-haul route, got %d", len(hacks))
	}
}

func TestDetectFuelSurcharge_longHaulHighYQ(t *testing.T) {
	// HEL->BKK is >7000km, BA is high-YQ.
	hacks := DetectFuelSurcharge("HEL", "BKK", []string{"BA"})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack for high-YQ airline on long-haul, got %d", len(hacks))
	}
	h := hacks[0]
	if h.Type != "fuel_surcharge" {
		t.Errorf("expected type fuel_surcharge, got %q", h.Type)
	}
	if h.Savings != 700 { // BA TypicalEUR 350 * 2 segments
		t.Errorf("expected savings 700, got %.0f", h.Savings)
	}
	if h.Currency != "EUR" {
		t.Errorf("expected EUR currency, got %q", h.Currency)
	}
}

func TestDetectFuelSurcharge_multipleHighYQ(t *testing.T) {
	// Both BA and LH are high-YQ.
	hacks := DetectFuelSurcharge("HEL", "BKK", []string{"BA", "LH"})
	if len(hacks) != 2 {
		t.Fatalf("expected 2 hacks for two high-YQ airlines, got %d", len(hacks))
	}
	types := map[string]bool{}
	for _, h := range hacks {
		types[h.Type] = true
		if h.Type != "fuel_surcharge" {
			t.Errorf("expected type fuel_surcharge, got %q", h.Type)
		}
	}
}

func TestDetectFuelSurcharge_zeroYQAirlineOnly(t *testing.T) {
	// TK has zero surcharges — should not trigger.
	hacks := DetectFuelSurcharge("HEL", "BKK", []string{"TK"})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for zero-YQ airline, got %d", len(hacks))
	}
}

func TestDetectFuelSurcharge_lowYQAirlineOnly(t *testing.T) {
	// SK has low surcharges — should not trigger (only "high" triggers).
	hacks := DetectFuelSurcharge("HEL", "SIN", []string{"SK"})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for low-YQ airline, got %d", len(hacks))
	}
}

func TestDetectFuelSurcharge_mediumYQAirlineOnly(t *testing.T) {
	// AY has medium surcharges — should not trigger (only "high" triggers).
	hacks := DetectFuelSurcharge("HEL", "BKK", []string{"AY"})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for medium-YQ airline, got %d", len(hacks))
	}
}

func TestDetectFuelSurcharge_mixedAirlines(t *testing.T) {
	// Mix of high-YQ and zero-YQ — should only flag the high-YQ ones.
	hacks := DetectFuelSurcharge("HEL", "BKK", []string{"TK", "BA", "SQ"})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack (only BA), got %d", len(hacks))
	}
	if hacks[0].Savings != 700 { // BA 350*2
		t.Errorf("expected savings 700, got %.0f", hacks[0].Savings)
	}
}

func TestDetectFuelSurcharge_unknownAirline(t *testing.T) {
	// Unknown airline code should be ignored.
	hacks := DetectFuelSurcharge("HEL", "BKK", []string{"XX"})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for unknown airline, got %d", len(hacks))
	}
}

func TestDetectFuelSurcharge_caseInsensitive(t *testing.T) {
	// Lowercase airline codes should still work.
	hacks := DetectFuelSurcharge("HEL", "BKK", []string{"ba"})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack for lowercase 'ba', got %d", len(hacks))
	}
}

func TestDetectFuelSurcharge_duplicateAirline(t *testing.T) {
	// Same airline passed twice should produce only one hack.
	hacks := DetectFuelSurcharge("HEL", "BKK", []string{"BA", "BA"})
	if len(hacks) != 1 {
		t.Errorf("expected 1 deduplicated hack, got %d", len(hacks))
	}
}

func TestDetectFuelSurcharge_unknownAirportsProceeds(t *testing.T) {
	// Both airports unknown — distance returns 0, should proceed optimistically.
	hacks := DetectFuelSurcharge("XYZ", "ABC", []string{"LH"})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack for unknown airports (optimistic), got %d", len(hacks))
	}
}

func TestDetectFuelSurcharge_hackHasRequiredFields(t *testing.T) {
	hacks := DetectFuelSurcharge("HEL", "BKK", []string{"LH"})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack, got %d", len(hacks))
	}
	h := hacks[0]
	if h.Title == "" {
		t.Error("expected non-empty title")
	}
	if h.Description == "" {
		t.Error("expected non-empty description")
	}
	if len(h.Risks) == 0 {
		t.Error("expected non-empty risks")
	}
	if len(h.Steps) == 0 {
		t.Error("expected non-empty steps")
	}
}

func TestDetectFuelSurcharge_suggestsAlternatives(t *testing.T) {
	// HEL->BKK with LH — should suggest zero-YQ alternatives for europe-asia.
	hacks := DetectFuelSurcharge("HEL", "BKK", []string{"LH"})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack, got %d", len(hacks))
	}
	// Steps should mention alternatives.
	found := false
	for _, step := range hacks[0].Steps {
		if len(step) > 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected steps to contain alternative suggestions")
	}
	// Should have at least 3 steps (route info, alternatives, search suggestion).
	if len(hacks[0].Steps) < 3 {
		t.Errorf("expected at least 3 steps, got %d", len(hacks[0].Steps))
	}
}

func TestDetectFuelSurcharge_savingsLH(t *testing.T) {
	hacks := DetectFuelSurcharge("LHR", "NRT", []string{"LH"})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack, got %d", len(hacks))
	}
	// LH typical is 250 EUR per segment, 500 RT.
	if hacks[0].Savings != 500 {
		t.Errorf("expected 500 EUR savings for LH, got %.0f", hacks[0].Savings)
	}
}

func TestDetectFuelSurcharge_savingsAF(t *testing.T) {
	hacks := DetectFuelSurcharge("CDG", "NRT", []string{"AF"})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack, got %d", len(hacks))
	}
	if hacks[0].Savings != 500 {
		t.Errorf("expected 500 EUR savings for AF, got %.0f", hacks[0].Savings)
	}
}

// --- Region classification tests ---

func TestAirportRegion_european(t *testing.T) {
	region := airportRegion("HEL")
	if region != "europe" {
		t.Errorf("HEL should be europe, got %q", region)
	}
}

func TestAirportRegion_asian(t *testing.T) {
	region := airportRegion("BKK")
	if region != "asia" {
		t.Errorf("BKK should be asia, got %q", region)
	}
}

func TestAirportRegion_americas(t *testing.T) {
	region := airportRegion("JFK")
	if region != "americas" {
		t.Errorf("JFK should be americas, got %q", region)
	}
}

func TestAirportRegion_african(t *testing.T) {
	region := airportRegion("ADD")
	if region != "africa" {
		t.Errorf("ADD should be africa, got %q", region)
	}
}

func TestAirportRegion_middleEast(t *testing.T) {
	region := airportRegion("DXB")
	if region != "middle_east" {
		t.Errorf("DXB should be middle_east, got %q", region)
	}
}

func TestAirportRegion_oceania(t *testing.T) {
	region := airportRegion("SYD")
	if region != "oceania" {
		t.Errorf("SYD should be oceania, got %q", region)
	}
}

func TestAirportRegion_unknown(t *testing.T) {
	region := airportRegion("XYZ")
	if region != "unknown" {
		t.Errorf("XYZ should be unknown, got %q", region)
	}
}

func TestClassifyRegion_europeAsia(t *testing.T) {
	region := classifyRegion("HEL", "BKK")
	if region != "asia-europe" {
		t.Errorf("HEL->BKK should be asia-europe, got %q", region)
	}
}

func TestClassifyRegion_europeAmericas(t *testing.T) {
	region := classifyRegion("HEL", "JFK")
	if region != "americas-europe" {
		t.Errorf("HEL->JFK should be americas-europe, got %q", region)
	}
}

func TestClassifyRegion_sameRegion(t *testing.T) {
	region := classifyRegion("HEL", "CDG")
	if region != "" {
		t.Errorf("intra-regional should be empty, got %q", region)
	}
}

// --- Static data tests ---

func TestAirlineSurcharges_populated(t *testing.T) {
	if len(airlineSurcharges) < 10 {
		t.Errorf("expected at least 10 airlines, got %d", len(airlineSurcharges))
	}
}

func TestAirlineSurcharges_validLevels(t *testing.T) {
	validLevels := map[string]bool{"none": true, "low": true, "medium": true, "high": true}
	for code, s := range airlineSurcharges {
		if !validLevels[s.Level] {
			t.Errorf("airline %s has invalid level %q", code, s.Level)
		}
		if s.Airline == "" {
			t.Errorf("airline %s has empty Airline field", code)
		}
		if s.AirlineName == "" {
			t.Errorf("airline %s has empty AirlineName field", code)
		}
		if s.Level == "high" && s.TypicalEUR <= 0 {
			t.Errorf("high-YQ airline %s should have positive TypicalEUR", code)
		}
		if s.Level == "none" && s.TypicalEUR != 0 {
			t.Errorf("zero-YQ airline %s should have TypicalEUR 0, got %.0f", code, s.TypicalEUR)
		}
	}
}

func TestHubRegions_allReferenceSurcharges(t *testing.T) {
	for code := range hubRegions {
		if _, ok := airlineSurcharges[code]; !ok {
			t.Errorf("hubRegions references airline %s not in airlineSurcharges", code)
		}
	}
}

func TestRegionServed_allReferenceSurcharges(t *testing.T) {
	for region, airlines := range regionServed {
		for _, code := range airlines {
			if _, ok := airlineSurcharges[code]; !ok {
				t.Errorf("regionServed[%s] references airline %s not in airlineSurcharges", region, code)
			}
		}
	}
}
