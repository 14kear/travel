package ground

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/MikkoParkkola/trvl/internal/testutil"
)

// --- Station lookup ---

func TestLookupDigitransitStation(t *testing.T) {
	tests := []struct {
		city     string
		wantName string
		wantOK   bool
	}{
		{"Helsinki", "Helsinki", true},
		{"helsinki", "Helsinki", true},
		{"HELSINKI", "Helsinki", true},
		{"  Helsinki  ", "Helsinki", true},
		{"Tampere", "Tampere", true},
		{"Turku", "Turku", true},
		{"Oulu", "Oulu", true},
		{"Jyväskylä", "Jyväskylä", true},
		{"jyväskylä", "Jyväskylä", true},
		{"jyvaskyla", "Jyväskylä", true}, // ASCII alias
		{"Kuopio", "Kuopio", true},
		{"Lahti", "Lahti", true},
		{"Rovaniemi", "Rovaniemi", true},
		{"Vaasa", "Vaasa", true},
		{"Kouvola", "Kouvola", true},
		{"Seinäjoki", "Seinäjoki", true},
		{"seinajoki", "Seinäjoki", true}, // ASCII alias
		{"Joensuu", "Joensuu", true},
		{"Hämeenlinna", "Hämeenlinna", true},
		{"hameenlinna", "Hämeenlinna", true}, // ASCII alias
		{"", "", false},
		{"Nonexistent", "", false},
		{"Stockholm", "", false},
		{"Amsterdam", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.city, func(t *testing.T) {
			station, ok := LookupDigitransitStation(tt.city)
			if ok != tt.wantOK {
				t.Fatalf("LookupDigitransitStation(%q) ok = %v, want %v", tt.city, ok, tt.wantOK)
			}
			if ok && station.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", station.Name, tt.wantName)
			}
		})
	}
}

func TestHasDigitransitStation(t *testing.T) {
	if !HasDigitransitStation("Helsinki") {
		t.Error("Helsinki should have a Digitransit station")
	}
	if !HasDigitransitStation("jyvaskyla") {
		t.Error("jyvaskyla (ASCII alias) should resolve to a Digitransit station")
	}
	if HasDigitransitStation("Atlantis") {
		t.Error("Atlantis should not have a Digitransit station")
	}
}

func TestAllDigitransitStationsHaveCoordinates(t *testing.T) {
	for city, station := range digitransitStations {
		if station.Lat == 0 {
			t.Errorf("station %q has zero Lat", city)
		}
		if station.Lon == 0 {
			t.Errorf("station %q has zero Lon", city)
		}
		if station.Name == "" {
			t.Errorf("station %q has empty Name", city)
		}
		// All Finnish stations should be within Finland's rough bounding box.
		if station.Lat < 59.5 || station.Lat > 70.5 {
			t.Errorf("station %q Lat %f looks outside Finland", city, station.Lat)
		}
		if station.Lon < 19.0 || station.Lon > 32.0 {
			t.Errorf("station %q Lon %f looks outside Finland", city, station.Lon)
		}
	}
}

// --- VR price lookup ---

func TestLookupVRPrice(t *testing.T) {
	tests := []struct {
		from string
		to   string
		want float64
	}{
		{"Helsinki", "Tampere", 22.50},
		{"Tampere", "Helsinki", 22.50}, // reverse direction
		{"helsinki", "tampere", 22.50}, // case-insensitive
		{"Helsinki", "Turku", 19.90},
		{"Helsinki", "Oulu", 59.90},
		{"Helsinki", "Jyväskylä", 34.90},
		{"Helsinki", "jyvaskyla", 34.90}, // ASCII alias
		{"Helsinki", "Kuopio", 39.90},
		{"Helsinki", "Lahti", 14.90},
		{"Helsinki", "Rovaniemi", 69.90},
		{"Helsinki", "Vaasa", 49.90},
		{"Helsinki", "Kouvola", 16.90},
		{"Tampere", "Turku", 19.90},
		{"Tampere", "Oulu", 39.90},
		{"Tampere", "Jyväskylä", 14.90},
		{"Tampere", "jyvaskyla", 14.90}, // ASCII alias
		{"Helsinki", "Nonexistent", 0},
		{"Stockholm", "Oslo", 0},
	}

	for _, tt := range tests {
		got := lookupVRPrice(tt.from, tt.to)
		if got != tt.want {
			t.Errorf("lookupVRPrice(%q, %q) = %.2f, want %.2f", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestVRPricesMapSymmetry(t *testing.T) {
	// Every entry should resolve in both directions.
	for key, price := range vrPrices {
		parts := strings.SplitN(key, "-", 2)
		if len(parts) != 2 {
			t.Errorf("vrPrices key %q should have exactly one hyphen separator", key)
			continue
		}
		got := lookupVRPrice(parts[0], parts[1])
		if got != price {
			t.Errorf("lookupVRPrice(%q, %q) = %.2f, want %.2f", parts[0], parts[1], got, price)
		}
		gotRev := lookupVRPrice(parts[1], parts[0])
		if gotRev != price {
			t.Errorf("lookupVRPrice reverse (%q, %q) = %.2f, want %.2f", parts[1], parts[0], gotRev, price)
		}
	}
}

// --- Rate limiter ---

func TestDigitransitRateLimiterConfiguration(t *testing.T) {
	assertLimiterConfiguration(t, digitransitLimiter, 12*time.Second, 1)
}

// --- Booking URL ---

func TestBuildVRBookingURL(t *testing.T) {
	u := buildVRBookingURL("Helsinki", "Tampere", "2026-04-10")
	if u == "" {
		t.Fatal("booking URL should not be empty")
	}
	if !strings.Contains(u, "vr.fi") {
		t.Error("should contain vr.fi")
	}
	if !strings.Contains(u, "Helsinki") {
		t.Error("should contain Helsinki")
	}
	if !strings.Contains(u, "Tampere") {
		t.Error("should contain Tampere")
	}
	if !strings.Contains(u, "2026-04-10") {
		t.Error("should contain date")
	}
}

// --- msToISO ---

func TestMsToISO(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{0, ""},
		{1744275600000, "2025-04-10T09:00:00Z"}, // known UTC epoch
	}
	for _, tt := range tests {
		got := msToISO(tt.ms)
		if got != tt.want {
			t.Errorf("msToISO(%d) = %q, want %q", tt.ms, got, tt.want)
		}
	}
}

// --- buildDigitransitQuery ---

func TestBuildDigitransitQuery(t *testing.T) {
	from := digitransitStation{60.1719, 24.9414, "Helsinki"}
	to := digitransitStation{61.4978, 23.7610, "Tampere"}
	q := buildDigitransitQuery(from, to, "2026-04-10")

	if !strings.Contains(q, "60.171900") {
		t.Errorf("query should contain from lat, got: %s", q)
	}
	if !strings.Contains(q, "61.497800") {
		t.Errorf("query should contain to lat, got: %s", q)
	}
	if !strings.Contains(q, "2026-04-10") {
		t.Error("query should contain date")
	}
	if !strings.Contains(q, "RAIL") {
		t.Error("query should filter by RAIL mode")
	}
}

// --- parseDigitransitItineraries ---

func TestParseDigitransitItineraries_Empty(t *testing.T) {
	from := digitransitStation{60.1719, 24.9414, "Helsinki"}
	to := digitransitStation{61.4978, 23.7610, "Tampere"}
	routes := parseDigitransitItineraries(nil, from, to, "2026-04-10", "EUR")
	if len(routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(routes))
	}
}

func TestParseDigitransitItineraries_NoRailLegs(t *testing.T) {
	from := digitransitStation{60.1719, 24.9414, "Helsinki"}
	to := digitransitStation{61.4978, 23.7610, "Tampere"}

	itins := []digitransitItinerary{
		{
			StartTime: 1744275600000,
			EndTime:   1744281600000,
			Duration:  6000,
			Legs: []digitransitLeg{
				{Mode: "BUS", StartTime: 1744275600000, EndTime: 1744281600000,
					From: digitransitStop{Name: "Helsinki"}, To: digitransitStop{Name: "Tampere"}},
			},
		},
	}
	routes := parseDigitransitItineraries(itins, from, to, "2026-04-10", "EUR")
	if len(routes) != 0 {
		t.Errorf("expected 0 routes for BUS-only itinerary, got %d", len(routes))
	}
}

func TestParseDigitransitItineraries_Basic(t *testing.T) {
	from := digitransitStation{60.1719, 24.9414, "Helsinki"}
	to := digitransitStation{61.4978, 23.7610, "Tampere"}

	// Helsinki→Tampere departure 09:00, arrival 10:40 (6000 seconds = 100 min)
	depMS := int64(1744275600000) // 2025-04-10T09:00:00Z
	arrMS := depMS + 6000*1000

	itins := []digitransitItinerary{
		{
			StartTime: depMS,
			EndTime:   arrMS,
			Duration:  6000,
			Legs: []digitransitLeg{
				{
					Mode:      "RAIL",
					StartTime: depMS,
					EndTime:   arrMS,
					From:      digitransitStop{Name: "Helsinki asema", Lat: 60.1719, Lon: 24.9414},
					To:        digitransitStop{Name: "Tampere asema", Lat: 61.4978, Lon: 23.7610},
					Route: &struct {
						ShortName string `json:"shortName"`
						LongName  string `json:"longName"`
					}{ShortName: "IC 27"},
				},
			},
		},
	}

	routes := parseDigitransitItineraries(itins, from, to, "2026-04-10", "EUR")
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}

	r := routes[0]
	if r.Provider != "vr" {
		t.Errorf("Provider = %q, want vr", r.Provider)
	}
	if r.Type != "train" {
		t.Errorf("Type = %q, want train", r.Type)
	}
	if r.Currency != "EUR" {
		t.Errorf("Currency = %q, want EUR", r.Currency)
	}
	if r.Price != 22.50 {
		t.Errorf("Price = %.2f, want 22.50 (Helsinki-Tampere VR fixed fare)", r.Price)
	}
	if r.Duration != 100 {
		t.Errorf("Duration = %d minutes, want 100", r.Duration)
	}
	if r.Transfers != 0 {
		t.Errorf("Transfers = %d, want 0", r.Transfers)
	}
	if r.Departure.City != "Helsinki" {
		t.Errorf("Departure.City = %q, want Helsinki", r.Departure.City)
	}
	if r.Arrival.City != "Tampere" {
		t.Errorf("Arrival.City = %q, want Tampere", r.Arrival.City)
	}
	if r.BookingURL == "" {
		t.Error("BookingURL should not be empty")
	}
	if len(r.Legs) != 1 {
		t.Errorf("Legs = %d, want 1", len(r.Legs))
	}
	if r.Legs[0].Provider != "IC 27" {
		t.Errorf("Legs[0].Provider = %q, want IC 27", r.Legs[0].Provider)
	}
}

func TestParseDigitransitItineraries_MultiLeg(t *testing.T) {
	from := digitransitStation{60.1719, 24.9414, "Helsinki"}
	to := digitransitStation{65.0121, 25.4651, "Oulu"}

	dep1 := int64(1744275600000)
	arr1 := dep1 + 3600*1000
	dep2 := arr1 + 600*1000
	arr2 := dep2 + 9000*1000

	itins := []digitransitItinerary{
		{
			StartTime: dep1,
			EndTime:   arr2,
			Duration:  int((arr2 - dep1) / 1000),
			Legs: []digitransitLeg{
				{Mode: "RAIL", StartTime: dep1, EndTime: arr1,
					From: digitransitStop{Name: "Helsinki asema"}, To: digitransitStop{Name: "Tampere asema"},
					Route: &struct {
						ShortName string `json:"shortName"`
						LongName  string `json:"longName"`
					}{ShortName: "IC 53"}},
				{Mode: "RAIL", StartTime: dep2, EndTime: arr2,
					From: digitransitStop{Name: "Tampere asema"}, To: digitransitStop{Name: "Oulu asema"},
					Route: &struct {
						ShortName string `json:"shortName"`
						LongName  string `json:"longName"`
					}{ShortName: "IC 53"}},
			},
		},
	}

	routes := parseDigitransitItineraries(itins, from, to, "2026-04-10", "EUR")
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	r := routes[0]
	if r.Transfers != 1 {
		t.Errorf("Transfers = %d, want 1", r.Transfers)
	}
	if len(r.Legs) != 2 {
		t.Errorf("Legs = %d, want 2", len(r.Legs))
	}
	// Helsinki-Oulu fixed fare
	if r.Price != 59.90 {
		t.Errorf("Price = %.2f, want 59.90", r.Price)
	}
}

func TestParseDigitransitItineraries_UnknownRoutePrice(t *testing.T) {
	// A route with no entry in vrPrices should have price 0.
	from := digitransitStation{62.6010, 29.7636, "Joensuu"}
	to := digitransitStation{63.0952, 21.6165, "Vaasa"}

	dep := int64(1744275600000)
	arr := dep + 4*3600*1000

	itins := []digitransitItinerary{
		{
			StartTime: dep, EndTime: arr, Duration: 4 * 3600,
			Legs: []digitransitLeg{
				{Mode: "RAIL", StartTime: dep, EndTime: arr,
					From: digitransitStop{Name: "Joensuu asema"}, To: digitransitStop{Name: "Vaasa asema"}},
			},
		},
	}
	routes := parseDigitransitItineraries(itins, from, to, "2026-04-10", "EUR")
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Price != 0 {
		t.Errorf("Price = %.2f, want 0 for unknown route", routes[0].Price)
	}
}

func TestParseDigitransitItineraries_CurrencyUppercased(t *testing.T) {
	from := digitransitStation{60.1719, 24.9414, "Helsinki"}
	to := digitransitStation{61.4978, 23.7610, "Tampere"}
	dep := int64(1744275600000)
	arr := dep + 6000*1000

	itins := []digitransitItinerary{
		{StartTime: dep, EndTime: arr, Duration: 6000,
			Legs: []digitransitLeg{
				{Mode: "RAIL", StartTime: dep, EndTime: arr,
					From: digitransitStop{Name: "Helsinki asema"}, To: digitransitStop{Name: "Tampere asema"}},
			}},
	}
	routes := parseDigitransitItineraries(itins, from, to, "2026-04-10", "eur")
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Currency != "EUR" {
		t.Errorf("Currency = %q, want EUR", routes[0].Currency)
	}
}

// --- SearchDigitransit error cases ---

func TestSearchDigitransit_UnknownFrom(t *testing.T) {
	ctx := context.Background()
	_, err := SearchDigitransit(ctx, "Nonexistent", "Helsinki", "2026-04-10", "EUR")
	if err == nil {
		t.Error("expected error for unknown from station")
	}
}

func TestSearchDigitransit_UnknownTo(t *testing.T) {
	ctx := context.Background()
	_, err := SearchDigitransit(ctx, "Helsinki", "Nonexistent", "2026-04-10", "EUR")
	if err == nil {
		t.Error("expected error for unknown to station")
	}
}

// --- Integration test (skipped in short mode / CI) ---

func TestSearchDigitransit_Integration(t *testing.T) {
	testutil.RequireLiveIntegration(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	travelDate := time.Now().AddDate(0, 0, 5).Format("2006-01-02")

	routes, err := SearchDigitransit(ctx, "Helsinki", "Tampere", travelDate, "EUR")
	if err != nil {
		t.Skipf("Digitransit API unavailable (expected in CI): %v", err)
	}
	if len(routes) == 0 {
		t.Skip("no VR routes found (may be outside booking window)")
	}

	r := routes[0]
	if r.Provider != "vr" {
		t.Errorf("provider = %q, want vr", r.Provider)
	}
	if r.Type != "train" {
		t.Errorf("type = %q, want train", r.Type)
	}
	if r.Duration <= 0 {
		t.Errorf("duration = %d, should be > 0", r.Duration)
	}
	if r.Departure.City != "Helsinki" {
		t.Errorf("departure city = %q, want Helsinki", r.Departure.City)
	}
	if r.Arrival.City != "Tampere" {
		t.Errorf("arrival city = %q, want Tampere", r.Arrival.City)
	}
	if r.Price != 22.50 {
		t.Errorf("price = %.2f, want 22.50", r.Price)
	}
	if r.BookingURL == "" {
		t.Error("booking URL should not be empty")
	}
}
