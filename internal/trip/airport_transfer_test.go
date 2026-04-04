package trip

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/MikkoParkkola/trvl/internal/destinations"
	"github.com/MikkoParkkola/trvl/internal/ground"
	"github.com/MikkoParkkola/trvl/internal/models"
)

func TestSearchAirportTransfers_UnknownAirportCode(t *testing.T) {
	_, err := searchAirportTransfers(context.Background(), AirportTransferInput{
		AirportCode: "ZZZ",
		Destination: "Hotel Lutetia Paris",
		Date:        "2026-07-01",
	}, airportTransferDeps{})
	if err == nil || !strings.Contains(err.Error(), "unknown airport_code") {
		t.Fatalf("expected unknown airport_code error, got %v", err)
	}
}

func TestSearchAirportTransfers_CombinesExactAndCityMatches(t *testing.T) {
	var (
		mu            sync.Mutex
		groundCalls   = make(map[string][]string)
		geocodeCalls  []string
		transitousHit bool
	)
	deps := airportTransferDeps{
		geocode: func(_ context.Context, query string) (destinations.GeoResult, error) {
			mu.Lock()
			geocodeCalls = append(geocodeCalls, query)
			mu.Unlock()
			return destinations.GeoResult{Locality: "Paris"}, nil
		},
		searchTransitous: func(_ context.Context, fromLat, fromLon, toLat, toLon float64, date string) ([]models.GroundRoute, error) {
			transitousHit = true
			return []models.GroundRoute{{
				Provider:  "transitous",
				Type:      "mixed",
				Price:     0,
				Currency:  "EUR",
				Duration:  42,
				Transfers: 0,
				Departure: models.GroundStop{Time: "2026-07-01T13:15:00+02:00"},
				Arrival:   models.GroundStop{Time: "2026-07-01T13:57:00+02:00"},
			}}, nil
		},
		searchGround: func(_ context.Context, from, to, date string, opts ground.SearchOptions) (*models.GroundSearchResult, error) {
			mu.Lock()
			groundCalls[from+"->"+to] = append([]string(nil), opts.Providers...)
			mu.Unlock()
			switch from + "->" + to {
			case "Paris->Paris":
				return &models.GroundSearchResult{
					Success: true,
					Count:   1,
					Routes: []models.GroundRoute{{
						Provider:  "db",
						Type:      "train",
						Price:     18,
						Currency:  "EUR",
						Duration:  55,
						Transfers: 1,
						Departure: models.GroundStop{Time: "2026-07-01T14:00:00+02:00"},
						Arrival:   models.GroundStop{Time: "2026-07-01T14:55:00+02:00"},
					}},
				}, nil
			default:
				t.Fatalf("unexpected search %s -> %s (%s)", from, to, date)
				return nil, nil
			}
		},
	}

	result, err := searchAirportTransfers(context.Background(), AirportTransferInput{
		AirportCode: "CDG",
		Destination: "Hotel Lutetia Paris",
		Date:        "2026-07-01",
	}, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error %q", result.Error)
	}
	if result.ExactMatches != 1 || result.CityMatches != 1 {
		t.Fatalf("exact/city matches = %d/%d, want 1/1", result.ExactMatches, result.CityMatches)
	}
	if len(result.Routes) != 2 {
		t.Fatalf("route count = %d, want 2", len(result.Routes))
	}
	if result.Routes[0].Provider != "transitous" {
		t.Fatalf("expected exact airport match first, got %q", result.Routes[0].Provider)
	}
	if !transitousHit {
		t.Fatal("expected exact Transitous search to run")
	}

	mu.Lock()
	defer mu.Unlock()
	if got := groundCalls["Paris->Paris"]; !reflect.DeepEqual(got, airportTransferCityProviders) {
		t.Fatalf("city providers = %v, want %v", got, airportTransferCityProviders)
	}
	if !reflect.DeepEqual(geocodeCalls, []string{"Hotel Lutetia Paris", "Paris CDG airport"}) {
		t.Fatalf("geocode calls = %v, want destination then airport query", geocodeCalls)
	}
}

func TestSearchAirportTransfers_FiltersByArrivalTime(t *testing.T) {
	deps := airportTransferDeps{
		geocode: func(context.Context, string) (destinations.GeoResult, error) {
			return destinations.GeoResult{Locality: "Paris"}, nil
		},
		searchTransitous: func(_ context.Context, fromLat, fromLon, toLat, toLon float64, date string) ([]models.GroundRoute, error) {
			return []models.GroundRoute{{
				Provider:  "transitous",
				Type:      "mixed",
				Price:     0,
				Currency:  "EUR",
				Duration:  42,
				Transfers: 0,
				Departure: models.GroundStop{Time: "2026-07-01T13:15:00+02:00"},
				Arrival:   models.GroundStop{Time: "2026-07-01T13:57:00+02:00"},
			}}, nil
		},
		searchGround: func(_ context.Context, from, to, _ string, opts ground.SearchOptions) (*models.GroundSearchResult, error) {
			switch from + "->" + to {
			case "Paris->Paris":
				if !reflect.DeepEqual(opts.Providers, airportTransferCityProviders) {
					t.Fatalf("unexpected providers: %v", opts.Providers)
				}
				return &models.GroundSearchResult{
					Success: true,
					Count:   1,
					Routes: []models.GroundRoute{{
						Provider:  "db",
						Type:      "train",
						Price:     18,
						Currency:  "EUR",
						Duration:  55,
						Transfers: 1,
						Departure: models.GroundStop{Time: "2026-07-01T15:00:00+02:00"},
						Arrival:   models.GroundStop{Time: "2026-07-01T15:55:00+02:00"},
					}},
				}, nil
			default:
				t.Fatalf("unexpected search %s -> %s", from, to)
				return nil, nil
			}
		},
	}

	result, err := searchAirportTransfers(context.Background(), AirportTransferInput{
		AirportCode: "CDG",
		Destination: "Hotel Lutetia Paris",
		Date:        "2026-07-01",
		ArrivalTime: "14:00",
	}, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Routes) != 1 {
		t.Fatalf("route count = %d, want 1", len(result.Routes))
	}
	if result.Routes[0].Provider != "db" {
		t.Fatalf("remaining provider = %q, want db", result.Routes[0].Provider)
	}
	if result.ExactMatches != 0 || result.CityMatches != 1 {
		t.Fatalf("exact/city matches = %d/%d, want 0/1", result.ExactMatches, result.CityMatches)
	}
}

func TestSearchAirportTransfers_TransitousOnlyProvider(t *testing.T) {
	var (
		groundCalls     int
		transitousCalls int
	)
	deps := airportTransferDeps{
		geocode: func(_ context.Context, query string) (destinations.GeoResult, error) {
			if query == "Hotel Lutetia Paris" || query == "Paris CDG airport" {
				return destinations.GeoResult{Locality: "Paris"}, nil
			}
			t.Fatalf("unexpected geocode query %q", query)
			return destinations.GeoResult{}, nil
		},
		searchTransitous: func(_ context.Context, fromLat, fromLon, toLat, toLon float64, date string) ([]models.GroundRoute, error) {
			transitousCalls++
			return nil, nil
		},
		searchGround: func(_ context.Context, from, to, _ string, opts ground.SearchOptions) (*models.GroundSearchResult, error) {
			groundCalls++
			t.Fatalf("unexpected city search %s -> %s with %v", from, to, opts.Providers)
			return nil, nil
		},
	}

	result, err := searchAirportTransfers(context.Background(), AirportTransferInput{
		AirportCode: "CDG",
		Destination: "Hotel Lutetia Paris",
		Date:        "2026-07-01",
		Providers:   []string{"transitous"},
	}, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transitousCalls != 1 {
		t.Fatalf("transitous calls = %d, want 1", transitousCalls)
	}
	if groundCalls != 0 {
		t.Fatalf("ground calls = %d, want 0", groundCalls)
	}
	if result.Success {
		t.Fatalf("expected no success when stub returns no routes")
	}
}

func TestGeocodeAirportTransferDestination_BiasesAmbiguousDestinations(t *testing.T) {
	var calls []string
	geo, err := geocodeAirportTransferDestination(context.Background(), func(_ context.Context, query string) (destinations.GeoResult, error) {
		calls = append(calls, query)
		if query == "Paddington Station, London" {
			return destinations.GeoResult{Locality: "London"}, nil
		}
		return destinations.GeoResult{}, context.DeadlineExceeded
	}, "Paddington Station", "London")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if geo.Locality != "London" {
		t.Fatalf("locality = %q, want London", geo.Locality)
	}
	if !reflect.DeepEqual(calls, []string{"Paddington Station, London"}) {
		t.Fatalf("geocode calls = %v, want biased query only", calls)
	}
}
