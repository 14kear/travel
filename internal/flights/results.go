package flights

import (
	"sort"
	"strings"
	"time"

	"github.com/MikkoParkkola/trvl/internal/batchexec"
	"github.com/MikkoParkkola/trvl/internal/models"
)

const flightTimeLayout = "2006-01-02T15:04"

func mergeFlightResults(opts SearchOptions, providers ...[]models.FlightResult) []models.FlightResult {
	total := 0
	for _, providerFlights := range providers {
		total += len(providerFlights)
	}

	merged := make([]models.FlightResult, 0, total)
	for _, providerFlights := range providers {
		merged = append(merged, providerFlights...)
	}
	merged = filterFlightResults(merged, opts)
	sortFlightResults(merged, opts.SortBy)
	return merged
}

func filterFlightResults(flights []models.FlightResult, opts SearchOptions) []models.FlightResult {
	if len(flights) == 0 {
		return nil
	}

	filtered := make([]models.FlightResult, 0, len(flights))
	for _, f := range flights {
		if opts.MaxPrice > 0 && f.Price > float64(opts.MaxPrice) {
			continue
		}
		if opts.MaxDuration > 0 && f.Duration > opts.MaxDuration {
			continue
		}
		if opts.MaxStops == models.NonStop && f.Stops > 0 {
			continue
		}
		if opts.MaxStops == models.OneStop && f.Stops > 1 {
			continue
		}
		if !flightDepartsWithinWindow(f, opts.DepartAfter, opts.DepartBefore) {
			continue
		}
		filtered = append(filtered, f)
	}

	if len(opts.Airlines) > 0 {
		filtered = filterFlightsByAirline(filtered, opts.Airlines)
	}
	if opts.RequireCheckedBag {
		filtered = filterFlightsWithCheckedBag(filtered)
	}
	if len(opts.Alliances) > 0 {
		filtered = filterFlightsByAlliance(filtered, opts.Alliances)
	}

	return filtered
}

func filterFlightsByAirline(flights []models.FlightResult, airlines []string) []models.FlightResult {
	if len(flights) == 0 {
		return nil
	}

	want := make(map[string]bool, len(airlines))
	for _, airline := range airlines {
		code := strings.TrimSpace(strings.ToUpper(airline))
		if code != "" {
			want[code] = true
		}
	}
	if len(want) == 0 {
		return flights
	}

	filtered := flights[:0]
	for _, f := range flights {
		matched := false
		for _, leg := range f.Legs {
			if want[strings.ToUpper(leg.AirlineCode)] {
				matched = true
				break
			}
		}
		if matched {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

func flightDepartsWithinWindow(f models.FlightResult, after, before string) bool {
	if after == "" && before == "" {
		return true
	}
	if len(f.Legs) == 0 || len(f.Legs[0].DepartureTime) < len("2006-01-02T15:04") {
		return false
	}

	clock := f.Legs[0].DepartureTime[len("2006-01-02T"):]
	if after != "" && clock < after {
		return false
	}
	if before != "" && clock > before {
		return false
	}
	return true
}

func sortFlightResults(flights []models.FlightResult, sortBy models.SortBy) {
	sort.SliceStable(flights, func(i, j int) bool {
		left := flights[i]
		right := flights[j]

		switch sortBy {
		case models.SortDuration:
			if left.Duration != right.Duration {
				return left.Duration < right.Duration
			}
		case models.SortDepartureTime:
			if cmp := compareFlightTimes(flightDeparture(left), flightDeparture(right)); cmp != 0 {
				return cmp < 0
			}
		case models.SortArrivalTime:
			if cmp := compareFlightTimes(flightArrival(left), flightArrival(right)); cmp != 0 {
				return cmp < 0
			}
		default:
			if cmp := compareFlightPrices(left.Price, right.Price); cmp != 0 {
				return cmp < 0
			}
		}

		if cmp := compareFlightPrices(left.Price, right.Price); cmp != 0 {
			return cmp < 0
		}
		if left.Duration != right.Duration {
			return left.Duration < right.Duration
		}
		if cmp := compareFlightTimes(flightDeparture(left), flightDeparture(right)); cmp != 0 {
			return cmp < 0
		}
		if routeCmp := strings.Compare(flightSortKey(left), flightSortKey(right)); routeCmp != 0 {
			return routeCmp < 0
		}
		return strings.Compare(left.Provider, right.Provider) < 0
	})
}

func compareFlightPrices(left, right float64) int {
	switch {
	case left <= 0 && right <= 0:
		return 0
	case left <= 0:
		return 1
	case right <= 0:
		return -1
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func compareFlightTimes(left, right time.Time) int {
	switch {
	case left.IsZero() && right.IsZero():
		return 0
	case left.IsZero():
		return 1
	case right.IsZero():
		return -1
	case left.Before(right):
		return -1
	case left.After(right):
		return 1
	default:
		return 0
	}
}

func parseFlexibleFlightTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		flightTimeLayout,
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func flightDurationFromLegs(legs []models.FlightLeg) int {
	if len(legs) == 0 {
		return 0
	}

	start, startOK := parseFlexibleFlightTime(legs[0].DepartureTime)
	end, endOK := parseFlexibleFlightTime(legs[len(legs)-1].ArrivalTime)
	if startOK && endOK && end.After(start) {
		return int(end.Sub(start).Minutes())
	}

	total := 0
	for _, leg := range legs {
		if leg.Duration > 0 {
			total += leg.Duration
		}
		if leg.LayoverMinutes > 0 {
			total += leg.LayoverMinutes
		}
	}
	return total
}

func flightDeparture(f models.FlightResult) time.Time {
	if len(f.Legs) == 0 {
		return time.Time{}
	}
	t, _ := time.Parse(flightTimeLayout, f.Legs[0].DepartureTime)
	return t
}

func flightArrival(f models.FlightResult) time.Time {
	if len(f.Legs) == 0 {
		return time.Time{}
	}
	t, _ := time.Parse(flightTimeLayout, f.Legs[len(f.Legs)-1].ArrivalTime)
	return t
}

func flightSortKey(f models.FlightResult) string {
	if len(f.Legs) == 0 {
		return ""
	}

	parts := []string{f.Legs[0].DepartureAirport.Code}
	for _, leg := range f.Legs {
		parts = append(parts, leg.ArrivalAirport.Code)
	}
	return strings.Join(parts, "->")
}

func flightSearchCurrency(result *models.FlightSearchResult) string {
	if result != nil {
		for _, f := range result.Flights {
			if f.Currency != "" {
				return f.Currency
			}
		}
	}
	return "EUR"
}

func tripTypeForSearch(opts SearchOptions) string {
	if opts.ReturnDate != "" {
		return "round_trip"
	}
	return "one_way"
}

func kiwiSearchEligible(client *batchexec.Client, opts SearchOptions) bool {
	if client == nil || client != batchexec.SharedClient() {
		return false
	}
	return kiwiEligibleOptions(opts)
}

func kiwiEligibleOptions(opts SearchOptions) bool {
	if opts.ReturnDate != "" {
		return false
	}
	if len(opts.Airlines) > 0 || len(opts.Alliances) > 0 {
		return false
	}
	if opts.CarryOnBags > 0 || opts.CheckedBags > 0 {
		return false
	}
	if opts.RequireCheckedBag || opts.ExcludeBasic || opts.LessEmissions {
		return false
	}
	return true
}
