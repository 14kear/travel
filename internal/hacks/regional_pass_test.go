package hacks

import (
	"context"
	"testing"
)

func TestDetectRegionalPass_emptyInput(t *testing.T) {
	hacks := detectRegionalPass(context.Background(), DetectorInput{})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for empty input, got %d", len(hacks))
	}
}

func TestDetectRegionalPass_missingBoth(t *testing.T) {
	hacks := detectRegionalPass(context.Background(), DetectorInput{
		Date: "2026-06-15",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks when both origin and dest empty, got %d", len(hacks))
	}
}

func TestDetectRegionalPass_unknownAirports(t *testing.T) {
	hacks := detectRegionalPass(context.Background(), DetectorInput{
		Origin:      "XYZ",
		Destination: "ABC",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for unknown airports, got %d", len(hacks))
	}
}

func TestDetectRegionalPass_germanyRoute(t *testing.T) {
	// FRA→MUC is within Germany — should suggest Deutschlandticket and BahnCards.
	hacks := detectRegionalPass(context.Background(), DetectorInput{
		Origin:      "FRA",
		Destination: "MUC",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack for Germany route")
	}

	foundDT := false
	for _, h := range hacks {
		if h.Type != "regional_pass" {
			t.Errorf("type = %q, want regional_pass", h.Type)
		}
		if containsSubstring(h.Title, "Deutschlandticket") {
			foundDT = true
		}
		if h.Title == "" {
			t.Error("title is empty")
		}
		if h.Description == "" {
			t.Error("description is empty")
		}
		if len(h.Steps) == 0 {
			t.Error("steps are empty")
		}
		if len(h.Risks) == 0 {
			t.Error("risks are empty")
		}
		if h.Savings != 0 {
			t.Errorf("advisory hack should have 0 savings, got %.0f", h.Savings)
		}
	}
	if !foundDT {
		t.Error("expected Deutschlandticket in results for German route")
	}
}

func TestDetectRegionalPass_austriaRoute(t *testing.T) {
	// VIE→INN is within Austria — should suggest Klimaticket and Vorteilscard.
	hacks := detectRegionalPass(context.Background(), DetectorInput{
		Origin:      "VIE",
		Destination: "INN",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack for Austria route")
	}

	foundKlima := false
	foundVorteil := false
	for _, h := range hacks {
		if containsSubstring(h.Title, "Klimaticket") {
			foundKlima = true
		}
		if containsSubstring(h.Title, "Vorteilscard") {
			foundVorteil = true
		}
	}
	if !foundKlima {
		t.Error("expected Klimaticket for Austria route")
	}
	if !foundVorteil {
		t.Error("expected Vorteilscard for Austria route")
	}
}

func TestDetectRegionalPass_switzerlandRoute(t *testing.T) {
	// ZRH→GVA — should suggest Swiss Half Fare Card.
	hacks := detectRegionalPass(context.Background(), DetectorInput{
		Origin:      "ZRH",
		Destination: "GVA",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack for Switzerland route")
	}

	foundSwiss := false
	for _, h := range hacks {
		if containsSubstring(h.Title, "Swiss Half Fare") {
			foundSwiss = true
		}
	}
	if !foundSwiss {
		t.Error("expected Swiss Half Fare Card for Switzerland route")
	}
}

func TestDetectRegionalPass_netherlandsRoute(t *testing.T) {
	// AMS→EIN — should suggest OV-chipkaart.
	hacks := detectRegionalPass(context.Background(), DetectorInput{
		Origin:      "AMS",
		Destination: "EIN",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack for Netherlands route")
	}

	foundOV := false
	for _, h := range hacks {
		if containsSubstring(h.Title, "OV-chipkaart") {
			foundOV = true
		}
	}
	if !foundOV {
		t.Error("expected OV-chipkaart for Netherlands route")
	}
}

func TestDetectRegionalPass_crossBorderAustriaGermany(t *testing.T) {
	// VIE→MUC crosses AT/DE — Vorteilscard is valid for both (AT, DE, CH).
	hacks := detectRegionalPass(context.Background(), DetectorInput{
		Origin:      "VIE",
		Destination: "MUC",
	})
	if len(hacks) == 0 {
		t.Fatal("expected hacks for cross-border AT→DE")
	}

	// Should have passes from both countries.
	hasDE := false
	hasAT := false
	for _, h := range hacks {
		if containsSubstring(h.Description, "Germany") || containsSubstring(h.Description, "DB") {
			hasDE = true
		}
		if containsSubstring(h.Description, "Austria") || containsSubstring(h.Description, "OBB") {
			hasAT = true
		}
	}
	if !hasDE {
		t.Error("expected German pass suggestions for VIE→MUC")
	}
	if !hasAT {
		t.Error("expected Austrian pass suggestions for VIE→MUC")
	}
}

func TestDetectRegionalPass_nonApplicableRoute(t *testing.T) {
	// HEL→PRG: Finland and Czech Republic have no passes in our list.
	hacks := detectRegionalPass(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "PRG",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for FI→CZ route, got %d", len(hacks))
	}
}

func TestDetectRegionalPass_originOnly(t *testing.T) {
	// Only origin set, in Germany — should still suggest passes.
	hacks := detectRegionalPass(context.Background(), DetectorInput{
		Origin: "FRA",
	})
	if len(hacks) == 0 {
		t.Fatal("expected hacks when only origin (German airport) is set")
	}
}

func TestDetectRegionalPass_destinationOnly(t *testing.T) {
	// Only destination set, in Netherlands — should suggest passes.
	hacks := detectRegionalPass(context.Background(), DetectorInput{
		Destination: "AMS",
	})
	if len(hacks) == 0 {
		t.Fatal("expected hacks when only destination (Dutch airport) is set")
	}
}

func TestDetectRegionalPass_currencyDefault(t *testing.T) {
	hacks := detectRegionalPass(context.Background(), DetectorInput{
		Origin:      "FRA",
		Destination: "MUC",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack")
	}
	if hacks[0].Currency != "EUR" {
		t.Errorf("currency = %q, want EUR", hacks[0].Currency)
	}
}

func TestDetectRegionalPass_customCurrency(t *testing.T) {
	hacks := detectRegionalPass(context.Background(), DetectorInput{
		Origin:      "FRA",
		Destination: "MUC",
		Currency:    "CHF",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack")
	}
	if hacks[0].Currency != "CHF" {
		t.Errorf("currency = %q, want CHF", hacks[0].Currency)
	}
}

func TestDetectRegionalPass_noDuplicates(t *testing.T) {
	// VIE→INN: both are AT airports, but each pass should appear only once.
	hacks := detectRegionalPass(context.Background(), DetectorInput{
		Origin:      "VIE",
		Destination: "INN",
	})
	seen := make(map[string]bool)
	for _, h := range hacks {
		if seen[h.Title] {
			t.Errorf("duplicate hack title: %s", h.Title)
		}
		seen[h.Title] = true
	}
}

// --- Static data tests ---

func TestRegionalPasses_populated(t *testing.T) {
	if len(regionalPasses) == 0 {
		t.Fatal("regionalPasses is empty")
	}
	for i, p := range regionalPasses {
		if p.Name == "" {
			t.Errorf("[%d] Name is empty", i)
		}
		if p.Country == "" {
			t.Errorf("[%d] Country is empty", i)
		}
		if p.PriceEUR <= 0 {
			t.Errorf("[%d] PriceEUR must be > 0, got %.2f", i, p.PriceEUR)
		}
		if p.Period == "" {
			t.Errorf("[%d] Period is empty", i)
		}
		if p.Coverage == "" {
			t.Errorf("[%d] Coverage is empty", i)
		}
		if len(p.ValidFor) == 0 {
			t.Errorf("[%d] ValidFor is empty", i)
		}
		if p.Notes == "" {
			t.Errorf("[%d] Notes is empty", i)
		}
	}
}

func TestIATAToCountry_populated(t *testing.T) {
	if len(iataToCountry) < 50 {
		t.Errorf("expected at least 50 entries in iataToCountry, got %d", len(iataToCountry))
	}
}

func TestIATAToCountry_spotCheck(t *testing.T) {
	tests := []struct {
		iata    string
		country string
	}{
		{"FRA", "DE"},
		{"VIE", "AT"},
		{"ZRH", "CH"},
		{"AMS", "NL"},
		{"HEL", "FI"},
		{"CDG", "FR"},
		{"FCO", "IT"},
		{"MAD", "ES"},
		{"LHR", "GB"},
	}
	for _, tc := range tests {
		got, ok := iataToCountry[tc.iata]
		if !ok {
			t.Errorf("iataToCountry missing %s", tc.iata)
			continue
		}
		if got != tc.country {
			t.Errorf("iataToCountry[%s] = %q, want %q", tc.iata, got, tc.country)
		}
	}
}
