package route

import (
	"testing"

	"github.com/MikkoParkkola/trvl/internal/models"
)

// makeItineraries generates n RouteItinerary values with varying price/duration
// so that the Pareto filter and sort have real work to do.
func makeItineraries(n int) []models.RouteItinerary {
	its := make([]models.RouteItinerary, n)
	for i := 0; i < n; i++ {
		its[i] = models.RouteItinerary{
			TotalPrice:    float64(10 + i*7),
			TotalDuration: 600 - i*5,
			Transfers:     i % 3,
			Currency:      "EUR",
		}
	}
	return its
}

func BenchmarkParetoFilter(b *testing.B) {
	its := makeItineraries(20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Copy so each iteration works on a fresh slice.
		input := make([]models.RouteItinerary, len(its))
		copy(input, its)
		_ = paretoFilter(input)
	}
}

func BenchmarkCandidateHubs(b *testing.B) {
	hel, _ := LookupHub("Helsinki")
	dbv, _ := LookupHub("Dubrovnik")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CandidateHubs(hel, dbv, 2.0)
	}
}

func BenchmarkHaversineKm(b *testing.B) {
	// Helsinki → Dubrovnik
	const lat1, lon1 = 60.1699, 24.9384
	const lat2, lon2 = 42.6507, 18.0944
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = haversineKm(lat1, lon1, lat2, lon2)
	}
}
