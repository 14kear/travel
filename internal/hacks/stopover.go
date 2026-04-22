package hacks

import (
	"context"
	"fmt"

	"github.com/MikkoParkkola/trvl/internal/flights"
	"github.com/MikkoParkkola/trvl/internal/models"
)

// detectStopover identifies when a route passes through an airline hub that
// offers an official multi-day stopover program.
func detectStopover(ctx context.Context, in DetectorInput) []Hack {
	if !in.valid() || in.Date == "" {
		return nil
	}

	result, err := flights.SearchFlights(ctx, in.Origin, in.Destination, in.Date, flights.SearchOptions{})
	if err != nil || !result.Success {
		return nil
	}

	seen := map[string]bool{}
	var hacks []Hack

	for _, f := range result.Flights {
		prog, hub, ok := StopoverOpportunityForFlight(in.Origin, in.Destination, f)
		if !ok || seen[hub] {
			continue
		}
		seen[hub] = true
		hacks = append(hacks, buildStopoverHack(in, prog, f, hub))
	}

	return hacks
}

// matchStopoverProgram returns an official stopover program if the airline/hub
// pair matches any registered official program.
func matchStopoverProgram(hub, airlineCode string) (StopoverProgram, bool) {
	if prog, ok := stopoverPrograms[airlineCode]; ok && prog.Hub == hub {
		return prog, true
	}
	for _, prog := range stopoverPrograms {
		if prog.Hub == hub {
			return prog, true
		}
	}
	return StopoverProgram{}, false
}

// stopoverProgramForHub resolves an official stopover program for a hub.
func stopoverProgramForHub(hub, airlineCode string) (StopoverProgram, bool) {
	if prog, ok := matchStopoverProgram(hub, airlineCode); ok {
		return prog, true
	}
	return StopoverProgram{}, false
}

// layoverAllowanceForHub resolves heuristic hub knowledge used for multi-stop
// suggestions. These are not official stopover programs and must not be
// surfaced as such.
func layoverAllowanceForHub(hub string) (StopoverProgram, bool) {
	if info, ok := hubStopoverAllowance[hub]; ok {
		return StopoverProgram{
			Airline:      info.Airline,
			Hub:          hub,
			MaxNights:    info.MaxNight,
			Restrictions: "Allowance depends on fare rules and booking channel",
			Official:     false,
		}, true
	}
	return StopoverProgram{}, false
}

// StopoverOpportunityForFlight inspects a priced itinerary and returns the
// first official airline stopover program that matches one of its intermediate hubs.
func StopoverOpportunityForFlight(origin, destination string, f models.FlightResult) (StopoverProgram, string, bool) {
	for _, leg := range f.Legs {
		hub := leg.ArrivalAirport.Code
		if hub == "" || hub == origin || hub == destination {
			continue
		}
		prog, ok := stopoverProgramForHub(hub, leg.AirlineCode)
		if ok {
			return prog, hub, true
		}
	}
	return StopoverProgram{}, "", false
}

func buildStopoverHack(in DetectorInput, prog StopoverProgram, f models.FlightResult, hub string) Hack {
	currency := in.currency()
	if f.Currency != "" {
		currency = f.Currency
	}

	hubName := hubCityName(hub)
	restrictions := prog.Restrictions
	if restrictions == "" {
		restrictions = "Availability depends on fare rules and booking channel"
	}
	steps := []string{
		fmt.Sprintf("Book your %s->%s ticket with %s via %s", in.Origin, in.Destination, prog.Airline, hub),
		fmt.Sprintf("Request a stopover in %s at time of booking (up to %d nights)", hubName, prog.MaxNights),
		"Check visa requirements for " + hubName,
		"Book accommodation in " + hubName + " (not included)",
	}
	if prog.URL == "" {
		steps[1] = fmt.Sprintf("Check with %s whether your fare allows a stopover in %s (up to %d nights)", prog.Airline, hubName, prog.MaxNights)
	}

	var citations []string
	if prog.URL != "" {
		citations = append(citations, prog.URL)
	}

	return Hack{
		Type:     "stopover",
		Title:    fmt.Sprintf("Official %s stopover (%s)", hubName, prog.Airline),
		Currency: currency,
		Savings:  0, // Stopover is a value-add, not a price saving vs naive booking
		Description: fmt.Sprintf(
			"%s offers an official stopover program in %s (up to %d nights) when transiting through %s. "+
				"Check the fare rules and booking channel requirements before relying on it.",
			prog.Airline, hubName, prog.MaxNights, hub,
		),
		Risks: []string{
			fmt.Sprintf("Restrictions: %s", restrictions),
			"Check whether the fare must be booked directly with " + prog.Airline,
			"Stopover programs are subject to availability and may change without notice",
			"Visa requirements for " + hubName + " may apply depending on your nationality",
		},
		Steps:     steps,
		Citations: citations,
	}
}

// StopoverCityName returns a human-readable city name for a stopover hub.
func StopoverCityName(code string) string {
	return hubCityName(code)
}

// hubCityName returns a human-readable city name for a hub airport code.
func hubCityName(code string) string {
	names := map[string]string{
		"HEL": "Helsinki",
		"KEF": "Reykjavik",
		"LIS": "Lisbon",
		"IST": "Istanbul",
		"DOH": "Doha",
		"DXB": "Dubai",
		"SIN": "Singapore",
		"AUH": "Abu Dhabi",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return models.LookupAirportName(code)
}
