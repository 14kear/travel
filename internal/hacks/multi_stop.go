package hacks

import (
	"context"
	"fmt"
	"time"

	"github.com/MikkoParkkola/trvl/internal/flights"
)

// hubStopoverAllowance lists airlines known to allow multi-day layovers at
// their hubs. This is heuristic knowledge and must not be presented as an
// official stopover program unless there is a matching official source.
var hubStopoverAllowance = map[string]struct {
	Airline  string
	MaxNight int
}{
	"AMS": {Airline: "KLM", MaxNight: 2},
	"HEL": {Airline: "Finnair", MaxNight: 5},
	"FRA": {Airline: "Lufthansa", MaxNight: 1},
	"MUC": {Airline: "Lufthansa", MaxNight: 1},
	"CDG": {Airline: "Air France", MaxNight: 1},
	"ZRH": {Airline: "Swiss", MaxNight: 1},
	"VIE": {Airline: "Austrian", MaxNight: 1},
	"IST": {Airline: "Turkish Airlines", MaxNight: 2},
	"DOH": {Airline: "Qatar Airways", MaxNight: 4},
	"DXB": {Airline: "Emirates", MaxNight: 4},
}

// multistopHubs keeps a light destination-specific prior for hubs that often
// produce useful long-layover itineraries. It is advisory only.
var multistopHubs = map[string][]string{
	"PRG": {"AMS", "FRA", "MUC", "CDG"},
	"VIE": {"AMS", "FRA", "MUC"},
	"BUD": {"AMS", "FRA", "MUC", "VIE"},
	"WAW": {"AMS", "FRA", "CDG"},
	"KRK": {"AMS", "FRA"},
	"ARN": {"AMS", "FRA", "CDG"},
	"CPH": {"AMS", "FRA"},
	"OSL": {"AMS", "FRA", "CDG"},
	"ATH": {"AMS", "FRA", "CDG", "IST"},
	"IST": {"AMS", "FRA", "CDG"},
	"DBV": {"AMS", "FRA", "MUC"},
	"SPU": {"AMS", "FRA", "MUC"},
	"BCN": {"AMS", "CDG"},
	"MAD": {"AMS", "CDG"},
	"FCO": {"AMS", "FRA", "CDG", "MUC"},
	"LIS": {"AMS", "CDG"},
	"HEL": {"AMS", "FRA"},
}

// minLayoverMinutesForStopover is the minimum layover duration (in minutes)
// for the layover to be worth flagging as a city-visit opportunity.
const minLayoverMinutesForStopover = 360 // 6 hours

// detectMultiStop identifies itineraries with a long enough intermediate hub
// stop to make a deliberate long layover practical. This is not the same as
// an official stopover program.
func detectMultiStop(ctx context.Context, in DetectorInput) []Hack {
	if !in.valid() || in.Date == "" {
		return nil
	}

	hubs := multistopHubs[in.Destination]

	opts := flights.SearchOptions{}
	if in.ReturnDate != "" {
		opts.ReturnDate = in.ReturnDate
	}
	baseResult, err := flights.SearchFlights(ctx, in.Origin, in.Destination, in.Date, opts)
	if err != nil || !baseResult.Success || len(baseResult.Flights) == 0 {
		return nil
	}
	currency := flightCurrency(baseResult, in.currency())

	var hacks []Hack
	seenHubs := map[string]bool{}

	for _, f := range baseResult.Flights {
		for i, leg := range f.Legs {
			hub := leg.ArrivalAirport.Code
			if seenHubs[hub] {
				continue
			}
			if i == len(f.Legs)-1 {
				continue
			}
			if hub == in.Origin || hub == in.Destination {
				continue
			}

			prog, official := stopoverProgramForHub(hub, leg.AirlineCode)
			if !official {
				prog, official = layoverAllowanceForHub(hub)
			}
			if !sliceContains(hubs, hub) && !official {
				continue
			}

			nextLeg := f.Legs[i+1]
			layover := layoverMinutes(leg.ArrivalTime, nextLeg.DepartureTime)
			if layover < minLayoverMinutesForStopover {
				continue
			}

			seenHubs[hub] = true

			airlineName := leg.Airline
			maxNights := 0
			if prog.Airline != "" {
				airlineName = prog.Airline
				maxNights = prog.MaxNights
			}

			var layoverDesc string
			switch {
			case layover >= 1440:
				layoverDesc = fmt.Sprintf("about %d days", layover/1440)
			default:
				layoverDesc = fmt.Sprintf("about %d hours", layover/60)
			}

			steps := []string{
				fmt.Sprintf("Book %s -> %s via %s with %s", in.Origin, in.Destination, hub, airlineName),
				fmt.Sprintf("If timings, airport access, and visa rules allow, use the %s layover at %s as a deliberate long stop", layoverDesc, hub),
				"Ensure you have any required transit visa for " + hub,
			}
			if prog.Official && maxNights > 0 {
				steps = append(steps, fmt.Sprintf("Ask %s for an official stopover extension of up to %d nights", airlineName, maxNights))
			} else if maxNights > 0 {
				steps = append(steps, fmt.Sprintf("Check with %s whether your fare allows extending the layover at %s for up to %d nights", airlineName, hub, maxNights))
			}

			citations := []string{
				googleFlightsURL(in.Destination, in.Origin, in.Date),
			}
			if prog.Official && prog.URL != "" {
				citations = append(citations, prog.URL)
			}

			description := fmt.Sprintf(
				"Your %s -> %s routing via %s has %s at %s (%s). This can make a planned long layover practical on one itinerary.",
				in.Origin, in.Destination, hub, layoverDesc, hub, airlineName,
			)
			if prog.Official {
				description = fmt.Sprintf(
					"Your %s -> %s routing via %s has %s at %s (%s), and %s has an official stopover program there.",
					in.Origin, in.Destination, hub, layoverDesc, hub, airlineName, airlineName,
				)
			}

			risks := []string{
				"Long layovers do not automatically mean an official free stopover program exists",
				"Any stopover extension must be requested at booking and may depend on fare rules",
				"Visa required for some hub countries depending on your nationality",
				"Airline schedule changes may shorten your layover without notice",
			}

			hacks = append(hacks, Hack{
				Type:        "multi_stop",
				Title:       fmt.Sprintf("Two-city trip: visit %s on the way to %s", hubCityName(hub), in.Destination),
				Currency:    currency,
				Savings:     0,
				Description: description,
				Risks:       risks,
				Steps:       steps,
				Citations:   citations,
			})
		}
	}

	return hacks
}

// sliceContains returns true if s contains v.
func sliceContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// layoverMinutes computes the layover in minutes between an arrival ISO time
// and the next departure ISO time. Returns 0 on parse errors.
func layoverMinutes(arrivalISO, departureISO string) int {
	arr, err1 := parseDatetime(arrivalISO)
	dep, err2 := parseDatetime(departureISO)
	if err1 != nil || err2 != nil {
		return 0
	}
	diff := dep.Sub(arr).Minutes()
	if diff < 0 {
		return 0
	}
	return int(diff)
}

// parseDatetime parses an ISO 8601 datetime string. Tries multiple formats.
func parseDatetime(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse datetime: %s", s)
}
