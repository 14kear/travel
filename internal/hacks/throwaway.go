package hacks

import (
	"context"
	"fmt"

	"github.com/MikkoParkkola/trvl/internal/baggage"
	"github.com/MikkoParkkola/trvl/internal/flights"
)

// detectThrowaway finds cases where a round-trip ticket is cheaper than a
// one-way, allowing the traveller to book round-trip and skip the return leg.
func detectThrowaway(ctx context.Context, in DetectorInput) []Hack {
	if !in.valid() || in.Date == "" {
		return nil
	}

	returnDate := in.ReturnDate
	if returnDate == "" {
		returnDate = addDays(in.Date, 7)
		if returnDate == "" {
			return nil
		}
	}

	owResult, err := flights.SearchFlights(ctx, in.Origin, in.Destination, in.Date, flights.SearchOptions{})
	if err != nil || !owResult.Success || len(owResult.Flights) == 0 {
		return nil
	}
	owPrice := minFlightPrice(owResult)
	if owPrice <= 0 {
		return nil
	}

	rtResult, err := flights.SearchFlights(ctx, in.Origin, in.Destination, in.Date, flights.SearchOptions{
		ReturnDate: returnDate,
	})
	if err != nil || !rtResult.Success || len(rtResult.Flights) == 0 {
		return nil
	}
	rtPrice := minFlightPrice(rtResult)
	if rtPrice <= 0 {
		return nil
	}

	if rtPrice >= owPrice {
		return nil
	}

	savings := owPrice - rtPrice
	if savings < 15 {
		return nil
	}

	currency := flightCurrency(owResult, in.currency())
	airlineCode := primaryAirlineCode(owResult)

	risks := []string{
		"Violates most airline contracts of carriage; the airline may cancel the return leg without refund",
		"Frequent-flyer account may be penalised or closed",
		"If you miss the outbound leg, the return is void automatically",
		"Do not check bags; luggage is tagged to the final destination",
	}
	if note := baggage.BaggageNote(airlineCode); note != "" {
		risks = append(risks, note)
	}

	return []Hack{{
		Type:     "throwaway",
		Title:    "Throwaway ticketing",
		Currency: currency,
		Savings:  roundSavings(savings),
		Description: fmt.Sprintf(
			"Round-trip %s->%s costs %s %.0f versus %.0f for the one-way. Book the round-trip and intentionally discard the return.",
			in.Origin, in.Destination, currency, rtPrice, owPrice,
		),
		Risks: risks,
		Steps: []string{
			fmt.Sprintf("Search round-trip %s->%s departing %s and returning %s", in.Origin, in.Destination, in.Date, returnDate),
			"Book the cheapest round-trip option",
			"Only board the outbound leg and ignore the return segment",
		},
		Citations: []string{
			googleFlightsURL(in.Destination, in.Origin, in.Date),
		},
	}}
}
