package flights

import "testing"

func TestMapDuffelOffer_Direct(t *testing.T) {
	offer := duffelOffer{
		TotalAmount:      "145.50",
		TotalCurrency:    "USD",
		TotalEmissionsKG: "471",
		Slices: []duffelSlice{
			{
				Duration:      "PT7H58M",
				FareBrandName: "Main Cabin",
				Segments: []duffelSegment{
					{
						Origin:      duffelPlace{IATACode: "LHR", Name: "Heathrow"},
						Destination: duffelPlace{IATACode: "JFK", Name: "John F. Kennedy International"},
						DepartingAt: "2026-07-01T10:00:00Z",
						ArrivingAt:  "2026-07-01T17:58:00Z",
						Duration:    "PT7H58M",
						MarketingCarrier: &duffelCarrier{
							IATACode: "BA",
							Name:     "British Airways",
						},
						MarketingCarrierFlightNumber: "117",
						Aircraft:                     &duffelAircraft{Name: "Boeing 777"},
						Passengers: []duffelSegmentPassenger{
							{
								Baggages: []duffelBaggage{
									{Type: "carry_on", Quantity: 1},
									{Type: "checked", Quantity: 1},
								},
							},
						},
					},
				},
			},
		},
	}

	got, ok := mapDuffelOffer(offer)
	if !ok {
		t.Fatal("expected Duffel offer to map successfully")
	}
	if got.Provider != "duffel" {
		t.Fatalf("provider = %q, want duffel", got.Provider)
	}
	if got.Currency != "USD" {
		t.Fatalf("currency = %q, want USD", got.Currency)
	}
	if got.Price != 145.50 {
		t.Fatalf("price = %v, want 145.50", got.Price)
	}
	if got.Duration != 478 {
		t.Fatalf("duration = %d, want 478", got.Duration)
	}
	if got.Stops != 0 {
		t.Fatalf("stops = %d, want 0", got.Stops)
	}
	if got.Emissions != 471000 {
		t.Fatalf("emissions = %d, want 471000", got.Emissions)
	}
	if len(got.Legs) != 1 {
		t.Fatalf("legs = %d, want 1", len(got.Legs))
	}
	if got.Legs[0].FlightNumber != "BA 117" {
		t.Fatalf("flight number = %q, want BA 117", got.Legs[0].FlightNumber)
	}
	if got.CarryOnIncluded == nil || !*got.CarryOnIncluded {
		t.Fatalf("expected carry-on included, got %v", got.CarryOnIncluded)
	}
	if got.CheckedBagsIncluded == nil || *got.CheckedBagsIncluded != 1 {
		t.Fatalf("expected 1 checked bag, got %v", got.CheckedBagsIncluded)
	}
}

func TestMapDuffelOffer_RoundTripUsesSliceDurations(t *testing.T) {
	offer := duffelOffer{
		TotalAmount:   "220.00",
		TotalCurrency: "EUR",
		Slices: []duffelSlice{
			{
				Duration: "PT2H00M",
				Segments: []duffelSegment{
					{
						Origin:      duffelPlace{IATACode: "HEL", Name: "Helsinki"},
						Destination: duffelPlace{IATACode: "AMS", Name: "Amsterdam"},
						DepartingAt: "2026-07-01T08:00:00+03:00",
						ArrivingAt:  "2026-07-01T09:00:00+02:00",
						Duration:    "PT2H00M",
						OperatingCarrier: &duffelCarrier{
							IATACode: "KL",
							Name:     "KLM",
						},
						OperatingCarrierFlightNumber: "1250",
					},
				},
			},
			{
				Duration: "PT2H30M",
				Segments: []duffelSegment{
					{
						Origin:      duffelPlace{IATACode: "AMS", Name: "Amsterdam"},
						Destination: duffelPlace{IATACode: "HEL", Name: "Helsinki"},
						DepartingAt: "2026-07-10T18:00:00+02:00",
						ArrivingAt:  "2026-07-10T21:30:00+03:00",
						Duration:    "PT2H30M",
						OperatingCarrier: &duffelCarrier{
							IATACode: "KL",
							Name:     "KLM",
						},
						OperatingCarrierFlightNumber: "1257",
					},
				},
			},
		},
	}

	got, ok := mapDuffelOffer(offer)
	if !ok {
		t.Fatal("expected Duffel round-trip offer to map successfully")
	}
	if got.Duration != 270 {
		t.Fatalf("duration = %d, want 270", got.Duration)
	}
	if got.Stops != 0 {
		t.Fatalf("stops = %d, want 0 for two direct slices", got.Stops)
	}
	if len(got.Legs) != 2 {
		t.Fatalf("legs = %d, want 2", len(got.Legs))
	}
	if got.Legs[1].LayoverMinutes != 0 {
		t.Fatalf("expected no artificial layover between outbound and return, got %d", got.Legs[1].LayoverMinutes)
	}
}

func TestDuffelOfferMatchesOptions_ExcludeBasic(t *testing.T) {
	offer := duffelOffer{
		Slices: []duffelSlice{{FareBrandName: "Basic Economy"}},
	}
	if duffelOfferMatchesOptions(offer, SearchOptions{ExcludeBasic: true}) {
		t.Fatal("expected basic fare to be filtered out")
	}
}

func TestDuffelBaggageAllowance_UnknownWhenNoPassengerData(t *testing.T) {
	offer := duffelOffer{
		Slices: []duffelSlice{
			{
				Segments: []duffelSegment{{}},
			},
		},
	}

	carryOnIncluded, checkedBagsIncluded := duffelBaggageAllowance(offer)
	if carryOnIncluded != nil {
		t.Fatalf("expected nil carry-on info, got %v", carryOnIncluded)
	}
	if checkedBagsIncluded != nil {
		t.Fatalf("expected nil checked bag info, got %v", checkedBagsIncluded)
	}
}

func TestDuffelSearchEnabled_DefaultOff(t *testing.T) {
	t.Setenv("TRVL_ENABLE_DUFFEL", "")
	if duffelSearchEnabled() {
		t.Fatal("expected Duffel search to be disabled by default")
	}
}

func TestDuffelSearchEnabled_TrueValues(t *testing.T) {
	for _, value := range []string{"true", "1", "yes", "on", " TRUE "} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("TRVL_ENABLE_DUFFEL", value)
			if !duffelSearchEnabled() {
				t.Fatalf("expected %q to enable Duffel search", value)
			}
		})
	}
}
