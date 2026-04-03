// Example: search flights from Helsinki to Barcelona
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/MikkoParkkola/trvl/internal/batchexec"
	"github.com/MikkoParkkola/trvl/internal/flights"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := batchexec.NewClient()
	result, err := flights.SearchFlightsWithClient(ctx, client, "HEL", "BCN", "2026-07-01", flights.SearchOptions{})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Found %d flights\n", result.Count)
	for i, f := range result.Flights {
		if i >= 5 {
			break
		}
		fmt.Printf("  %s %.0f — %s %s (%d stops)\n", f.Currency, f.Price, f.Legs[0].Airline, f.Legs[0].FlightNumber, f.Stops)
	}
}
