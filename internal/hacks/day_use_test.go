package hacks

import (
	"context"
	"testing"
)

func TestDetectDayUse_emptyInput(t *testing.T) {
	hacks := detectDayUse(context.Background(), DetectorInput{})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for empty input, got %d", len(hacks))
	}
}

func TestDetectDayUse_missingOrigin(t *testing.T) {
	hacks := detectDayUse(context.Background(), DetectorInput{
		Destination: "BCN",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks with missing origin, got %d", len(hacks))
	}
}

func TestDetectDayUse_missingDestination(t *testing.T) {
	hacks := detectDayUse(context.Background(), DetectorInput{
		Origin: "HEL",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks with missing destination, got %d", len(hacks))
	}
}

func TestDetectDayUse_fires(t *testing.T) {
	hacks := detectDayUse(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
	})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack, got %d", len(hacks))
	}
	h := hacks[0]
	if h.Type != "day_use_hotel" {
		t.Errorf("expected type day_use_hotel, got %q", h.Type)
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
	if len(h.Citations) == 0 {
		t.Error("expected non-empty citations")
	}
	if h.Savings != 0 {
		t.Errorf("advisory hack should have 0 savings, got %.0f", h.Savings)
	}
}

func TestDetectDayUse_currencyDefault(t *testing.T) {
	hacks := detectDayUse(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
	})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack, got %d", len(hacks))
	}
	if hacks[0].Currency != "EUR" {
		t.Errorf("expected EUR default, got %q", hacks[0].Currency)
	}
}

func TestDetectDayUse_customCurrency(t *testing.T) {
	hacks := detectDayUse(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		Currency:    "SEK",
	})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack, got %d", len(hacks))
	}
	if hacks[0].Currency != "SEK" {
		t.Errorf("expected SEK currency, got %q", hacks[0].Currency)
	}
}

func TestDetectDayUse_hasCitations(t *testing.T) {
	hacks := detectDayUse(context.Background(), DetectorInput{
		Origin:      "LHR",
		Destination: "JFK",
	})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack, got %d", len(hacks))
	}
	citations := hacks[0].Citations
	if len(citations) != 2 {
		t.Fatalf("expected 2 citations, got %d", len(citations))
	}
	// Verify the two known citation URLs.
	wantURLs := map[string]bool{
		"https://www.dayuse.com":      false,
		"https://www.hotelsbyday.com": false,
	}
	for _, c := range citations {
		if _, ok := wantURLs[c]; ok {
			wantURLs[c] = true
		}
	}
	for url, found := range wantURLs {
		if !found {
			t.Errorf("missing expected citation: %s", url)
		}
	}
}

func TestDetectDayUse_notInDetectFlightTips(t *testing.T) {
	// Verify that detectDayUse is NOT included in DetectFlightTips (too generic).
	hacks := DetectFlightTips(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		Date:        "2026-06-01",
	})
	for _, h := range hacks {
		if h.Type == "day_use_hotel" {
			t.Error("day_use_hotel should NOT be in DetectFlightTips")
		}
	}
}
