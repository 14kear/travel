package hacks

import (
	"context"
	"fmt"
	"strings"
)

// alternativeAirport describes a cheaper alternative airport near a major
// destination, together with ground-transit details.
type alternativeAirport struct {
	IATA          string
	City          string
	TransportCost float64 // EUR one-way
	TransportMin  int     // minutes to city centre
	TransportMode string  // "bus", "train", etc.
	Notes         string  // human-readable transit description
}

// destinationAlternatives maps a primary destination airport to nearby
// alternatives that are often served by low-cost carriers at lower fares.
var destinationAlternatives = map[string][]alternativeAirport{
	"MXP": {{IATA: "BGY", City: "Bergamo", TransportCost: 10, TransportMin: 60, TransportMode: "bus", Notes: "Orio al Serio shuttle to Milano Centrale"}},
	"LIN": {{IATA: "BGY", City: "Bergamo", TransportCost: 10, TransportMin: 60, TransportMode: "bus", Notes: "Orio al Serio shuttle"}},
	"BCN": {{IATA: "GRO", City: "Girona", TransportCost: 12, TransportMin: 75, TransportMode: "bus", Notes: "Sagales bus to Barcelona centre"}},
	"CDG": {
		{IATA: "BVA", City: "Beauvais", TransportCost: 17, TransportMin: 80, TransportMode: "bus", Notes: "Navette shuttle to Porte Maillot"},
		{IATA: "ORY", City: "Orly", TransportCost: 10, TransportMin: 40, TransportMode: "train", Notes: "OrlyVal plus RER"},
	},
	"LHR": {
		{IATA: "STN", City: "London Stansted", TransportCost: 12, TransportMin: 50, TransportMode: "train", Notes: "Stansted Express"},
		{IATA: "LTN", City: "London Luton", TransportCost: 15, TransportMin: 45, TransportMode: "train", Notes: "Thameslink plus shuttle"},
	},
	"FCO": {{IATA: "CIA", City: "Ciampino", TransportCost: 6, TransportMin: 40, TransportMode: "bus", Notes: "SIT or Terravision to Termini"}},
	"OSL": {{IATA: "TRF", City: "Torp Sandefjord", TransportCost: 15, TransportMin: 90, TransportMode: "bus", Notes: "Torp-ekspressen to Oslo"}},
	"ARN": {
		{IATA: "NYO", City: "Skavsta", TransportCost: 15, TransportMin: 80, TransportMode: "bus", Notes: "Flygbussarna to Stockholm"},
		{IATA: "VST", City: "Vasteras", TransportCost: 15, TransportMin: 75, TransportMode: "bus", Notes: "Bus to Stockholm"},
	},
	"BRU": {{IATA: "CRL", City: "Charleroi", TransportCost: 15, TransportMin: 60, TransportMode: "bus", Notes: "Flibco shuttle to Brussels"}},
	"AMS": {{IATA: "EIN", City: "Eindhoven", TransportCost: 12, TransportMin: 90, TransportMode: "train", Notes: "Train to Amsterdam Centraal"}},
	"CPH": {{IATA: "MMX", City: "Malmo", TransportCost: 10, TransportMin: 35, TransportMode: "train", Notes: "Oresund train to Copenhagen"}},
}

// detectDestinationAirport suggests checking alternative destination airports
// that may offer cheaper fares on low-cost carriers. This is purely advisory.
func detectDestinationAirport(_ context.Context, in DetectorInput) []Hack {
	if !in.valid() {
		return nil
	}

	alternatives, ok := destinationAlternatives[in.Destination]
	if !ok {
		return nil
	}

	currency := in.currency()
	var hacks []Hack
	for _, alt := range alternatives {
		if alt.IATA == in.Origin {
			continue
		}

		groundCost := alt.TransportCost
		if converted, cur := ApproxConvertCurrency(alt.TransportCost, "EUR", currency); cur == currency {
			groundCost = converted
		}

		searchURL := fmt.Sprintf(
			"https://www.google.com/travel/flights?q=Flights+to+%s+from+%s",
			alt.IATA, in.Origin,
		)
		if in.Date != "" {
			searchURL += "+on+" + in.Date
		}

		hacks = append(hacks, Hack{
			Type:     "destination_airport",
			Title:    fmt.Sprintf("Fly into %s (%s) instead of %s", alt.City, alt.IATA, in.Destination),
			Currency: currency,
			Savings:  0,
			Description: fmt.Sprintf(
				"Low-cost carriers often fly to %s (%s) at lower fares than %s. Ground transit to the city: %s, about %d min, about %.0f %s. %s.",
				alt.City, alt.IATA, in.Destination,
				alt.TransportMode, alt.TransportMin, groundCost, currency,
				alt.Notes,
			),
			Risks: []string{
				fmt.Sprintf("Ground transit adds about %d minutes to reach the city centre", alt.TransportMin),
				"Transport schedules may not align with late-night or early-morning arrivals",
				fmt.Sprintf("Budget about %.0f %s for the %s transfer", groundCost, currency, alt.TransportMode),
			},
			Steps: []string{
				fmt.Sprintf("Compare prices: search %s->%s alongside %s->%s", in.Origin, alt.IATA, in.Origin, in.Destination),
				fmt.Sprintf("If cheaper, book %s->%s and take %s to the city (%s)", in.Origin, alt.IATA, alt.TransportMode, alt.Notes),
				fmt.Sprintf("Factor in about %.0f %s of ground transport when comparing the total trip price", groundCost, currency),
			},
			Citations: []string{searchURL},
		})
	}

	return hacks
}

// destinationCity returns a human-readable city name for common IATA codes
// used by destination alternatives. Falls back to the code itself.
func destinationCity(iata string) string {
	cities := map[string]string{
		"MXP": "Milan", "LIN": "Milan", "BGY": "Bergamo",
		"BCN": "Barcelona", "GRO": "Girona",
		"CDG": "Paris", "BVA": "Beauvais", "ORY": "Paris Orly",
		"LHR": "London", "STN": "London Stansted", "LTN": "London Luton",
		"FCO": "Rome", "CIA": "Rome Ciampino",
		"OSL": "Oslo", "TRF": "Torp Sandefjord",
		"ARN": "Stockholm", "NYO": "Skavsta", "VST": "Vasteras",
		"BRU": "Brussels", "CRL": "Charleroi",
		"AMS": "Amsterdam", "EIN": "Eindhoven",
		"CPH": "Copenhagen", "MMX": "Malmo",
	}
	if name, ok := cities[iata]; ok {
		return name
	}
	return iata
}

// destinationAlternativesForDisplay returns a formatted summary of all known
// destination alternatives, useful for the MCP tool description.
func destinationAlternativesForDisplay() string {
	var sb strings.Builder
	for dest, alts := range destinationAlternatives {
		for _, alt := range alts {
			fmt.Fprintf(&sb, "%s -> %s (%s): %s, ~%d min, ~%.0f EUR\n",
				dest, alt.IATA, alt.City, alt.TransportMode, alt.TransportMin, alt.TransportCost)
		}
	}
	return sb.String()
}
