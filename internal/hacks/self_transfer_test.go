package hacks

import (
	"context"
	"testing"
)

func TestDetectSelfTransfer_emptyInput(t *testing.T) {
	hacks := detectSelfTransfer(context.Background(), DetectorInput{})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for empty input, got %d", len(hacks))
	}
}

func TestDetectSelfTransfer_missingOrigin(t *testing.T) {
	hacks := detectSelfTransfer(context.Background(), DetectorInput{
		Destination: "BCN",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for missing origin, got %d", len(hacks))
	}
}

func TestDetectSelfTransfer_missingDestination(t *testing.T) {
	hacks := detectSelfTransfer(context.Background(), DetectorInput{
		Origin: "HEL",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for missing destination, got %d", len(hacks))
	}
}

func TestDetectSelfTransfer_sameOriginDestination(t *testing.T) {
	hacks := detectSelfTransfer(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "HEL",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for same origin/dest, got %d", len(hacks))
	}
}

func TestDetectSelfTransfer_directLCCRoute(t *testing.T) {
	// STN→BCN is a dense LCC direct route — self-transfer should not fire.
	hacks := detectSelfTransfer(context.Background(), DetectorInput{
		Origin:      "STN",
		Destination: "BCN",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for direct LCC route STN→BCN, got %d", len(hacks))
	}
}

func TestDetectSelfTransfer_directLCCRouteReverse(t *testing.T) {
	// BCN→STN reverse direction should also be detected as direct.
	hacks := detectSelfTransfer(context.Background(), DetectorInput{
		Origin:      "BCN",
		Destination: "STN",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for reverse LCC route BCN→STN, got %d", len(hacks))
	}
}

func TestDetectSelfTransfer_applicableRoute(t *testing.T) {
	// HEL→AGP (Malaga) is not a dense LCC route — self-transfer should fire.
	hacks := detectSelfTransfer(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "AGP",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack for HEL→AGP")
	}
	for _, h := range hacks {
		if h.Type != "self_transfer" {
			t.Errorf("type = %q, want self_transfer", h.Type)
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
}

func TestDetectSelfTransfer_excludesOriginAndDestHubs(t *testing.T) {
	// When origin or destination IS a hub, that hub should not appear.
	hacks := detectSelfTransfer(context.Background(), DetectorInput{
		Origin:      "BGY",
		Destination: "AGP",
	})
	for _, h := range hacks {
		if containsSubstring(h.Title, "BGY") {
			t.Errorf("hub BGY should be excluded since it is the origin, got title: %s", h.Title)
		}
	}

	hacks2 := detectSelfTransfer(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
	})
	for _, h := range hacks2 {
		if containsSubstring(h.Title, "(BCN)") {
			t.Errorf("hub BCN should be excluded since it is the destination, got title: %s", h.Title)
		}
	}
}

func TestDetectSelfTransfer_currencyDefault(t *testing.T) {
	hacks := detectSelfTransfer(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "AGP",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack")
	}
	if hacks[0].Currency != "EUR" {
		t.Errorf("currency = %q, want EUR", hacks[0].Currency)
	}
}

func TestDetectSelfTransfer_customCurrency(t *testing.T) {
	hacks := detectSelfTransfer(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "AGP",
		Currency:    "GBP",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack")
	}
	if hacks[0].Currency != "GBP" {
		t.Errorf("currency = %q, want GBP", hacks[0].Currency)
	}
}

func TestDetectSelfTransfer_caseInsensitive(t *testing.T) {
	hacks := detectSelfTransfer(context.Background(), DetectorInput{
		Origin:      "hel",
		Destination: "agp",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack for lowercase input")
	}
}

// --- Static data tests ---

func TestSelfTransferHubs_populated(t *testing.T) {
	if len(selfTransferHubs) == 0 {
		t.Fatal("selfTransferHubs is empty")
	}
	for i, hub := range selfTransferHubs {
		if len(hub.IATA) != 3 {
			t.Errorf("[%d] IATA %q is not 3 chars", i, hub.IATA)
		}
		if hub.City == "" {
			t.Errorf("[%d] City is empty", i)
		}
		if hub.MinConnectionMin <= 0 {
			t.Errorf("[%d] MinConnectionMin must be > 0, got %d", i, hub.MinConnectionMin)
		}
		if len(hub.Airlines) == 0 {
			t.Errorf("[%d] Airlines is empty", i)
		}
		if hub.Terminal == "" {
			t.Errorf("[%d] Terminal is empty", i)
		}
	}
}

func TestHubAirlineNames(t *testing.T) {
	names := hubAirlineNames([]string{"FR", "W6"})
	if names != "Ryanair, Wizz Air" {
		t.Errorf("got %q, want %q", names, "Ryanair, Wizz Air")
	}
}

func TestHubAirlineNames_unknownCode(t *testing.T) {
	names := hubAirlineNames([]string{"ZZ"})
	if names != "ZZ" {
		t.Errorf("got %q, want %q", names, "ZZ")
	}
}

func TestIsDirectLCCRoute(t *testing.T) {
	tests := []struct {
		origin, dest string
		want         bool
	}{
		{"STN", "BCN", true},
		{"BCN", "STN", true}, // reverse
		{"HEL", "AGP", false},
		{"HEL", "PRG", false},
		{"BGY", "CRL", true},
	}
	for _, tc := range tests {
		got := isDirectLCCRoute(tc.origin, tc.dest)
		if got != tc.want {
			t.Errorf("isDirectLCCRoute(%q, %q) = %v, want %v", tc.origin, tc.dest, got, tc.want)
		}
	}
}
