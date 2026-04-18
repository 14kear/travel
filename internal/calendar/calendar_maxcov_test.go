package calendar

import (
	"testing"

	"github.com/MikkoParkkola/trvl/internal/trips"
)

func TestEmojiFor_AllTypes(t *testing.T) {
	tests := []struct {
		legType string
		want    string
	}{
		{"flight", "✈️"},
		{"train", "🚆"},
		{"bus", "🚌"},
		{"ferry", "⛴️"},
		{"hotel", "🏨"},
		{"activity", "🎯"},
		{"unknown", "📍"},
		{"", "📍"},
	}
	for _, tt := range tests {
		t.Run(tt.legType, func(t *testing.T) {
			got := emojiFor(tt.legType)
			if got != tt.want {
				t.Errorf("emojiFor(%q) = %q, want %q", tt.legType, got, tt.want)
			}
		})
	}
}

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"flight", "Flight"},
		{"hotel", "Hotel"},
		{"", ""},
		{"A", "A"},
		{"already Capitalized", "Already Capitalized"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := capitalizeFirst(tt.input)
			if got != tt.want {
				t.Errorf("capitalizeFirst(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSummaryFor_AllTypes(t *testing.T) {
	t.Run("hotel_with_provider", func(t *testing.T) {
		leg := trips.TripLeg{Type: "hotel", Provider: "Hilton", To: "Paris"}
		got := summaryFor(leg)
		if got == "" {
			t.Error("should not be empty")
		}
	})

	t.Run("hotel_without_provider", func(t *testing.T) {
		leg := trips.TripLeg{Type: "hotel", To: "Paris"}
		got := summaryFor(leg)
		if got == "" {
			t.Error("should not be empty")
		}
	})

	t.Run("flight_from_to", func(t *testing.T) {
		leg := trips.TripLeg{Type: "flight", From: "Helsinki", To: "Paris"}
		got := summaryFor(leg)
		if got == "" {
			t.Error("should not be empty")
		}
	})

	t.Run("flight_to_only", func(t *testing.T) {
		leg := trips.TripLeg{Type: "flight", To: "Paris"}
		got := summaryFor(leg)
		if got == "" {
			t.Error("should not be empty")
		}
	})

	t.Run("flight_no_destinations", func(t *testing.T) {
		leg := trips.TripLeg{Type: "flight"}
		got := summaryFor(leg)
		if got == "" {
			t.Error("should not be empty")
		}
	})

	t.Run("bus", func(t *testing.T) {
		leg := trips.TripLeg{Type: "bus", From: "A", To: "B"}
		got := summaryFor(leg)
		if got == "" {
			t.Error("should not be empty")
		}
	})

	t.Run("activity", func(t *testing.T) {
		leg := trips.TripLeg{Type: "activity", To: "Museum"}
		got := summaryFor(leg)
		if got == "" {
			t.Error("should not be empty")
		}
	})
}

func TestLocationFor(t *testing.T) {
	t.Run("hotel_with_provider", func(t *testing.T) {
		leg := trips.TripLeg{Type: "hotel", Provider: "Hilton", To: "Paris"}
		got := locationFor(leg)
		if got != "Hilton, Paris" {
			t.Errorf("locationFor = %q, want Hilton, Paris", got)
		}
	})

	t.Run("hotel_no_provider", func(t *testing.T) {
		leg := trips.TripLeg{Type: "hotel", To: "Paris"}
		got := locationFor(leg)
		if got != "Paris" {
			t.Errorf("locationFor = %q, want Paris", got)
		}
	})

	t.Run("flight", func(t *testing.T) {
		leg := trips.TripLeg{Type: "flight", To: "London"}
		got := locationFor(leg)
		if got != "London" {
			t.Errorf("locationFor = %q, want London", got)
		}
	})
}

func TestDescriptionFor(t *testing.T) {
	t.Run("full", func(t *testing.T) {
		leg := trips.TripLeg{
			Provider:   "Finnair",
			Reference:  "AY123",
			Price:      199.50,
			Currency:   "EUR",
			BookingURL: "https://finnair.com",
		}
		got := descriptionFor(leg)
		if got == "" {
			t.Error("should not be empty")
		}
	})

	t.Run("minimal", func(t *testing.T) {
		leg := trips.TripLeg{}
		got := descriptionFor(leg)
		if got == "" {
			t.Error("should not be empty (always has generated-by footer)")
		}
	})
}
