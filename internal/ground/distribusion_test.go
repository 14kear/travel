package ground

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// ---- HasDistribusionKey ----

func TestHasDistribusionKey_Missing(t *testing.T) {
	t.Setenv("DISTRIBUSION_API_KEY", "")
	if HasDistribusionKey() {
		t.Error("HasDistribusionKey() = true when env var is empty, want false")
	}
}

func TestHasDistribusionKey_Set(t *testing.T) {
	t.Setenv("DISTRIBUSION_API_KEY", "test-key-abc123")
	if !HasDistribusionKey() {
		t.Error("HasDistribusionKey() = false when env var is set, want true")
	}
}

// ---- distribusionStationCode ----

func TestDistribusionStationCode_KnownCities(t *testing.T) {
	tests := []struct {
		city string
		want string
	}{
		{"Helsinki", "FIHELS"},
		{"helsinki", "FIHELS"},
		{"  Helsinki  ", "FIHELS"},
		{"hel", "FIHELS"},
		{"Tallinn", "EETLLS"},
		{"tallinn", "EETLLS"},
		{"tll", "EETLLS"},
		{"Stockholm", "SESTON"},
		{"Gothenburg", "SEGOTS"},
		{"Göteborg", "SEGOTS"},
		{"goteborg", "SEGOTS"},
		{"Berlin", "DEBERZ"},
		{"Paris", "FRPARS"},
		{"London", "GBLONS"},
		{"Amsterdam", "NLAMSZ"},
		{"Copenhagen", "DKCPHS"},
		{"Riga", "LVRIXS"},
		{"Vienna", "ATVIES"},
		{"Warsaw", "PLWAWS"},
		{"Prague", "CZPRGS"},
		{"Rome", "ITROMS"},
		{"Madrid", "ESMADS"},
		{"Zurich", "CHZRHS"},
	}

	for _, tt := range tests {
		t.Run(tt.city, func(t *testing.T) {
			got := distribusionStationCode(tt.city)
			if got != tt.want {
				t.Errorf("distribusionStationCode(%q) = %q, want %q", tt.city, got, tt.want)
			}
		})
	}
}

func TestDistribusionStationCode_UnknownCities(t *testing.T) {
	unknowns := []string{"", "Atlantis", "Xyzzy", "Middle-earth"}
	for _, city := range unknowns {
		t.Run(city, func(t *testing.T) {
			got := distribusionStationCode(city)
			if got != "" {
				t.Errorf("distribusionStationCode(%q) = %q, want empty string", city, got)
			}
		})
	}
}

func TestDistribusionStationCode_AllCodesNonEmpty(t *testing.T) {
	for alias, code := range distribusionStationCodes {
		if code == "" {
			t.Errorf("station code for alias %q is empty", alias)
		}
		if len(code) < 5 {
			t.Errorf("station code for alias %q (%q) is shorter than 5 chars", alias, code)
		}
	}
}

// ---- normaliseDistribusionType ----

func TestNormaliseDistribusionType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"train", "train"},
		{"TRAIN", "train"},
		{"rail", "train"},
		{"ferry", "ferry"},
		{"FERRY", "ferry"},
		{"sea", "ferry"},
		{"bus", "bus"},
		{"BUS", "bus"},
		{"coach", "bus"},
		{"", "bus"},
		{"unknown", "bus"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normaliseDistribusionType(tt.input)
			if got != tt.want {
				t.Errorf("normaliseDistribusionType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---- JSONAPI response parsing ----

func TestDistribusionJSONAPIParsing_BusRoute(t *testing.T) {
	const rawJSON = `{
		"data": [
			{
				"id": "conn-001",
				"type": "connection",
				"attributes": {
					"departure_time": "2026-05-10T08:00:00",
					"arrival_time": "2026-05-10T14:00:00",
					"duration_in_minutes": 360,
					"lowest_price": 1999,
					"currency": "EUR",
					"traffic_type": "bus",
					"marketing_carrier_code": "FLIXBUS",
					"departure_station_code": "FIHELS",
					"arrival_station_code": "EETLLS",
					"available": true
				}
			}
		],
		"included": [
			{
				"id": "FIHELS",
				"type": "station",
				"attributes": {
					"name": "Helsinki Bus Terminal",
					"city": "Helsinki"
				}
			},
			{
				"id": "EETLLS",
				"type": "station",
				"attributes": {
					"name": "Tallinn Bus Terminal",
					"city": "Tallinn"
				}
			},
			{
				"id": "FLIXBUS",
				"type": "marketing_carrier",
				"attributes": {
					"trade_name_en": "FlixBus"
				}
			}
		],
		"jsonapi": {"version": "1.0"},
		"meta": {
			"departure_station_code": "FIHELS",
			"arrival_station_code": "EETLLS"
		}
	}`

	var envelope distribusionJSONAPI
	if err := json.Unmarshal([]byte(rawJSON), &envelope); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if len(envelope.Data) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(envelope.Data))
	}
	conn := envelope.Data[0]
	if conn.ID != "conn-001" {
		t.Errorf("ID = %q, want conn-001", conn.ID)
	}
	if conn.Attributes.LowestPrice != 1999 {
		t.Errorf("LowestPrice = %d, want 1999", conn.Attributes.LowestPrice)
	}
	if conn.Attributes.DurationInMinutes != 360 {
		t.Errorf("Duration = %d, want 360", conn.Attributes.DurationInMinutes)
	}
	if conn.Attributes.TrafficType != "bus" {
		t.Errorf("TrafficType = %q, want bus", conn.Attributes.TrafficType)
	}
	if !conn.Attributes.Available {
		t.Error("Available should be true")
	}

	if len(envelope.Included) != 3 {
		t.Fatalf("expected 3 included resources, got %d", len(envelope.Included))
	}
}

func TestDistribusionJSONAPIParsing_TrainRoute(t *testing.T) {
	const rawJSON = `{
		"data": [
			{
				"id": "conn-train-001",
				"type": "connection",
				"attributes": {
					"departure_time": "2026-05-10T09:00:00",
					"arrival_time": "2026-05-10T11:30:00",
					"duration_in_minutes": 150,
					"lowest_price": 3700,
					"currency": "EUR",
					"traffic_type": "train",
					"marketing_carrier_code": "VR",
					"departure_station_code": "FIHELS",
					"arrival_station_code": "FITMPX",
					"available": true
				}
			}
		],
		"included": [],
		"jsonapi": {"version": "1.0"},
		"meta": {}
	}`

	var envelope distribusionJSONAPI
	if err := json.Unmarshal([]byte(rawJSON), &envelope); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	conn := envelope.Data[0]
	if conn.Attributes.TrafficType != "train" {
		t.Errorf("TrafficType = %q, want train", conn.Attributes.TrafficType)
	}
	if conn.Attributes.LowestPrice != 3700 {
		t.Errorf("LowestPrice = %d, want 3700", conn.Attributes.LowestPrice)
	}
}

func TestDistribusionJSONAPIParsing_UnavailableFiltered(t *testing.T) {
	const rawJSON = `{
		"data": [
			{
				"id": "conn-unavail",
				"type": "connection",
				"attributes": {
					"departure_time": "2026-05-10T06:00:00",
					"arrival_time": "2026-05-10T12:00:00",
					"duration_in_minutes": 360,
					"lowest_price": 2500,
					"currency": "EUR",
					"traffic_type": "bus",
					"marketing_carrier_code": "FLIX",
					"departure_station_code": "FIHELS",
					"arrival_station_code": "EETLLS",
					"available": false
				}
			}
		],
		"included": [],
		"jsonapi": {"version": "1.0"},
		"meta": {}
	}`

	var envelope distribusionJSONAPI
	if err := json.Unmarshal([]byte(rawJSON), &envelope); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if len(envelope.Data) != 1 {
		t.Fatalf("expected 1 data element, got %d", len(envelope.Data))
	}
	if envelope.Data[0].Attributes.Available {
		t.Error("Available should be false for this test fixture")
	}
}

func TestDistribusionJSONAPIParsing_EmptyResponse(t *testing.T) {
	const rawJSON = `{
		"data": [],
		"included": [],
		"jsonapi": {"version": "1.0"},
		"meta": {}
	}`

	var envelope distribusionJSONAPI
	if err := json.Unmarshal([]byte(rawJSON), &envelope); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if len(envelope.Data) != 0 {
		t.Errorf("expected 0 data elements, got %d", len(envelope.Data))
	}
}

// ---- Mock HTTP server tests ----

// mockDistribusionServer returns a test HTTP server that responds to
// /connections/find with the given JSONAPI payload.
func mockDistribusionServer(t *testing.T, payload string, statusCode int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header.
		if r.Header.Get("Api-Key") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(payload))
	}))
}

func TestSearchDistribusion_NoAPIKey(t *testing.T) {
	t.Setenv("DISTRIBUSION_API_KEY", "")

	ctx := context.Background()
	routes, err := SearchDistribusion(ctx, "Helsinki", "Tallinn", "2026-05-10", "EUR")
	if err != nil {
		t.Errorf("expected nil error when no API key, got: %v", err)
	}
	if routes != nil {
		t.Errorf("expected nil routes when no API key, got %d routes", len(routes))
	}
}

func TestSearchDistribusion_UnknownOrigin(t *testing.T) {
	t.Setenv("DISTRIBUSION_API_KEY", "test-key")

	ctx := context.Background()
	routes, err := SearchDistribusion(ctx, "Atlantis", "Helsinki", "2026-05-10", "EUR")
	if err != nil {
		t.Errorf("expected nil error for unknown origin, got: %v", err)
	}
	if routes != nil {
		t.Errorf("expected nil routes for unknown origin, got %d", len(routes))
	}
}

func TestSearchDistribusion_UnknownDestination(t *testing.T) {
	t.Setenv("DISTRIBUSION_API_KEY", "test-key")

	ctx := context.Background()
	routes, err := SearchDistribusion(ctx, "Helsinki", "Atlantis", "2026-05-10", "EUR")
	if err != nil {
		t.Errorf("expected nil error for unknown destination, got: %v", err)
	}
	if routes != nil {
		t.Errorf("expected nil routes for unknown destination, got %d", len(routes))
	}
}

func TestSearchDistribusion_MockServer_Success(t *testing.T) {
	seatsAvail := 42
	payload, _ := json.Marshal(distribusionJSONAPI{
		Data: []distribusionData{
			{
				ID:   "conn-mock-001",
				Type: "connection",
				Attributes: distribusionConnectionAttrs{
					DepartureTime:        "2026-05-10T08:00:00",
					ArrivalTime:          "2026-05-10T10:30:00",
					DurationInMinutes:    150,
					LowestPrice:          2200,
					Currency:             "EUR",
					TrafficType:          "bus",
					MarketingCarrierCode: "FLIX",
					DepartureStationCode: "FIHELS",
					ArrivalStationCode:   "EETLLS",
					Available:            true,
					SeatsAvailable:       &seatsAvail,
				},
			},
		},
		Included: []distribusionIncluded{
			{
				ID:   "FIHELS",
				Type: "station",
				Attributes: distribusionIncludedAttrs{
					Name: "Helsinki Kamppi",
					City: "Helsinki",
				},
			},
			{
				ID:   "EETLLS",
				Type: "station",
				Attributes: distribusionIncludedAttrs{
					Name: "Tallinn Bus Station",
					City: "Tallinn",
				},
			},
			{
				ID:   "FLIX",
				Type: "marketing_carrier",
				Attributes: distribusionIncludedAttrs{
					TradeNameEn: "FlixBus",
				},
			},
		},
	})

	srv := mockDistribusionServer(t, string(payload), http.StatusOK)
	defer srv.Close()

	// Point the client at the mock server by temporarily replacing the base URL.
	origBase := distribusionAPIBase
	// We can't reassign the const, so we test via the full flow using the
	// real SearchDistribusion with a custom HTTP transport that redirects to mock.
	_ = origBase

	// Instead, test the parsing logic directly (the HTTP layer is tested via
	// TestSearchDistribusion_HTTP below which uses a real server + env override).
	t.Run("parse_response", func(t *testing.T) {
		var envelope distribusionJSONAPI
		if err := json.Unmarshal(payload, &envelope); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		stationNames := make(map[string]string)
		carrierNames := make(map[string]string)
		for _, inc := range envelope.Included {
			switch inc.Type {
			case "station":
				city := inc.Attributes.City
				if city == "" {
					city = inc.Attributes.Name
				}
				stationNames[inc.ID] = city
			case "marketing_carrier":
				name := inc.Attributes.TradeNameEn
				if name == "" {
					name = inc.ID
				}
				carrierNames[inc.ID] = name
			}
		}

		if stationNames["FIHELS"] != "Helsinki" {
			t.Errorf("stationNames[FIHELS] = %q, want Helsinki", stationNames["FIHELS"])
		}
		if stationNames["EETLLS"] != "Tallinn" {
			t.Errorf("stationNames[EETLLS] = %q, want Tallinn", stationNames["EETLLS"])
		}
		if carrierNames["FLIX"] != "FlixBus" {
			t.Errorf("carrierNames[FLIX] = %q, want FlixBus", carrierNames["FLIX"])
		}

		conn := envelope.Data[0]
		priceFloat := float64(conn.Attributes.LowestPrice) / 100.0
		if priceFloat != 22.0 {
			t.Errorf("price = %.2f, want 22.00", priceFloat)
		}
		if conn.Attributes.DurationInMinutes != 150 {
			t.Errorf("duration = %d, want 150", conn.Attributes.DurationInMinutes)
		}
		if conn.Attributes.SeatsAvailable == nil || *conn.Attributes.SeatsAvailable != 42 {
			t.Errorf("seats available = %v, want 42", conn.Attributes.SeatsAvailable)
		}
	})
}

func TestSearchDistribusion_MockServer_HTTP(t *testing.T) {
	// This test verifies the full HTTP flow against a mock server.
	// It swaps the distribusionHTTPClient transport so requests go to the mock.
	// Reset the rate limiter to avoid waiting in tests.
	origLimiter := distribusionLimiter
	distribusionLimiter = rate.NewLimiter(rate.Inf, 1)
	defer func() { distribusionLimiter = origLimiter }()
	seatsAvail := 10
	payload, _ := json.Marshal(distribusionJSONAPI{
		Data: []distribusionData{
			{
				ID:   "conn-http-001",
				Type: "connection",
				Attributes: distribusionConnectionAttrs{
					DepartureTime:        "2026-06-01T07:00:00",
					ArrivalTime:          "2026-06-01T09:00:00",
					DurationInMinutes:    120,
					LowestPrice:          1500,
					Currency:             "EUR",
					TrafficType:          "bus",
					MarketingCarrierCode: "LUXEX",
					DepartureStationCode: "FIHELS",
					ArrivalStationCode:   "EETLLS",
					Available:            true,
					SeatsAvailable:       &seatsAvail,
				},
			},
		},
		Included: []distribusionIncluded{
			{
				ID:   "FIHELS",
				Type: "station",
				Attributes: distribusionIncludedAttrs{City: "Helsinki"},
			},
			{
				ID:   "EETLLS",
				Type: "station",
				Attributes: distribusionIncludedAttrs{City: "Tallinn"},
			},
			{
				ID:   "LUXEX",
				Type: "marketing_carrier",
				Attributes: distribusionIncludedAttrs{TradeNameEn: "Lux Express"},
			},
		},
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Api-Key") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Verify query parameters are present.
		q := r.URL.Query()
		if q.Get("departure_stations") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if q.Get("arrival_stations") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if q.Get("departure_date") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	t.Setenv("DISTRIBUSION_API_KEY", "mock-api-key")

	// Swap the HTTP client transport to redirect to the mock server.
	origClient := distribusionHTTPClient
	distribusionHTTPClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &mockDistribusionTransport{
			real:     srv.URL,
			original: distribusionAPIBase,
		},
	}
	defer func() { distribusionHTTPClient = origClient }()

	ctx := context.Background()
	routes, err := SearchDistribusion(ctx, "Helsinki", "Tallinn", "2026-06-01", "EUR")
	if err != nil {
		t.Fatalf("SearchDistribusion failed: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("expected at least one route")
	}

	r := routes[0]
	if r.Provider != "distribusion" {
		t.Errorf("Provider = %q, want distribusion", r.Provider)
	}
	if r.Type != "bus" {
		t.Errorf("Type = %q, want bus", r.Type)
	}
	if r.Price != 15.0 {
		t.Errorf("Price = %.2f, want 15.00", r.Price)
	}
	if r.Currency != "EUR" {
		t.Errorf("Currency = %q, want EUR", r.Currency)
	}
	if r.Duration != 120 {
		t.Errorf("Duration = %d, want 120", r.Duration)
	}
	if r.Departure.City != "Helsinki" {
		t.Errorf("Departure.City = %q, want Helsinki", r.Departure.City)
	}
	if r.Arrival.City != "Tallinn" {
		t.Errorf("Arrival.City = %q, want Tallinn", r.Arrival.City)
	}
	if r.Departure.Time != "2026-06-01T07:00:00" {
		t.Errorf("Departure.Time = %q, want 2026-06-01T07:00:00", r.Departure.Time)
	}
	if r.BookingURL == "" {
		t.Error("BookingURL should not be empty")
	}
	if r.SeatsLeft == nil || *r.SeatsLeft != 10 {
		t.Errorf("SeatsLeft = %v, want 10", r.SeatsLeft)
	}
	if len(r.Amenities) == 0 || r.Amenities[0] != "Lux Express" {
		t.Errorf("Amenities = %v, want [Lux Express]", r.Amenities)
	}
}

func TestSearchDistribusion_MockServer_HTTP401(t *testing.T) {
	origLimiter := distribusionLimiter
	distribusionLimiter = rate.NewLimiter(rate.Inf, 1)
	defer func() { distribusionLimiter = origLimiter }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	t.Setenv("DISTRIBUSION_API_KEY", "bad-key")

	origClient := distribusionHTTPClient
	distribusionHTTPClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &mockDistribusionTransport{
			real:     srv.URL,
			original: distribusionAPIBase,
		},
	}
	defer func() { distribusionHTTPClient = origClient }()

	ctx := context.Background()
	_, err := SearchDistribusion(ctx, "Helsinki", "Tallinn", "2026-06-01", "EUR")
	if err == nil {
		t.Error("expected error for HTTP 401, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention 401, got: %v", err)
	}
}

func TestSearchDistribusion_MockServer_UnavailableSkipped(t *testing.T) {
	origLimiter := distribusionLimiter
	distribusionLimiter = rate.NewLimiter(rate.Inf, 1)
	defer func() { distribusionLimiter = origLimiter }()

	payload, _ := json.Marshal(distribusionJSONAPI{
		Data: []distribusionData{
			{
				ID:   "conn-unavail",
				Type: "connection",
				Attributes: distribusionConnectionAttrs{
					DepartureTime:     "2026-06-01T07:00:00",
					ArrivalTime:       "2026-06-01T09:00:00",
					DurationInMinutes: 120,
					LowestPrice:       1500,
					Currency:          "EUR",
					TrafficType:       "bus",
					Available:         false, // unavailable
				},
			},
		},
		Included: []distribusionIncluded{},
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	t.Setenv("DISTRIBUSION_API_KEY", "test-key")

	origClient := distribusionHTTPClient
	distribusionHTTPClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &mockDistribusionTransport{
			real:     srv.URL,
			original: distribusionAPIBase,
		},
	}
	defer func() { distribusionHTTPClient = origClient }()

	ctx := context.Background()
	routes, err := SearchDistribusion(ctx, "Helsinki", "Tallinn", "2026-06-01", "EUR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 0 {
		t.Errorf("expected 0 routes (all unavailable), got %d", len(routes))
	}
}

func TestSearchDistribusion_RateLimiterConfiguration(t *testing.T) {
	assertLimiterConfiguration(t, distribusionLimiter, 6*time.Second, 1)
}

// ---- Integration test ----

func TestSearchDistribusion_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if os.Getenv("DISTRIBUSION_API_KEY") == "" {
		t.Skip("DISTRIBUSION_API_KEY not set — skipping live test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	date := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	routes, err := SearchDistribusion(ctx, "Helsinki", "Tallinn", date, "EUR")
	if err != nil {
		t.Skipf("Distribusion unavailable: %v", err)
	}
	if len(routes) == 0 {
		t.Skip("no Distribusion routes found for Helsinki-Tallinn")
	}

	r := routes[0]
	if r.Provider != "distribusion" {
		t.Errorf("Provider = %q, want distribusion", r.Provider)
	}
	if r.Price <= 0 {
		t.Errorf("Price = %.2f, should be > 0", r.Price)
	}
	if r.Currency == "" {
		t.Error("Currency should not be empty")
	}
	if r.Duration <= 0 {
		t.Errorf("Duration = %d, should be > 0", r.Duration)
	}
	if r.Departure.City == "" {
		t.Error("Departure.City should not be empty")
	}
	if r.Arrival.City == "" {
		t.Error("Arrival.City should not be empty")
	}
	if r.BookingURL == "" {
		t.Error("BookingURL should not be empty")
	}
}

// mockDistribusionTransport redirects requests from the real Distribusion API
// base URL to the mock server URL.
type mockDistribusionTransport struct {
	real     string // mock server URL (e.g. "http://127.0.0.1:PORT")
	original string // real API base URL being replaced
}

func (t *mockDistribusionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the host to point at the mock server.
	newURL := *req.URL
	mockBase := strings.TrimRight(t.real, "/")
	origBase := strings.TrimRight(t.original, "/")
	newURL.Scheme = "http"
	// Replace just the host+scheme portion of the URL.
	rawURL := req.URL.String()
	rawURL = strings.Replace(rawURL, origBase, mockBase, 1)
	parsed, err := req.URL.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	_ = newURL
	newReq := req.Clone(req.Context())
	newReq.URL = parsed
	newReq.Host = parsed.Host
	return http.DefaultTransport.RoundTrip(newReq)
}
