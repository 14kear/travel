package hacks

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestDetectBackToBack_emptyInput(t *testing.T) {
	hacks := detectBackToBack(context.Background(), DetectorInput{})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for empty input, got %d", len(hacks))
	}
}

func TestDetectBackToBack_missingOrigin(t *testing.T) {
	hacks := detectBackToBack(context.Background(), DetectorInput{
		Destination: "BCN",
		Date:        "2026-06-01",
		ReturnDate:  "2026-06-04",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks with missing origin, got %d", len(hacks))
	}
}

func TestDetectBackToBack_missingDestination(t *testing.T) {
	hacks := detectBackToBack(context.Background(), DetectorInput{
		Origin:     "HEL",
		Date:       "2026-06-01",
		ReturnDate: "2026-06-04",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks with missing destination, got %d", len(hacks))
	}
}

func TestDetectBackToBack_noReturnDate(t *testing.T) {
	// One-way search — back-to-back doesn't apply.
	hacks := detectBackToBack(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		Date:        "2026-06-01",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for one-way search, got %d", len(hacks))
	}
}

func TestDetectBackToBack_noDate(t *testing.T) {
	hacks := detectBackToBack(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		ReturnDate:  "2026-06-04",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks with missing depart date, got %d", len(hacks))
	}
}

func TestDetectBackToBack_invalidDates(t *testing.T) {
	hacks := detectBackToBack(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		Date:        "not-a-date",
		ReturnDate:  "2026-06-04",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for invalid depart date, got %d", len(hacks))
	}

	hacks = detectBackToBack(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		Date:        "2026-06-01",
		ReturnDate:  "bad-date",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for invalid return date, got %d", len(hacks))
	}
}

func TestDetectBackToBack_shortTrip(t *testing.T) {
	// 3-night trip — should fire.
	depart := time.Now().AddDate(0, 0, 30)
	ret := depart.AddDate(0, 0, 3)
	hacks := detectBackToBack(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		Date:        depart.Format("2006-01-02"),
		ReturnDate:  ret.Format("2006-01-02"),
	})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack for 3-night trip, got %d", len(hacks))
	}
	h := hacks[0]
	if h.Type != "back_to_back" {
		t.Errorf("expected type back_to_back, got %q", h.Type)
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
	if h.Savings != 0 {
		t.Errorf("advisory hack should have 0 savings, got %.0f", h.Savings)
	}
}

func TestDetectBackToBack_weekTrip(t *testing.T) {
	// 7-night trip — should fire.
	depart := time.Now().AddDate(0, 0, 30)
	ret := depart.AddDate(0, 0, 7)
	hacks := detectBackToBack(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		Date:        depart.Format("2006-01-02"),
		ReturnDate:  ret.Format("2006-01-02"),
	})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack for 7-night trip, got %d", len(hacks))
	}
}

func TestDetectBackToBack_twoWeekTrip(t *testing.T) {
	// 14-night trip — boundary, should fire.
	depart := time.Now().AddDate(0, 0, 30)
	ret := depart.AddDate(0, 0, 14)
	hacks := detectBackToBack(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		Date:        depart.Format("2006-01-02"),
		ReturnDate:  ret.Format("2006-01-02"),
	})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack for 14-night trip, got %d", len(hacks))
	}
}

func TestDetectBackToBack_longTrip(t *testing.T) {
	// 15-night trip — too long for back-to-back.
	depart := time.Now().AddDate(0, 0, 30)
	ret := depart.AddDate(0, 0, 15)
	hacks := detectBackToBack(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		Date:        depart.Format("2006-01-02"),
		ReturnDate:  ret.Format("2006-01-02"),
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for 15-night trip, got %d", len(hacks))
	}
}

func TestDetectBackToBack_zeroNights(t *testing.T) {
	// Same-day return — too short.
	depart := time.Now().AddDate(0, 0, 30)
	hacks := detectBackToBack(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		Date:        depart.Format("2006-01-02"),
		ReturnDate:  depart.Format("2006-01-02"),
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for same-day return, got %d", len(hacks))
	}
}

func TestDetectBackToBack_oneNight(t *testing.T) {
	// 1-night trip — boundary, should fire.
	depart := time.Now().AddDate(0, 0, 30)
	ret := depart.AddDate(0, 0, 1)
	hacks := detectBackToBack(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		Date:        depart.Format("2006-01-02"),
		ReturnDate:  ret.Format("2006-01-02"),
	})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack for 1-night trip, got %d", len(hacks))
	}
}

func TestDetectBackToBack_currencyDefault(t *testing.T) {
	depart := time.Now().AddDate(0, 0, 30)
	ret := depart.AddDate(0, 0, 3)
	hacks := detectBackToBack(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		Date:        depart.Format("2006-01-02"),
		ReturnDate:  ret.Format("2006-01-02"),
	})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack, got %d", len(hacks))
	}
	if hacks[0].Currency != "EUR" {
		t.Errorf("expected EUR default, got %q", hacks[0].Currency)
	}
}

func TestDetectBackToBack_customCurrency(t *testing.T) {
	depart := time.Now().AddDate(0, 0, 30)
	ret := depart.AddDate(0, 0, 3)
	hacks := detectBackToBack(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		Date:        depart.Format("2006-01-02"),
		ReturnDate:  ret.Format("2006-01-02"),
		Currency:    "GBP",
	})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack, got %d", len(hacks))
	}
	if hacks[0].Currency != "GBP" {
		t.Errorf("expected GBP currency, got %q", hacks[0].Currency)
	}
}

func TestDetectBackToBack_descriptionContainsRoute(t *testing.T) {
	depart := time.Now().AddDate(0, 0, 30)
	ret := depart.AddDate(0, 0, 5)
	hacks := detectBackToBack(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
		Date:        depart.Format("2006-01-02"),
		ReturnDate:  ret.Format("2006-01-02"),
	})
	if len(hacks) != 1 {
		t.Fatalf("expected 1 hack, got %d", len(hacks))
	}
	desc := hacks[0].Description
	if desc == "" {
		t.Fatal("expected non-empty description")
	}
	// Description should mention both airports.
	if !strings.Contains(desc, "HEL") || !strings.Contains(desc, "BCN") {
		t.Errorf("description should mention origin and destination, got: %s", desc)
	}
}
