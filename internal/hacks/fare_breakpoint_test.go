package hacks

import (
	"context"
	"testing"
)

func TestDetectFareBreakpoint_emptyInput(t *testing.T) {
	hacks := detectFareBreakpoint(context.Background(), DetectorInput{})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for empty input, got %d", len(hacks))
	}
}

func TestDetectFareBreakpoint_missingOrigin(t *testing.T) {
	hacks := detectFareBreakpoint(context.Background(), DetectorInput{
		Destination: "BKK",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks with missing origin, got %d", len(hacks))
	}
}

func TestDetectFareBreakpoint_missingDestination(t *testing.T) {
	hacks := detectFareBreakpoint(context.Background(), DetectorInput{
		Origin: "HEL",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks with missing destination, got %d", len(hacks))
	}
}

func TestDetectFareBreakpoint_unknownAirports(t *testing.T) {
	hacks := detectFareBreakpoint(context.Background(), DetectorInput{
		Origin:      "XYZ",
		Destination: "ABC",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for unknown airports, got %d", len(hacks))
	}
}

func TestDetectFareBreakpoint_shortHaulNoResults(t *testing.T) {
	// HEL→PRG is ~1320km — too short for any breakpoint hub.
	hacks := detectFareBreakpoint(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "PRG",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for short-haul route, got %d", len(hacks))
	}
}

func TestDetectFareBreakpoint_longHaulFindsHubs(t *testing.T) {
	// HEL→BKK is ~8300km — should find Istanbul and Gulf hubs.
	hacks := detectFareBreakpoint(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BKK",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one fare breakpoint hack for HEL→BKK")
	}

	// All returned hacks should have the correct type.
	for _, h := range hacks {
		if h.Type != "fare_breakpoint" {
			t.Errorf("expected type fare_breakpoint, got %q", h.Type)
		}
		if h.Savings != 0 {
			t.Errorf("advisory hack should have 0 savings, got %.0f", h.Savings)
		}
		if h.Title == "" {
			t.Error("expected non-empty title")
		}
		if h.Description == "" {
			t.Error("expected non-empty description")
		}
		if len(h.Steps) == 0 {
			t.Error("expected non-empty steps")
		}
		if len(h.Risks) == 0 {
			t.Error("expected non-empty risks")
		}
	}
}

func TestDetectFareBreakpoint_istanbulFoundForEuropeAsia(t *testing.T) {
	// HEL→BKK should include Istanbul as a breakpoint.
	hacks := detectFareBreakpoint(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BKK",
	})

	found := false
	for _, h := range hacks {
		for _, s := range h.Steps {
			if containsSubstring(s, "IST") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("expected Istanbul (IST) as a fare breakpoint hub for HEL→BKK")
	}
}

func TestDetectFareBreakpoint_hubIsOriginSkipped(t *testing.T) {
	// IST→BKK: Istanbul is origin, should not suggest routing via IST.
	hacks := detectFareBreakpoint(context.Background(), DetectorInput{
		Origin:      "IST",
		Destination: "BKK",
	})

	for _, h := range hacks {
		for _, s := range h.Steps {
			if containsSubstring(s, "IST→IST") {
				t.Error("should not suggest routing via a hub that is the origin")
			}
		}
	}
}

func TestDetectFareBreakpoint_hubIsDestinationSkipped(t *testing.T) {
	// HEL→IST: Istanbul is destination, should not suggest routing via IST.
	hacks := detectFareBreakpoint(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "IST",
	})

	for _, h := range hacks {
		for _, s := range h.Steps {
			if containsSubstring(s, "IST→IST") {
				t.Error("should not suggest routing via a hub that is the destination")
			}
		}
	}
}

func TestDetectFareBreakpoint_geographicSanityFilter(t *testing.T) {
	// All returned hubs should have via-distance < 1.5× direct distance.
	hacks := detectFareBreakpoint(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BKK",
	})

	directDist := airportDistanceKm("HEL", "BKK")
	for _, h := range hacks {
		// Extract hub IATA from the first step (format: "Search HEL→XXX→BKK ...").
		hub := extractHubFromHack(h)
		if hub == "" {
			continue
		}
		legA := airportDistanceKm("HEL", hub)
		legB := airportDistanceKm(hub, "BKK")
		viaDist := legA + legB
		if viaDist > directDist*1.5 {
			t.Errorf("hub %s via-distance %.0f > 1.5× direct %.0f", hub, viaDist, directDist)
		}
	}
}

func TestDetectFareBreakpoint_bogotaNotForEuropeAsia(t *testing.T) {
	// HEL→BKK: Bogotá (BOG) should NOT appear — it's wildly out of the way
	// and has MinDistanceKm 7000 for LatAm routes.
	hacks := detectFareBreakpoint(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BKK",
	})

	for _, h := range hacks {
		hub := extractHubFromHack(h)
		if hub == "BOG" {
			t.Error("Bogotá should not be suggested for HEL→BKK (geographic sanity)")
		}
	}
}

func TestDetectFareBreakpoint_transatlanticRoute(t *testing.T) {
	// HEL→GRU (~10000km): should find Lisbon/Madrid as breakpoints.
	hacks := detectFareBreakpoint(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "GRU",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack for HEL→GRU transatlantic route")
	}

	foundLIS := false
	foundMAD := false
	for _, h := range hacks {
		hub := extractHubFromHack(h)
		if hub == "LIS" {
			foundLIS = true
		}
		if hub == "MAD" {
			foundMAD = true
		}
	}
	if !foundLIS && !foundMAD {
		t.Error("expected Lisbon or Madrid as breakpoint for HEL→GRU")
	}
}

func TestDetectFareBreakpoint_currencyDefault(t *testing.T) {
	hacks := detectFareBreakpoint(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BKK",
	})
	for _, h := range hacks {
		if h.Currency != "EUR" {
			t.Errorf("expected EUR default currency, got %q", h.Currency)
		}
	}
}

func TestDetectFareBreakpoint_customCurrency(t *testing.T) {
	hacks := detectFareBreakpoint(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BKK",
		Currency:    "USD",
	})
	for _, h := range hacks {
		if h.Currency != "USD" {
			t.Errorf("expected USD currency, got %q", h.Currency)
		}
	}
}

func TestDetectFareBreakpoint_casablancaForAfricaRoute(t *testing.T) {
	// HEL→JNB (~9000km): Casablanca (CMN) and Addis Ababa (ADD) should
	// be considered. CMN has MinDistanceKm 2000, ADD has 4000.
	hacks := detectFareBreakpoint(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "JNB",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack for HEL→JNB")
	}

	foundADD := false
	for _, h := range hacks {
		hub := extractHubFromHack(h)
		if hub == "ADD" {
			foundADD = true
		}
	}
	if !foundADD {
		t.Error("expected Addis Ababa (ADD) as breakpoint for HEL→JNB")
	}
}

func TestDetectFareBreakpoint_staticData(t *testing.T) {
	if len(fareBreakpointHubs) == 0 {
		t.Fatal("fareBreakpointHubs is empty")
	}
	for _, hub := range fareBreakpointHubs {
		if len(hub.IATA) != 3 {
			t.Errorf("hub IATA %q is not 3 characters", hub.IATA)
		}
		if hub.City == "" {
			t.Errorf("hub %s has empty city", hub.IATA)
		}
		if hub.Airline == "" {
			t.Errorf("hub %s has empty airline", hub.IATA)
		}
		if hub.Zone == "" {
			t.Errorf("hub %s has empty zone", hub.IATA)
		}
		if hub.MinDistanceKm <= 0 {
			t.Errorf("hub %s has non-positive MinDistanceKm", hub.IATA)
		}
	}
}

func TestDetectFareBreakpoint_allHubsHaveCoords(t *testing.T) {
	for _, hub := range fareBreakpointHubs {
		if _, ok := airportCoords[hub.IATA]; !ok {
			t.Errorf("hub %s (%s) missing from airportCoords", hub.IATA, hub.City)
		}
	}
}

func TestDetectFareBreakpoint_allHubsHaveAirlineNames(t *testing.T) {
	for _, hub := range fareBreakpointHubs {
		if _, ok := airlineNames[hub.Airline]; !ok {
			t.Errorf("hub %s airline %s missing from airlineNames", hub.IATA, hub.Airline)
		}
	}
}

// --- helpers ---
// containsSubstring is declared in accommodation_split_test.go (same package).

// extractHubFromHack extracts the hub IATA from the first step of a fare
// breakpoint hack. Expected format: "Search HEL→XXX→BKK on ..."
func extractHubFromHack(h Hack) string {
	if len(h.Steps) == 0 {
		return ""
	}
	step := h.Steps[0]
	// Find pattern: "→XXX→" where XXX is 3 uppercase letters.
	for i := 0; i < len(step)-6; i++ {
		// UTF-8 → is 3 bytes (0xE2 0x86 0x92).
		if step[i] == 0xE2 && i+8 < len(step) &&
			step[i+1] == 0x86 && step[i+2] == 0x92 &&
			isUpperIATA(step[i+3:i+6]) &&
			step[i+6] == 0xE2 && step[i+7] == 0x86 && step[i+8] == 0x92 {
			return step[i+3 : i+6]
		}
	}
	return ""
}
