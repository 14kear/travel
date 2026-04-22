package hacks

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/MikkoParkkola/trvl/internal/baggage"
	"github.com/MikkoParkkola/trvl/internal/flights"
	"github.com/MikkoParkkola/trvl/internal/models"
)

// hiddenCityExtensions is deprecated compatibility data kept only so older
// tests and coverage loops can compile. Hidden-city routing no longer depends
// on a hardcoded hub->beyond list.
var hiddenCityExtensions = map[string][]string{}

// HiddenCityExtensionsForDestination returns nil because hidden-city candidate
// generation no longer depends on a hardcoded list of beyond airports.
func HiddenCityExtensionsForDestination(string) []string {
	return nil
}

// HiddenCityCandidateAirports returns a deterministic ranked list of plausible
// "beyond" airports for a hub destination. It works from the route itself:
// the requested destination acts as hub B, and candidate C airports are scored
// from the broader airport universe without requiring a hardcoded hub list.
func HiddenCityCandidateAirports(dest, origin string, limit int) []string {
	dest = strings.ToUpper(strings.TrimSpace(dest))
	origin = strings.ToUpper(strings.TrimSpace(origin))
	if dest == "" || origin == "" || dest == origin {
		return nil
	}
	if limit <= 0 {
		limit = 12
	}

	destCountry := airportCountryCode(dest)
	originCountry := airportCountryCode(origin)
	destRegion := airportMacroRegion(dest)
	directDistance := airportDistanceKm(origin, dest)

	type scoredAirport struct {
		code  string
		score int
	}

	var ranked []scoredAirport
	for code := range hiddenCityAirportUniverse() {
		if code == "" || code == dest || code == origin {
			continue
		}

		score := 0
		candCountry := airportCountryCode(code)
		candRegion := airportMacroRegion(code)
		originToCand := airportDistanceKm(origin, code)
		hubToCand := airportDistanceKm(dest, code)

		if hubToCand <= 0 || originToCand <= 0 {
			continue
		}
		if hubToCand < 250 || hubToCand > 6500 {
			continue
		}

		switch {
		case hubToCand >= 300 && hubToCand <= 2500:
			score += 28
		case hubToCand <= 4500:
			score += 18
		default:
			score += 8
		}

		if directDistance > 0 {
			switch {
			case originToCand >= directDistance+250:
				score += 26
			case originToCand >= directDistance+75:
				score += 14
			case originToCand < directDistance:
				score -= 16
			}
		}

		if candCountry != "" && candCountry == destCountry {
			score += 12
		}
		if originCountry != "" && destCountry != "" && originCountry != destCountry &&
			candCountry != "" && candCountry != originCountry {
			score += 8
		}

		if destRegion != "" && candRegion == destRegion {
			score += 18
		} else if regionsAdjacent(destRegion, candRegion) {
			score += 8
		}

		if score >= 28 {
			ranked = append(ranked, scoredAirport{code: code, score: score})
		}
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].code < ranked[j].code
		}
		return ranked[i].score > ranked[j].score
	})

	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	out := make([]string, 0, len(ranked))
	for _, cand := range ranked {
		out = append(out, cand.code)
	}
	return out
}

// detectHiddenCity searches for flights where the ticket to a beyond airport
// is cheaper than the ticket ending at the requested hub destination.
func detectHiddenCity(ctx context.Context, in DetectorInput) []Hack {
	if !in.valid() || in.Date == "" {
		return nil
	}

	beyonds := HiddenCityCandidateAirports(in.Destination, in.Origin, 14)
	if len(beyonds) == 0 {
		return nil
	}

	directResult, err := flights.SearchFlights(ctx, in.Origin, in.Destination, in.Date, flights.SearchOptions{})
	if err != nil || !directResult.Success || len(directResult.Flights) == 0 {
		return nil
	}
	directPrice := minFlightPrice(directResult)
	if directPrice <= 0 {
		return nil
	}
	currency := flightCurrency(directResult, in.currency())

	var best *Hack
	bestSavings := 0.0
	for _, beyond := range beyonds {
		beyondResult, err := flights.SearchFlights(ctx, in.Origin, beyond, in.Date, flights.SearchOptions{})
		if err != nil || !beyondResult.Success || len(beyondResult.Flights) == 0 {
			continue
		}
		beyondPrice := minFlightPrice(beyondResult)
		if beyondPrice <= 0 || beyondPrice >= directPrice {
			continue
		}
		if !routesThroughDestination(beyondResult, in.Destination) {
			continue
		}

		savings := directPrice - beyondPrice
		if savings <= bestSavings {
			continue
		}
		bestSavings = savings
		airlineCode := primaryAirlineCode(beyondResult)
		hack := buildHiddenCityHack(in, beyond, beyondPrice, directPrice, currency, airlineCode)
		best = &hack
	}

	if best == nil {
		return nil
	}
	return []Hack{*best}
}

// FlightRoutesThroughAirport returns true when at least one flight routes
// through the given airport before the final destination.
func FlightRoutesThroughAirport(flts []models.FlightResult, airport string) bool {
	airport = strings.ToUpper(strings.TrimSpace(airport))
	if airport == "" {
		return false
	}
	for _, f := range flts {
		if len(f.Legs) < 2 {
			continue
		}
		for i := 0; i < len(f.Legs)-1; i++ {
			if strings.ToUpper(strings.TrimSpace(f.Legs[i].ArrivalAirport.Code)) == airport {
				return true
			}
		}
	}
	return false
}

// routesThroughDestination returns true when at least one flight in the result
// has an intermediate stop at the destination airport.
func routesThroughDestination(result *models.FlightSearchResult, dest string) bool {
	if result == nil || !result.Success {
		return false
	}
	return FlightRoutesThroughAirport(result.Flights, dest)
}

func buildHiddenCityHack(in DetectorInput, beyond string, beyondPrice, directPrice float64, currency, airlineCode string) Hack {
	bagsWarning := "If you need checked baggage, ask at check-in to short-check it to the intermediate stop; this often works in practice, but confirm it before relying on the strategy"

	risks := []string{
		"Violates airline contracts of carriage; the airline may penalise your account or cancel remaining segments",
		"Cannot be used on a round-trip where you still need later segments; the airline may cancel the rest of the ticket",
		"Flight path may change, so always verify that the itinerary still transits the hub before travel",
		"Operational disruptions can reroute the ticket and break the plan",
	}
	steps := []string{
		fmt.Sprintf("Search flights %s->%s on %s", in.Origin, beyond, in.Date),
		fmt.Sprintf("Confirm the itinerary transits %s", in.Destination),
		fmt.Sprintf("Disembark at %s and skip the onward connection to %s", in.Destination, beyond),
	}

	if in.CarryOnOnly {
		risks = append(risks, "Carry-on only is the simplest setup for this strategy")
		steps = append([]string{"Book a carry-on-only ticket"}, steps...)
	} else {
		risks = append(risks, bagsWarning)
		steps = append([]string{"If you need checked baggage, ask the check-in agent to short-check it to the intermediate stop and collect it before exiting"}, steps...)
	}

	if note := baggage.BaggageNote(airlineCode); note != "" {
		risks = append(risks, note)
	}

	return Hack{
		Type:     "hidden_city",
		Title:    "Hidden city ticketing",
		Currency: currency,
		Savings:  roundSavings(directPrice - beyondPrice),
		Description: fmt.Sprintf(
			"A ticket %s->%s that transits %s costs %s %.0f, cheaper than ending the trip at %s for %.0f. Exit at %s and skip the final leg.",
			in.Origin, beyond, in.Destination, currency, beyondPrice, in.Destination, directPrice, in.Destination,
		),
		Risks: risks,
		Steps: steps,
		Citations: []string{
			googleFlightsURL(beyond, in.Origin, in.Date),
		},
	}
}

// primaryAirlineCode extracts the IATA code of the first airline from flight results.
func primaryAirlineCode(result *models.FlightSearchResult) string {
	if result == nil || !result.Success {
		return ""
	}
	for _, f := range result.Flights {
		for _, leg := range f.Legs {
			if leg.AirlineCode != "" {
				return leg.AirlineCode
			}
		}
	}
	return ""
}

func hiddenCityAirportUniverse() map[string]struct{} {
	universe := make(map[string]struct{}, len(models.AirportNames))
	for code := range models.AirportNames {
		universe[code] = struct{}{}
	}
	return universe
}

func airportCountryCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if cc := iataToCountry[code]; cc != "" {
		return cc
	}
	return hiddenCityCountryHints[code]
}

func airportMacroRegion(code string) string {
	cc := airportCountryCode(code)
	if cc == "" {
		return ""
	}
	return hiddenCityCountryRegions[cc]
}

func regionsAdjacent(a, b string) bool {
	if a == "" || b == "" || a == b {
		return false
	}
	switch a {
	case "europe":
		return b == "middle_east" || b == "eurasia" || b == "africa"
	case "middle_east":
		return b == "europe" || b == "eurasia" || b == "asia" || b == "africa"
	case "eurasia":
		return b == "europe" || b == "middle_east" || b == "asia"
	case "asia":
		return b == "middle_east" || b == "eurasia"
	case "americas":
		return b == "africa"
	case "africa":
		return b == "europe" || b == "middle_east" || b == "americas"
	}
	return false
}

var hiddenCityCountryHints = map[string]string{
	"ALA": "KZ",
	"AUH": "AE",
	"BAK": "AZ",
	"BKK": "TH",
	"BOG": "CO",
	"BOM": "IN",
	"CAN": "CN",
	"CGK": "ID",
	"CMB": "LK",
	"CPT": "ZA",
	"DAC": "BD",
	"DEL": "IN",
	"DOH": "QA",
	"DXB": "AE",
	"EVN": "AM",
	"FRU": "KG",
	"GRU": "BR",
	"KHG": "CN",
	"KHI": "PK",
	"KMG": "CN",
	"KRL": "CN",
	"KUL": "MY",
	"LAD": "AO",
	"LHW": "CN",
	"LIM": "PE",
	"MIA": "US",
	"MNL": "PH",
	"OSS": "KG",
	"PEK": "CN",
	"PKX": "CN",
	"SCL": "CL",
	"SIN": "SG",
	"TAS": "UZ",
	"TBS": "GE",
	"TFU": "CN",
	"URC": "CN",
	"XIY": "CN",
	"YIN": "CN",
}

var hiddenCityCountryRegions = map[string]string{
	"AE": "middle_east",
	"AM": "eurasia",
	"AO": "africa",
	"AT": "europe",
	"AZ": "eurasia",
	"BD": "asia",
	"BE": "europe",
	"BG": "europe",
	"BR": "americas",
	"CH": "europe",
	"CL": "americas",
	"CN": "asia",
	"CO": "americas",
	"CZ": "europe",
	"DE": "europe",
	"DK": "europe",
	"ES": "europe",
	"FI": "europe",
	"FR": "europe",
	"GB": "europe",
	"GE": "eurasia",
	"GR": "europe",
	"HR": "europe",
	"HU": "europe",
	"ID": "asia",
	"IE": "europe",
	"IN": "asia",
	"IS": "europe",
	"IT": "europe",
	"KG": "eurasia",
	"KZ": "eurasia",
	"LK": "asia",
	"LT": "europe",
	"LV": "europe",
	"MY": "asia",
	"NL": "europe",
	"NO": "europe",
	"PE": "americas",
	"PH": "asia",
	"PK": "asia",
	"PL": "europe",
	"PT": "europe",
	"QA": "middle_east",
	"RO": "europe",
	"RS": "europe",
	"SE": "europe",
	"SG": "asia",
	"TH": "asia",
	"TR": "middle_east",
	"US": "americas",
	"UZ": "eurasia",
	"ZA": "africa",
}
