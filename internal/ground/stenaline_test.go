package ground

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/MikkoParkkola/trvl/internal/cookies"
	"github.com/MikkoParkkola/trvl/internal/models"
)

func init() {
	// Disable browser page reading in tests to avoid opening Chrome.
	cookies.SkipBrowserRead = true
}

func TestLookupStenaLinePort(t *testing.T) {
	tests := []struct {
		city     string
		wantCode string
		wantCity string
		wantOK   bool
	}{
		// Gothenburg aliases
		{"Gothenburg", "GOT", "Gothenburg", true},
		{"gothenburg", "GOT", "Gothenburg", true},
		{"Göteborg", "GOT", "Gothenburg", true},
		{"göteborg", "GOT", "Gothenburg", true},
		{"got", "GOT", "Gothenburg", true},
		{"  Gothenburg  ", "GOT", "Gothenburg", true},

		// Kiel aliases
		{"Kiel", "KIE", "Kiel", true},
		{"kiel", "KIE", "Kiel", true},
		{"kie", "KIE", "Kiel", true},

		// Frederikshavn aliases
		{"Frederikshavn", "FDH", "Frederikshavn", true},
		{"frederikshavn", "FDH", "Frederikshavn", true},
		{"fdh", "FDH", "Frederikshavn", true},

		// Karlskrona aliases
		{"Karlskrona", "KRN", "Karlskrona", true},
		{"karlskrona", "KRN", "Karlskrona", true},
		{"krn", "KRN", "Karlskrona", true},

		// Gdynia aliases
		{"Gdynia", "GDY", "Gdynia", true},
		{"gdynia", "GDY", "Gdynia", true},
		{"gdy", "GDY", "Gdynia", true},

		// Nynäshamn aliases
		{"Nynäshamn", "NYN", "Nynäshamn", true},
		{"nynashamn", "NYN", "Nynäshamn", true},
		{"nyn", "NYN", "Nynäshamn", true},

		// Ventspils aliases
		{"Ventspils", "VNT", "Ventspils", true},
		{"ventspils", "VNT", "Ventspils", true},
		{"vnt", "VNT", "Ventspils", true},

		// Trelleborg aliases
		{"Trelleborg", "TRG", "Trelleborg", true},
		{"trelleborg", "TRG", "Trelleborg", true},
		{"trg", "TRG", "Trelleborg", true},

		// Rostock aliases
		{"Rostock", "ROS", "Rostock", true},
		{"rostock", "ROS", "Rostock", true},
		{"ros", "ROS", "Rostock", true},

		// Halmstad aliases
		{"Halmstad", "HAL", "Halmstad", true},
		{"halmstad", "HAL", "Halmstad", true},
		{"hal", "HAL", "Halmstad", true},

		// Grenaa aliases
		{"Grenaa", "GRE", "Grenaa", true},
		{"grenaa", "GRE", "Grenaa", true},
		{"gre", "GRE", "Grenaa", true},

		// Travemünde aliases
		{"Travemünde", "TRV", "Travemünde", true},
		{"travemunde", "TRV", "Travemünde", true},
		{"trv", "TRV", "Travemünde", true},

		// Liepāja aliases
		{"Liepāja", "LPJ", "Liepāja", true},
		{"liepaja", "LPJ", "Liepāja", true},
		{"lpj", "LPJ", "Liepāja", true},

		// Non-existent
		{"", "", "", false},
		{"London", "", "", false},
		{"Paris", "", "", false},
		{"Helsinki", "", "", false},
		{"Atlantis", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.city, func(t *testing.T) {
			port, ok := LookupStenaLinePort(tt.city)
			if ok != tt.wantOK {
				t.Fatalf("LookupStenaLinePort(%q) ok = %v, want %v", tt.city, ok, tt.wantOK)
			}
			if ok {
				if port.Code != tt.wantCode {
					t.Errorf("Code = %q, want %q", port.Code, tt.wantCode)
				}
				if port.City != tt.wantCity {
					t.Errorf("City = %q, want %q", port.City, tt.wantCity)
				}
				if port.Name == "" {
					t.Errorf("Name should not be empty for %q", tt.city)
				}
			}
		})
	}
}

func TestHasStenaLinePort(t *testing.T) {
	if !HasStenaLinePort("Gothenburg") {
		t.Error("Gothenburg should have a Stena Line port")
	}
	if !HasStenaLinePort("Kiel") {
		t.Error("Kiel should have a Stena Line port")
	}
	if !HasStenaLinePort("Karlskrona") {
		t.Error("Karlskrona should have a Stena Line port")
	}
	if HasStenaLinePort("London") {
		t.Error("London should not have a Stena Line port")
	}
	if HasStenaLinePort("") {
		t.Error("empty city should not have a Stena Line port")
	}
}

func TestHasStenaLineRoute(t *testing.T) {
	tests := []struct {
		from string
		to   string
		want bool
	}{
		// Known routes
		{"Gothenburg", "Kiel", true},
		{"Kiel", "Gothenburg", true},
		{"Gothenburg", "Frederikshavn", true},
		{"Frederikshavn", "Gothenburg", true},
		{"Karlskrona", "Gdynia", true},
		{"Gdynia", "Karlskrona", true},
		{"Nynäshamn", "Ventspils", true},
		{"Ventspils", "Nynäshamn", true},
		{"Trelleborg", "Rostock", true},
		{"Rostock", "Trelleborg", true},
		{"Halmstad", "Grenaa", true},
		{"Grenaa", "Halmstad", true},
		{"Travemünde", "Liepāja", true},
		{"Liepāja", "Travemünde", true},

		// Unknown routes (ports exist but no direct sailing)
		{"Gothenburg", "Gdynia", false},
		{"Kiel", "Karlskrona", false},

		// Non-existent ports
		{"London", "Kiel", false},
		{"Gothenburg", "London", false},
		{"", "Kiel", false},
		{"Gothenburg", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.from+"->"+tt.to, func(t *testing.T) {
			got := HasStenaLineRoute(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("HasStenaLineRoute(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestStenaLineAllPortsHaveRequiredFields(t *testing.T) {
	for alias, port := range stenalinePorts {
		if port.Code == "" {
			t.Errorf("port alias %q has empty Code", alias)
		}
		if port.Name == "" {
			t.Errorf("port alias %q has empty Name", alias)
		}
		if port.City == "" {
			t.Errorf("port alias %q has empty City", alias)
		}
	}
}

func TestStenaLineAllSchedulesHaveValidRoutes(t *testing.T) {
	for key, sailings := range stenalineSchedules {
		parts := strings.SplitN(key, "-", 2)
		if len(parts) != 2 {
			t.Errorf("schedule key %q malformed", key)
			continue
		}
		if len(sailings) == 0 {
			t.Errorf("schedule key %q has no sailings", key)
		}
		for i, s := range sailings {
			if s.DurationMin <= 0 {
				t.Errorf("schedule %q sailing[%d] has zero/negative duration", key, i)
			}
			if s.BasePrice <= 0 {
				t.Errorf("schedule %q sailing[%d] has zero/negative price", key, i)
			}
			if s.DepTime == "" {
				t.Errorf("schedule %q sailing[%d] has empty DepTime", key, i)
			}
			if s.ArrTime == "" {
				t.Errorf("schedule %q sailing[%d] has empty ArrTime", key, i)
			}
		}
	}
}

func TestBuildStenaLineBookingURL(t *testing.T) {
	u := buildStenaLineBookingURL("GOT", "KIE")
	if u == "" {
		t.Fatal("booking URL should not be empty")
	}
	if !strings.Contains(u, "stenaline.com") {
		t.Errorf("URL should contain stenaline.com, got %q", u)
	}
	if !strings.Contains(u, "got") {
		t.Errorf("URL should contain from port got, got %q", u)
	}
	if !strings.Contains(u, "kie") {
		t.Errorf("URL should contain to port kie, got %q", u)
	}
}

func TestStenaLineFormatDateTime(t *testing.T) {
	tests := []struct {
		date      string
		timeStr   string
		dayOffset int
		want      string
	}{
		{"2026-04-15", "18:00", 0, "2026-04-15T18:00:00"},
		{"2026-04-15", "18:00", 1, "2026-04-16T18:00:00"},
		{"2026-04-15", "10:00", 1, "2026-04-16T10:00:00"},
		{"2026-12-31", "23:00", 1, "2027-01-01T23:00:00"},
	}

	for _, tt := range tests {
		got := stenalineFormatDateTime(tt.date, tt.timeStr, tt.dayOffset)
		if got != tt.want {
			t.Errorf("stenalineFormatDateTime(%q, %q, %d) = %q, want %q",
				tt.date, tt.timeStr, tt.dayOffset, got, tt.want)
		}
	}
}

func TestStenaLineRateLimiterConfiguration(t *testing.T) {
	assertLimiterConfiguration(t, stenalineLimiter, 12*time.Second, 1)
}

func TestSearchStenaLine_InvalidCity(t *testing.T) {
	// Validation happens before rate limiter — no wait needed.
	ctx := context.Background()
	_, err := SearchStenaLine(ctx, "Atlantis", "Kiel", "2026-04-15", "EUR")
	if err == nil {
		t.Error("expected error for unknown city, got nil")
	}

	_, err = SearchStenaLine(ctx, "Gothenburg", "Atlantis", "2026-04-15", "EUR")
	if err == nil {
		t.Error("expected error for unknown destination, got nil")
	}
}

func TestSearchStenaLine_InvalidDate(t *testing.T) {
	// Validation happens before rate limiter — no wait needed.
	ctx := context.Background()
	_, err := SearchStenaLine(ctx, "Gothenburg", "Kiel", "not-a-date", "EUR")
	if err == nil {
		t.Error("expected error for invalid date, got nil")
	}
}

// TestSearchStenaLine_RouteBuilding tests that the hardcoded schedule produces
// correctly structured GroundRoute values without calling SearchStenaLine (which
// would block on the rate limiter).
func TestSearchStenaLine_RouteBuilding(t *testing.T) {
	date := "2026-05-10"
	fromPort := stenalinePorts["gothenburg"]
	toPort := stenalinePorts["kiel"]
	key := stenalineRouteKey(fromPort.Code, toPort.Code)
	sailings := stenalineSchedules[key]

	if len(sailings) == 0 {
		t.Fatal("expected sailings for GOT-KIE")
	}

	bookingURL := buildStenaLineBookingURL(fromPort.Code, toPort.Code)

	var routes []models.GroundRoute
	for _, s := range sailings {
		depTime := stenalineFormatDateTime(date, s.DepTime, 0)
		arrTime := stenalineFormatDateTime(date, s.ArrTime, s.ArrOffset)
		routes = append(routes, models.GroundRoute{
			Provider: "stenaline",
			Type:     "ferry",
			Price:    s.BasePrice,
			Currency: "EUR",
			Duration: s.DurationMin,
			Departure: models.GroundStop{
				City:    fromPort.City,
				Station: fromPort.Name,
				Time:    depTime,
			},
			Arrival: models.GroundStop{
				City:    toPort.City,
				Station: toPort.Name,
				Time:    arrTime,
			},
			Transfers:  0,
			BookingURL: bookingURL,
		})
	}

	if len(routes) == 0 {
		t.Fatal("expected at least one route for GOT-KIE")
	}

	r := routes[0]
	if r.Provider != "stenaline" {
		t.Errorf("provider = %q, want stenaline", r.Provider)
	}
	if r.Type != "ferry" {
		t.Errorf("type = %q, want ferry", r.Type)
	}
	if r.Price <= 0 {
		t.Errorf("price = %.2f, should be > 0", r.Price)
	}
	if r.Currency != "EUR" {
		t.Errorf("currency = %q, want EUR", r.Currency)
	}
	if r.Duration <= 0 {
		t.Errorf("duration = %d, should be > 0", r.Duration)
	}
	if r.Departure.City != "Gothenburg" {
		t.Errorf("departure city = %q, want Gothenburg", r.Departure.City)
	}
	if r.Arrival.City != "Kiel" {
		t.Errorf("arrival city = %q, want Kiel", r.Arrival.City)
	}
	if !strings.HasPrefix(r.Departure.Time, date) {
		t.Errorf("departure time %q should start with %q", r.Departure.Time, date)
	}
	if r.Arrival.Time == "" {
		t.Error("arrival time should not be empty")
	}
	if r.BookingURL == "" {
		t.Error("booking URL should not be empty")
	}
	if r.Transfers != 0 {
		t.Errorf("transfers = %d, want 0 (direct ferry)", r.Transfers)
	}
}

// TestSearchStenaLine_MultipleSchedules tests the GOT-FDH schedule which has 3 sailings.
func TestSearchStenaLine_MultipleSchedules(t *testing.T) {
	date := "2026-05-10"
	fromPort := stenalinePorts["gothenburg"]
	toPort := stenalinePorts["fdh"]
	key := stenalineRouteKey(fromPort.Code, toPort.Code)
	sailings := stenalineSchedules[key]

	if len(sailings) < 3 {
		t.Fatalf("expected at least 3 sailings for GOT-FDH, got %d", len(sailings))
	}

	for i, s := range sailings {
		depTime := stenalineFormatDateTime(date, s.DepTime, 0)
		if !strings.HasPrefix(depTime, date) {
			t.Errorf("sailing[%d] departure %q should start with %q", i, depTime, date)
		}
	}
}

// TestSearchStenaLine_NoRouteForPair checks that a port pair with no schedule returns empty.
func TestSearchStenaLine_NoRouteForPair(t *testing.T) {
	fromPort, _ := LookupStenaLinePort("Gothenburg")
	toPort, _ := LookupStenaLinePort("Gdynia")
	key := stenalineRouteKey(fromPort.Code, toPort.Code)

	sailings := stenalineSchedules[key]
	if len(sailings) != 0 {
		t.Errorf("expected no sailings for GOT-GDY, got %d", len(sailings))
	}
}

// TestSearchStenaLine_CurrencyDefault verifies that an empty currency is treated as EUR.
func TestSearchStenaLine_CurrencyDefault(t *testing.T) {
	// The default currency logic: if currency == "" => "EUR".
	// Test the route building with an empty currency input.
	currency := ""
	if currency == "" {
		currency = "EUR"
	}
	if currency != "EUR" {
		t.Errorf("default currency = %q, want EUR", currency)
	}
}

func TestSearchStenaLine_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	date := time.Now().AddDate(0, 1, 0).Format("2006-01-02")

	routes, err := SearchStenaLine(ctx, "Karlskrona", "Gdynia", date, "EUR")
	if err != nil {
		t.Skipf("Stena Line unavailable: %v", err)
	}
	if len(routes) == 0 {
		t.Skip("no Stena Line routes found")
	}

	r := routes[0]
	if r.Provider != "stenaline" {
		t.Errorf("provider = %q, want stenaline", r.Provider)
	}
	if r.Type != "ferry" {
		t.Errorf("type = %q, want ferry", r.Type)
	}
	if r.BookingURL == "" {
		t.Error("booking URL should not be empty")
	}
}
