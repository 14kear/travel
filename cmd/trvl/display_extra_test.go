package main

import (
	"strings"
	"testing"

	"github.com/MikkoParkkola/trvl/internal/models"
)

// --- flightProviderLabel ---

func TestFlightProviderLabel_Empty(t *testing.T) {
	f := models.FlightResult{Provider: ""}
	if got := flightProviderLabel(f); got != "" {
		t.Errorf("expected empty label for empty provider, got %q", got)
	}
}

func TestFlightProviderLabel_Google(t *testing.T) {
	f := models.FlightResult{Provider: "google_flights"}
	if got := flightProviderLabel(f); got != "Google" {
		t.Errorf("expected 'Google', got %q", got)
	}
}

func TestFlightProviderLabel_GoogleMixedCase(t *testing.T) {
	f := models.FlightResult{Provider: "GOOGLE_FLIGHTS"}
	if got := flightProviderLabel(f); got != "Google" {
		t.Errorf("expected 'Google' for uppercase provider, got %q", got)
	}
}

func TestFlightProviderLabel_Kiwi(t *testing.T) {
	f := models.FlightResult{Provider: "kiwi"}
	if got := flightProviderLabel(f); got != "Kiwi" {
		t.Errorf("expected 'Kiwi', got %q", got)
	}
}

func TestFlightProviderLabel_Duffel(t *testing.T) {
	f := models.FlightResult{Provider: "duffel"}
	if got := flightProviderLabel(f); got != "Duffel" {
		t.Errorf("expected 'Duffel', got %q", got)
	}
}

func TestFlightProviderLabel_Unknown(t *testing.T) {
	f := models.FlightResult{Provider: "some_other_provider"}
	if got := flightProviderLabel(f); got != "some_other_provider" {
		t.Errorf("expected raw provider name, got %q", got)
	}
}

func TestFlightProviderLabel_Whitespace(t *testing.T) {
	f := models.FlightResult{Provider: "  kiwi  "}
	if got := flightProviderLabel(f); got != "Kiwi" {
		t.Errorf("expected 'Kiwi' with trimmed whitespace, got %q", got)
	}
}

// --- flightWarnings ---

func TestFlightWarnings_Empty(t *testing.T) {
	f := models.FlightResult{}
	if got := flightWarnings(f); got != "" {
		t.Errorf("expected empty warning, got %q", got)
	}
}

func TestFlightWarnings_SelfConnect(t *testing.T) {
	f := models.FlightResult{SelfConnect: true}
	got := flightWarnings(f)
	if !strings.Contains(got, "Self-connect") {
		t.Errorf("expected self-connect warning, got %q", got)
	}
}

func TestFlightWarnings_ExplicitWarnings(t *testing.T) {
	f := models.FlightResult{Warnings: []string{"Short layover", "Visa required"}}
	got := flightWarnings(f)
	if !strings.Contains(got, "Short layover") {
		t.Errorf("expected 'Short layover' in warnings, got %q", got)
	}
	if !strings.Contains(got, "Visa required") {
		t.Errorf("expected 'Visa required' in warnings, got %q", got)
	}
}

func TestFlightWarnings_WarningsTakePrecedence(t *testing.T) {
	// Explicit Warnings should be returned, not the SelfConnect fallback.
	f := models.FlightResult{SelfConnect: true, Warnings: []string{"Custom warning"}}
	got := flightWarnings(f)
	if !strings.Contains(got, "Custom warning") {
		t.Errorf("expected explicit warning to take precedence, got %q", got)
	}
}

// --- hotelSourceLabel ---

func TestHotelSourceLabel_Google(t *testing.T) {
	if got := hotelSourceLabel("google_hotels"); got != "Google" {
		t.Errorf("expected 'Google', got %q", got)
	}
}

func TestHotelSourceLabel_Airbnb(t *testing.T) {
	if got := hotelSourceLabel("airbnb"); got != "Airbnb" {
		t.Errorf("expected 'Airbnb', got %q", got)
	}
}

func TestHotelSourceLabel_Booking(t *testing.T) {
	if got := hotelSourceLabel("booking"); got != "Booking" {
		t.Errorf("expected 'Booking', got %q", got)
	}
}

func TestHotelSourceLabel_Trivago(t *testing.T) {
	if got := hotelSourceLabel("trivago"); got != "Trivago" {
		t.Errorf("expected 'Trivago', got %q", got)
	}
}

func TestHotelSourceLabel_Unknown(t *testing.T) {
	if got := hotelSourceLabel("hostelworld"); got != "hostelworld" {
		t.Errorf("expected raw provider name, got %q", got)
	}
}

func TestHotelSourceLabel_CaseInsensitive(t *testing.T) {
	if got := hotelSourceLabel("AIRBNB"); got != "Airbnb" {
		t.Errorf("expected case-insensitive match for Airbnb, got %q", got)
	}
}

func TestHotelSourceLabel_Whitespace(t *testing.T) {
	if got := hotelSourceLabel("  booking  "); got != "Booking" {
		t.Errorf("expected trimmed whitespace match for Booking, got %q", got)
	}
}

// --- hotelSourceLabels ---

func TestHotelSourceLabels_Empty(t *testing.T) {
	h := models.HotelResult{}
	if got := hotelSourceLabels(h); got != "" {
		t.Errorf("expected empty for no sources, got %q", got)
	}
}

func TestHotelSourceLabels_SingleSource(t *testing.T) {
	h := models.HotelResult{
		Sources: []models.PriceSource{{Provider: "google_hotels"}},
	}
	if got := hotelSourceLabels(h); got != "Google" {
		t.Errorf("expected 'Google', got %q", got)
	}
}

func TestHotelSourceLabels_MultipleSources(t *testing.T) {
	h := models.HotelResult{
		Sources: []models.PriceSource{
			{Provider: "google_hotels"},
			{Provider: "booking"},
			{Provider: "airbnb"},
		},
	}
	got := hotelSourceLabels(h)
	if !strings.Contains(got, "Google") {
		t.Errorf("expected Google in sources, got %q", got)
	}
	if !strings.Contains(got, "Booking") {
		t.Errorf("expected Booking in sources, got %q", got)
	}
	if !strings.Contains(got, "Airbnb") {
		t.Errorf("expected Airbnb in sources, got %q", got)
	}
}

func TestHotelSourceLabels_DeduplicatesSameProvider(t *testing.T) {
	h := models.HotelResult{
		Sources: []models.PriceSource{
			{Provider: "google_hotels"},
			{Provider: "google_hotels"},
		},
	}
	got := hotelSourceLabels(h)
	// Should appear only once.
	if strings.Count(got, "Google") != 1 {
		t.Errorf("expected Google to appear exactly once, got %q", got)
	}
}

func TestHotelSourceLabels_EmptyProviderSkipped(t *testing.T) {
	h := models.HotelResult{
		Sources: []models.PriceSource{
			{Provider: ""},
		},
	}
	got := hotelSourceLabels(h)
	if got != "" {
		t.Errorf("expected empty for blank provider, got %q", got)
	}
}
