package ground

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// ============================================================
// SearchOebb via httptest (was 0%)
// ============================================================

func TestSearchOebb_MockHappyPath(t *testing.T) {
	origClient := oebbClient
	origLimiter := oebbLimiter
	t.Cleanup(func() {
		oebbClient = origClient
		oebbLimiter = origLimiter
	})
	oebbLimiter = rate.NewLimiter(rate.Limit(1000), 1)

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "anonymousToken"):
			json.NewEncoder(w).Encode(oebbShopAnonymousTokenResponse{
				AccessToken: "test-token-123",
			})
		case strings.Contains(r.URL.Path, "initUserData"):
			w.WriteHeader(http.StatusNoContent)
		case strings.Contains(r.URL.Path, "timetable"):
			json.NewEncoder(w).Encode(oebbShopTimetableResponse{
				Connections: []oebbShopConnection{
					{
						ID: "conn-1",
						From: struct {
							Departure string `json:"departure"`
						}{Departure: "2026-07-01T08:00:00+02:00"},
						To: struct {
							Arrival string `json:"arrival"`
						}{Arrival: "2026-07-01T12:13:00+02:00"},
						Duration: 15180000, // 253 min in ms
					},
				},
			})
		case strings.Contains(r.URL.Path, "prices"):
			json.NewEncoder(w).Encode(oebbShopPricesResponse{
				Offers: []oebbShopOffer{
					{ConnectionID: "conn-1", Price: 29.90, FirstClass: false},
					{ConnectionID: "conn-1", Price: 49.90, FirstClass: true},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oebbClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	routes, err := SearchOebb(context.Background(), "Vienna", "Munich", "2026-07-01", "EUR")
	if err != nil {
		t.Fatalf("SearchOebb: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("expected at least 1 route")
	}
	r := routes[0]
	if r.Provider != "oebb" {
		t.Errorf("provider = %q, want oebb", r.Provider)
	}
	if r.Type != "train" {
		t.Errorf("type = %q, want train", r.Type)
	}
	if r.Price != 29.90 {
		t.Errorf("price = %v, want 29.90 (cheapest 2nd class)", r.Price)
	}
	if r.Currency != "EUR" {
		t.Errorf("currency = %q, want EUR", r.Currency)
	}
	if r.Departure.City != "Vienna" {
		t.Errorf("departure city = %q, want Vienna", r.Departure.City)
	}
	if r.Arrival.City != "Munich" {
		t.Errorf("arrival city = %q, want Munich", r.Arrival.City)
	}
}

func TestSearchOebb_MockUnknownCity(t *testing.T) {
	_, err := SearchOebb(context.Background(), "NoSuchCityXYZ", "Munich", "2026-07-01", "EUR")
	if err == nil {
		t.Fatal("expected error for unknown city")
	}
}

func TestSearchOebb_MockUnknownToCity(t *testing.T) {
	_, err := SearchOebb(context.Background(), "Vienna", "NoSuchCityXYZ", "2026-07-01", "EUR")
	if err == nil {
		t.Fatal("expected error for unknown to city")
	}
}

func TestSearchOebb_MockInvalidDate(t *testing.T) {
	_, err := SearchOebb(context.Background(), "Vienna", "Munich", "not-a-date", "EUR")
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestSearchOebb_MockDefaultCurrency(t *testing.T) {
	origClient := oebbClient
	origLimiter := oebbLimiter
	t.Cleanup(func() {
		oebbClient = origClient
		oebbLimiter = origLimiter
	})
	oebbLimiter = rate.NewLimiter(rate.Limit(1000), 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "anonymousToken"):
			json.NewEncoder(w).Encode(oebbShopAnonymousTokenResponse{AccessToken: "tok"})
		case strings.Contains(r.URL.Path, "initUserData"):
			w.WriteHeader(http.StatusNoContent)
		case strings.Contains(r.URL.Path, "timetable"):
			json.NewEncoder(w).Encode(oebbShopTimetableResponse{})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	oebbClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	routes, err := SearchOebb(context.Background(), "Vienna", "Munich", "2026-07-01", "")
	if err != nil {
		t.Fatalf("SearchOebb: %v", err)
	}
	// No connections returned => nil routes, no error.
	_ = routes
}

func TestSearchOebb_MockTokenError(t *testing.T) {
	origClient := oebbClient
	origLimiter := oebbLimiter
	t.Cleanup(func() {
		oebbClient = origClient
		oebbLimiter = origLimiter
	})
	oebbLimiter = rate.NewLimiter(rate.Limit(1000), 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	oebbClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	_, err := SearchOebb(context.Background(), "Vienna", "Munich", "2026-07-01", "EUR")
	if err == nil {
		t.Fatal("expected error for token failure")
	}
}

// ============================================================
// SearchRenfe via httptest (was 0%)
// ============================================================

func TestSearchRenfe_MockHappyPath(t *testing.T) {
	origClient := renfeClient
	origLimiter := renfeLimiter
	t.Cleanup(func() {
		renfeClient = origClient
		renfeLimiter = origLimiter
	})
	renfeLimiter = rate.NewLimiter(rate.Limit(1000), 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(renfePriceCalendarResponse{
			Origin:      renfeStationInfo{Name: "Madrid", ExtID: "60000"},
			Destination: renfeStationInfo{Name: "Barcelona", ExtID: "71801"},
			Journeys: []renfeJourneyEntry{
				{Date: "2026-07-01", MinPriceAvailable: true, MinPrice: 35.50},
				{Date: "2026-07-02", MinPriceAvailable: true, MinPrice: 40.00}, // different date, skipped
			},
		})
	}))
	defer srv.Close()

	renfeClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	routes, err := SearchRenfe(context.Background(), "Madrid", "Barcelona", "2026-07-01", "EUR")
	if err != nil {
		t.Fatalf("SearchRenfe: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	r := routes[0]
	if r.Provider != "renfe" {
		t.Errorf("provider = %q, want renfe", r.Provider)
	}
	if r.Price != 35.50 {
		t.Errorf("price = %v, want 35.50", r.Price)
	}
	if r.BookingURL == "" {
		t.Error("expected non-empty booking URL")
	}
}

func TestSearchRenfe_MockUnknownCity(t *testing.T) {
	_, err := SearchRenfe(context.Background(), "NoSuchCityXYZ", "Barcelona", "2026-07-01", "EUR")
	if err == nil {
		t.Fatal("expected error for unknown city")
	}
}

func TestSearchRenfe_MockUnknownToCity(t *testing.T) {
	_, err := SearchRenfe(context.Background(), "Madrid", "NoSuchCityXYZ", "2026-07-01", "EUR")
	if err == nil {
		t.Fatal("expected error for unknown to city")
	}
}

func TestSearchRenfe_MockInvalidDate(t *testing.T) {
	_, err := SearchRenfe(context.Background(), "Madrid", "Barcelona", "not-a-date", "EUR")
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestSearchRenfe_MockNoNumericID(t *testing.T) {
	// Paris has no Numeric station ID.
	_, err := SearchRenfe(context.Background(), "Paris", "Lyon", "2026-07-01", "EUR")
	if err == nil {
		t.Fatal("expected error for station without numeric ID")
	}
}

func TestSearchRenfe_MockHTTPError(t *testing.T) {
	origClient := renfeClient
	origLimiter := renfeLimiter
	t.Cleanup(func() {
		renfeClient = origClient
		renfeLimiter = origLimiter
	})
	renfeLimiter = rate.NewLimiter(rate.Limit(1000), 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer srv.Close()

	renfeClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	_, err := SearchRenfe(context.Background(), "Madrid", "Barcelona", "2026-07-01", "EUR")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestSearchRenfe_MockNoPricesForDate(t *testing.T) {
	origClient := renfeClient
	origLimiter := renfeLimiter
	t.Cleanup(func() {
		renfeClient = origClient
		renfeLimiter = origLimiter
	})
	renfeLimiter = rate.NewLimiter(rate.Limit(1000), 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(renfePriceCalendarResponse{
			Journeys: []renfeJourneyEntry{
				{Date: "2026-07-01", MinPriceAvailable: false, MinPrice: 0},
			},
		})
	}))
	defer srv.Close()

	renfeClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	routes, err := SearchRenfe(context.Background(), "Madrid", "Barcelona", "2026-07-01", "EUR")
	if err != nil {
		t.Fatalf("SearchRenfe: %v", err)
	}
	if len(routes) != 0 {
		t.Errorf("expected 0 routes for unavailable prices, got %d", len(routes))
	}
}

// ============================================================
// SearchSnalltaget via httptest (was 0%)
// ============================================================

func TestSearchSnalltaget_MockHappyPath(t *testing.T) {
	origClient := snalltagetClient
	origLimiter := snalltagetLimiter
	t.Cleanup(func() {
		snalltagetClient = origClient
		snalltagetLimiter = origLimiter
	})
	snalltagetLimiter = rate.NewLimiter(rate.Limit(1000), 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(snalltagetTripsResponse{
			Trips: []snalltagetTrip{
				{
					DepartureTime: "2026-07-01T18:00:00",
					ArrivalTime:   "2026-07-02T06:30:00",
					Duration:      750,
					Origin:        snalltagetTripStop{Name: "Stockholm Central"},
					Destination:   snalltagetTripStop{Name: "Malmö C"},
					Prices: []snalltagetPrice{
						{Amount: 299, Currency: "SEK", Class: "2nd"},
						{Amount: 499, Currency: "SEK", Class: "1st"},
					},
					Segments: []snalltagetSegment{
						{
							DepartureTime: "2026-07-01T18:00:00",
							ArrivalTime:   "2026-07-02T06:30:00",
							Origin:        snalltagetTripStop{Name: "Stockholm Central"},
							Destination:   snalltagetTripStop{Name: "Malmö C"},
							TrainNumber:   "SN 101",
							Operator:      "Snälltåget",
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	snalltagetClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	routes, err := SearchSnalltaget(context.Background(), "Stockholm", "Malmö", "2026-07-01", "SEK")
	if err != nil {
		t.Fatalf("SearchSnalltaget: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	r := routes[0]
	if r.Provider != "snalltaget" {
		t.Errorf("provider = %q, want snalltaget", r.Provider)
	}
	if r.Price != 299 {
		t.Errorf("price = %v, want 299", r.Price)
	}
	if r.Currency != "SEK" {
		t.Errorf("currency = %q, want SEK", r.Currency)
	}
	if len(r.Legs) != 1 {
		t.Fatalf("expected 1 leg, got %d", len(r.Legs))
	}
}

func TestSearchSnalltaget_MockUnknownCity(t *testing.T) {
	_, err := SearchSnalltaget(context.Background(), "NoSuchCityXYZ", "Malmö", "2026-07-01", "SEK")
	if err == nil {
		t.Fatal("expected error for unknown city")
	}
}

func TestSearchSnalltaget_MockUnknownToCity(t *testing.T) {
	_, err := SearchSnalltaget(context.Background(), "Stockholm", "NoSuchCityXYZ", "2026-07-01", "SEK")
	if err == nil {
		t.Fatal("expected error for unknown to city")
	}
}

func TestSearchSnalltaget_MockDefaultCurrency(t *testing.T) {
	origClient := snalltagetClient
	origLimiter := snalltagetLimiter
	t.Cleanup(func() {
		snalltagetClient = origClient
		snalltagetLimiter = origLimiter
	})
	snalltagetLimiter = rate.NewLimiter(rate.Limit(1000), 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(snalltagetTripsResponse{})
	}))
	defer srv.Close()

	snalltagetClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	// Empty currency defaults to SEK.
	routes, err := SearchSnalltaget(context.Background(), "Stockholm", "Malmö", "2026-07-01", "")
	if err != nil {
		t.Fatalf("SearchSnalltaget: %v", err)
	}
	_ = routes
}

func TestSearchSnalltaget_MockNon200(t *testing.T) {
	origClient := snalltagetClient
	origLimiter := snalltagetLimiter
	t.Cleanup(func() {
		snalltagetClient = origClient
		snalltagetLimiter = origLimiter
	})
	snalltagetLimiter = rate.NewLimiter(rate.Limit(1000), 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	snalltagetClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	// Non-200 returns nil routes, no error (graceful degradation).
	routes, err := SearchSnalltaget(context.Background(), "Stockholm", "Malmö", "2026-07-01", "SEK")
	if err != nil {
		t.Fatalf("SearchSnalltaget: %v", err)
	}
	if len(routes) != 0 {
		t.Errorf("expected 0 routes for 403, got %d", len(routes))
	}
}

func TestSearchSnalltaget_MockDataKey(t *testing.T) {
	origClient := snalltagetClient
	origLimiter := snalltagetLimiter
	t.Cleanup(func() {
		snalltagetClient = origClient
		snalltagetLimiter = origLimiter
	})
	snalltagetLimiter = rate.NewLimiter(rate.Limit(1000), 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Use "data" key instead of "trips".
		json.NewEncoder(w).Encode(snalltagetTripsResponse{
			Data: []snalltagetTrip{
				{
					DepartureTime: "2026-07-01T18:00:00",
					ArrivalTime:   "2026-07-02T06:30:00",
					Duration:      750,
					Fares: []snalltagetPrice{
						{Amount: 199, Currency: "SEK"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	snalltagetClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	routes, err := SearchSnalltaget(context.Background(), "Stockholm", "Malmö", "2026-07-01", "SEK")
	if err != nil {
		t.Fatalf("SearchSnalltaget: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 route via data key, got %d", len(routes))
	}
	if routes[0].Price != 199 {
		t.Errorf("price = %v, want 199", routes[0].Price)
	}
}

// ============================================================
// SearchNS via httptest (was 16.2%)
// ============================================================

func TestSearchNS_MockHappyPath(t *testing.T) {
	origClient := nsClient
	origLimiter := nsLimiter
	t.Cleanup(func() {
		nsClient = origClient
		nsLimiter = origLimiter
	})
	nsLimiter = rate.NewLimiter(rate.Limit(1000), 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nsTripsResponse{
			Trips: []nsTrip{
				{
					Legs: []nsTripLeg{
						{
							Origin: nsStop{
								Name:            "Amsterdam Centraal",
								PlannedDateTime: "2026-07-01T08:00:00+02:00",
							},
							Destination: nsStop{
								Name:            "Rotterdam Centraal",
								PlannedDateTime: "2026-07-01T08:40:00+02:00",
							},
							TrainCategory: "Intercity",
						},
					},
					OptimalPrice:             &nsPrice{TotalPriceInCents: 1740},
					Transfers:                0,
					PlannedDurationInMinutes: 40,
				},
			},
		})
	}))
	defer srv.Close()

	nsClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	routes, err := SearchNS(context.Background(), "Amsterdam", "Rotterdam", "2026-07-01", "EUR")
	if err != nil {
		t.Fatalf("SearchNS: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("expected at least 1 route")
	}
	r := routes[0]
	if r.Provider != "ns" {
		t.Errorf("provider = %q, want ns", r.Provider)
	}
	if r.Price != 17.40 {
		t.Errorf("price = %v, want 17.40", r.Price)
	}
	if r.Duration != 40 {
		t.Errorf("duration = %d, want 40", r.Duration)
	}
	if r.Transfers != 0 {
		t.Errorf("transfers = %d, want 0", r.Transfers)
	}
	if r.BookingURL == "" {
		t.Error("expected non-empty booking URL")
	}
}

func TestSearchNS_MockDefaultCurrency(t *testing.T) {
	origClient := nsClient
	origLimiter := nsLimiter
	t.Cleanup(func() {
		nsClient = origClient
		nsLimiter = origLimiter
	})
	nsLimiter = rate.NewLimiter(rate.Limit(1000), 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nsTripsResponse{})
	}))
	defer srv.Close()

	nsClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	routes, err := SearchNS(context.Background(), "Amsterdam", "Rotterdam", "2026-07-01", "")
	if err != nil {
		t.Fatalf("SearchNS: %v", err)
	}
	_ = routes
}

func TestSearchNS_MockHTTPError(t *testing.T) {
	origClient := nsClient
	origLimiter := nsLimiter
	t.Cleanup(func() {
		nsClient = origClient
		nsLimiter = origLimiter
	})
	nsLimiter = rate.NewLimiter(rate.Limit(1000), 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer srv.Close()

	nsClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	_, err := SearchNS(context.Background(), "Amsterdam", "Rotterdam", "2026-07-01", "EUR")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// ============================================================
// SearchFinnlines via httptest (was 0%)
// ============================================================

func TestSearchFinnlines_MockHappyPath(t *testing.T) {
	origClient := finnlinesClient
	origLimiter := finnlinesLimiter
	t.Cleanup(func() {
		finnlinesClient = origClient
		finnlinesLimiter = origLimiter
	})
	finnlinesLimiter = rate.NewLimiter(rate.Limit(1000), 1)

	charge := 4500
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(finnlinesGraphQLResponse{
			Data: struct {
				ListTimeTableAvailability []finnlinesTimetableEntry `json:"listTimeTableAvailability"`
			}{
				ListTimeTableAvailability: []finnlinesTimetableEntry{
					{
						SailingCode:   "FI001",
						DepartureDate: "2026-07-01",
						DepartureTime: "17:00",
						ArrivalDate:   "2026-07-02",
						ArrivalTime:   "09:30",
						DeparturePort: "FIHEL",
						ArrivalPort:   "DETRV",
						IsAvailable:   true,
						ShipName:      "FINNSTAR",
						CrossingTime:  "29:30",
						ChargeTotal:   &charge,
					},
				},
			},
		})
	}))
	defer srv.Close()

	finnlinesClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	routes, err := SearchFinnlines(context.Background(), "Helsinki", "Travemünde", "2026-07-01", "EUR")
	if err != nil {
		t.Fatalf("SearchFinnlines: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("expected at least 1 route")
	}
	r := routes[0]
	if r.Provider != "finnlines" {
		t.Errorf("provider = %q, want finnlines", r.Provider)
	}
	if r.Type != "ferry" {
		t.Errorf("type = %q, want ferry", r.Type)
	}
	if r.Price != 45.00 {
		t.Errorf("price = %v, want 45.00 (4500 cents / 100)", r.Price)
	}
	if r.Duration != 29*60+30 {
		t.Errorf("duration = %d, want %d", r.Duration, 29*60+30)
	}
}

func TestSearchFinnlines_MockUnknownCity(t *testing.T) {
	_, err := SearchFinnlines(context.Background(), "NoSuchCityXYZ", "Travemünde", "2026-07-01", "EUR")
	if err == nil {
		t.Fatal("expected error for unknown city")
	}
}

func TestSearchFinnlines_MockUnknownToCity(t *testing.T) {
	_, err := SearchFinnlines(context.Background(), "Helsinki", "NoSuchCityXYZ", "2026-07-01", "EUR")
	if err == nil {
		t.Fatal("expected error for unknown to city")
	}
}

func TestSearchFinnlines_MockInvalidDate(t *testing.T) {
	_, err := SearchFinnlines(context.Background(), "Helsinki", "Travemünde", "not-a-date", "EUR")
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestSearchFinnlines_MockHTTPError(t *testing.T) {
	origClient := finnlinesClient
	origLimiter := finnlinesLimiter
	t.Cleanup(func() {
		finnlinesClient = origClient
		finnlinesLimiter = origLimiter
	})
	finnlinesLimiter = rate.NewLimiter(rate.Limit(1000), 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer srv.Close()

	finnlinesClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	_, err := SearchFinnlines(context.Background(), "Helsinki", "Travemünde", "2026-07-01", "EUR")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// ============================================================
// parseEurostarSearchResponse (was 0%)
// ============================================================

func TestParseEurostarSearchResponse_HappyPath(t *testing.T) {
	body := `{
		"data": {
			"cheapestFaresSearch": [{
				"cheapestFares": [
					{"date": "2026-07-01", "price": 39.0},
					{"date": "2026-07-02", "price": 49.0}
				]
			}]
		}
	}`

	fromStation, _ := LookupEurostarStation("London")
	toStation, _ := LookupEurostarStation("Paris")

	routes, err := parseEurostarSearchResponse(
		context.Background(),
		[]byte(body),
		fromStation, toStation,
		"2026-07-01", "GBP", false,
	)
	if err != nil {
		t.Fatalf("parseEurostarSearchResponse: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("expected at least 1 route")
	}
	// Check first route.
	r := routes[0]
	if r.Provider != "eurostar" {
		t.Errorf("provider = %q, want eurostar", r.Provider)
	}
	if r.Currency != "GBP" {
		t.Errorf("currency = %q, want GBP", r.Currency)
	}
}

func TestParseEurostarSearchResponse_InvalidJSON(t *testing.T) {
	fromStation, _ := LookupEurostarStation("London")
	toStation, _ := LookupEurostarStation("Paris")

	_, err := parseEurostarSearchResponse(
		context.Background(),
		[]byte("not json"),
		fromStation, toStation,
		"2026-07-01", "GBP", false,
	)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseEurostarSearchResponse_GraphQLError(t *testing.T) {
	body := `{"errors": [{"message": "something went wrong"}]}`

	fromStation, _ := LookupEurostarStation("London")
	toStation, _ := LookupEurostarStation("Paris")

	_, err := parseEurostarSearchResponse(
		context.Background(),
		[]byte(body),
		fromStation, toStation,
		"2026-07-01", "GBP", false,
	)
	if err == nil {
		t.Fatal("expected error for GraphQL error")
	}
	if !strings.Contains(err.Error(), "something went wrong") {
		t.Errorf("error should contain message, got: %v", err)
	}
}

// ============================================================
// oebbShopGetToken via httptest (was 0%)
// ============================================================

func TestOebbShopGetToken_MockHappyPath(t *testing.T) {
	origClient := oebbClient
	t.Cleanup(func() { oebbClient = origClient })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(oebbShopAnonymousTokenResponse{AccessToken: "test-token"})
	}))
	defer srv.Close()

	oebbClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	token, err := oebbShopGetToken(context.Background())
	if err != nil {
		t.Fatalf("oebbShopGetToken: %v", err)
	}
	if token != "test-token" {
		t.Errorf("token = %q, want test-token", token)
	}
}

func TestOebbShopGetToken_MockEmptyToken(t *testing.T) {
	origClient := oebbClient
	t.Cleanup(func() { oebbClient = origClient })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(oebbShopAnonymousTokenResponse{AccessToken: ""})
	}))
	defer srv.Close()

	oebbClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	_, err := oebbShopGetToken(context.Background())
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

// ============================================================
// oebbShopInitUserData via httptest (was 0%)
// ============================================================

func TestOebbShopInitUserData_MockSuccess(t *testing.T) {
	origClient := oebbClient
	t.Cleanup(func() { oebbClient = origClient })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	oebbClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	err := oebbShopInitUserData(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("oebbShopInitUserData: %v", err)
	}
}

func TestOebbShopInitUserData_MockHTTPError(t *testing.T) {
	origClient := oebbClient
	t.Cleanup(func() { oebbClient = origClient })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	oebbClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	err := oebbShopInitUserData(context.Background(), "test-token")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// ============================================================
// oebbShopSearchTimetable via httptest (was 0%)
// ============================================================

func TestOebbShopSearchTimetable_MockHappyPath(t *testing.T) {
	origClient := oebbClient
	t.Cleanup(func() { oebbClient = origClient })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(oebbShopTimetableResponse{
			Connections: []oebbShopConnection{
				{
					ID: "c1",
					From: struct {
						Departure string `json:"departure"`
					}{Departure: "2026-07-01T08:00:00"},
					To: struct {
						Arrival string `json:"arrival"`
					}{Arrival: "2026-07-01T12:00:00"},
					Duration: 14400000,
				},
			},
		})
	}))
	defer srv.Close()

	oebbClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	from := oebbStation{ExtID: "1190100", Number: 1290401, Name: "Wien Hbf", City: "Vienna"}
	to := oebbStation{ExtID: "8000261", Number: 8000261, Name: "München Hbf", City: "Munich"}

	conns, err := oebbShopSearchTimetable(context.Background(), "test-token", from, to, "2026-07-01T08:00:00.000")
	if err != nil {
		t.Fatalf("oebbShopSearchTimetable: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].ID != "c1" {
		t.Errorf("id = %q, want c1", conns[0].ID)
	}
}

func TestOebbShopSearchTimetable_MockHTTPError(t *testing.T) {
	origClient := oebbClient
	t.Cleanup(func() { oebbClient = origClient })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer srv.Close()

	oebbClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	from := oebbStation{ExtID: "1190100", Number: 1290401, Name: "Wien Hbf", City: "Vienna"}
	to := oebbStation{ExtID: "8000261", Number: 8000261, Name: "München Hbf", City: "Munich"}

	_, err := oebbShopSearchTimetable(context.Background(), "test-token", from, to, "2026-07-01T08:00:00.000")
	if err == nil {
		t.Fatal("expected error for HTTP 400")
	}
}

// ============================================================
// oebbShopGetPrices via httptest (was 0%)
// ============================================================

func TestOebbShopGetPrices_MockHappyPath(t *testing.T) {
	origClient := oebbClient
	t.Cleanup(func() { oebbClient = origClient })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(oebbShopPricesResponse{
			Offers: []oebbShopOffer{
				{ConnectionID: "c1", Price: 29.90, FirstClass: false},
				{ConnectionID: "c1", Price: 49.90, FirstClass: true},
			},
		})
	}))
	defer srv.Close()

	oebbClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	offers, err := oebbShopGetPrices(context.Background(), "test-token", []string{"c1"})
	if err != nil {
		t.Fatalf("oebbShopGetPrices: %v", err)
	}
	if len(offers) != 2 {
		t.Fatalf("expected 2 offers, got %d", len(offers))
	}
}

func TestOebbShopGetPrices_EmptyConnectionIDs(t *testing.T) {
	offers, err := oebbShopGetPrices(context.Background(), "test-token", nil)
	if err != nil {
		t.Fatalf("oebbShopGetPrices: %v", err)
	}
	if offers != nil {
		t.Errorf("expected nil offers for empty IDs, got %d", len(offers))
	}
}

func TestOebbShopGetPrices_MockHTTPError(t *testing.T) {
	origClient := oebbClient
	t.Cleanup(func() { oebbClient = origClient })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}))
	defer srv.Close()

	oebbClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	_, err := oebbShopGetPrices(context.Background(), "test-token", []string{"c1"})
	if err == nil {
		t.Fatal("expected error for HTTP 403")
	}
}

// ============================================================
// fetchFinnlinesTimetablesWithCabin via httptest (was 0%)
// ============================================================

func TestFetchFinnlinesTimetablesWithCabin_MockHappyPath(t *testing.T) {
	origClient := finnlinesClient
	t.Cleanup(func() { finnlinesClient = origClient })

	charge := 8900
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(finnlinesGraphQLResponse{
			Data: struct {
				ListTimeTableAvailability []finnlinesTimetableEntry `json:"listTimeTableAvailability"`
			}{
				ListTimeTableAvailability: []finnlinesTimetableEntry{
					{
						SailingCode:   "FI001",
						DepartureDate: "2026-07-01",
						DepartureTime: "17:00",
						ArrivalDate:   "2026-07-02",
						ArrivalTime:   "09:30",
						IsAvailable:   true,
						ChargeTotal:   &charge,
					},
				},
			},
		})
	}))
	defer srv.Close()

	finnlinesClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	entries, err := fetchFinnlinesTimetablesWithCabin(context.Background(), "FIHEL", "DETRV", "2026-07-01", "B2I")
	if err != nil {
		t.Fatalf("fetchFinnlinesTimetablesWithCabin: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if *entries[0].ChargeTotal != 8900 {
		t.Errorf("chargeTotal = %d, want 8900", *entries[0].ChargeTotal)
	}
}

func TestFetchFinnlinesTimetablesWithCabin_MockGraphQLError(t *testing.T) {
	origClient := finnlinesClient
	t.Cleanup(func() { finnlinesClient = origClient })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"errors":[{"message":"cabin not found"}]}`))
	}))
	defer srv.Close()

	finnlinesClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	_, err := fetchFinnlinesTimetablesWithCabin(context.Background(), "FIHEL", "DETRV", "2026-07-01", "INVALID")
	if err == nil {
		t.Fatal("expected error for GraphQL error")
	}
}

func TestFetchFinnlinesTimetablesWithCabin_MockHTTPError(t *testing.T) {
	origClient := finnlinesClient
	t.Cleanup(func() { finnlinesClient = origClient })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("unavailable"))
	}))
	defer srv.Close()

	finnlinesClient = &http.Client{
		Transport: &redirectTransport{target: srv.URL},
		Timeout:   5 * time.Second,
	}

	_, err := fetchFinnlinesTimetablesWithCabin(context.Background(), "FIHEL", "DETRV", "2026-07-01", "B2I")
	if err == nil {
		t.Fatal("expected error for HTTP 503")
	}
}

// ============================================================
// stenalineRouteKey + stenalineFormatDateTime helpers
// ============================================================

func TestStenalineRouteKey(t *testing.T) {
	tests := []struct {
		from, to string
		want     string
	}{
		{"got", "kie", "GOT-KIE"},
		{"GOT", "KIE", "GOT-KIE"},
		{"fdh", "got", "FDH-GOT"},
	}
	for _, tt := range tests {
		got := stenalineRouteKey(tt.from, tt.to)
		if got != tt.want {
			t.Errorf("stenalineRouteKey(%q, %q) = %q, want %q", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestStenalineFormatDateTime_NextDay(t *testing.T) {
	got := stenalineFormatDateTime("2026-07-01", "04:00", 1)
	want := "2026-07-02T04:00:00"
	if got != want {
		t.Errorf("stenalineFormatDateTime with dayOffset=1: got %q, want %q", got, want)
	}
}

func TestStenalineFormatDateTime_SameDay(t *testing.T) {
	got := stenalineFormatDateTime("2026-07-01", "18:00", 0)
	want := "2026-07-01T18:00:00"
	if got != want {
		t.Errorf("stenalineFormatDateTime with dayOffset=0: got %q, want %q", got, want)
	}
}

func TestStenalineFormatDateTime_InvalidDate(t *testing.T) {
	got := stenalineFormatDateTime("not-a-date", "18:00", 0)
	// Falls back to concatenation.
	if got != "not-a-dateT18:00:00" {
		t.Errorf("stenalineFormatDateTime invalid date fallback: got %q", got)
	}
}

// ============================================================
// SearchStenaLine additional edge cases (was 34.6%)
// ============================================================

func TestSearchStenaLine_MockCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := SearchStenaLine(ctx, "Gothenburg", "Kiel", "2026-07-01", "EUR")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// ============================================================
// tallinkNormalizeDateTime additional cases
// ============================================================

func TestTallinkNormalizeDateTime_FullISO(t *testing.T) {
	// Already has seconds — should return as-is.
	got := tallinkNormalizeDateTime("2026-07-01T08:00:00")
	if got != "2026-07-01T08:00:00" {
		t.Errorf("got %q, want 2026-07-01T08:00:00", got)
	}
}

func TestTallinkNormalizeDateTime_Empty(t *testing.T) {
	got := tallinkNormalizeDateTime("")
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

// ============================================================
// buildRenfeBookingURL
// ============================================================

func TestBuildRenfeBookingURL_Format(t *testing.T) {
	from := RenfeStation{Numeric: "60000", City: "Madrid"}
	to := RenfeStation{Numeric: "71801", City: "Barcelona"}
	url := buildRenfeBookingURL(from, to, "2026-07-01")
	if !strings.Contains(url, "60000") || !strings.Contains(url, "71801") {
		t.Errorf("URL should contain station IDs: %s", url)
	}
	if !strings.Contains(url, "venta.renfe.com") {
		t.Errorf("URL should point to venta.renfe.com: %s", url)
	}
}

// ============================================================
// buildStenaLineBookingURL
// ============================================================

func TestBuildStenaLineBookingURL_Lowercase(t *testing.T) {
	url := buildStenaLineBookingURL("GOT", "KIE")
	if !strings.Contains(url, "got-kie") {
		t.Errorf("URL should contain lowercase route: %s", url)
	}
	if !strings.Contains(url, "stenaline.com") {
		t.Errorf("URL should point to stenaline.com: %s", url)
	}
}

// ============================================================
// buildTallinkBookingURL
// ============================================================

func TestBuildTallinkBookingURL_Format(t *testing.T) {
	url := buildTallinkBookingURL("HEL", "TAL", "2026-07-01")
	if !strings.Contains(url, "from=hel") || !strings.Contains(url, "to=tal") {
		t.Errorf("URL should contain lowercase port codes: %s", url)
	}
	if !strings.Contains(url, "2026-07-01") {
		t.Errorf("URL should contain date: %s", url)
	}
}
