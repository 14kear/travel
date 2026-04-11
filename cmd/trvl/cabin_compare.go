package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/MikkoParkkola/trvl/internal/flights"
	"github.com/MikkoParkkola/trvl/internal/models"
)

// cabinSpec defines a cabin class to search.
type cabinSpec struct {
	Name  string
	Class models.CabinClass
}

var cabinClasses = []cabinSpec{
	{"Economy", models.Economy},
	{"Premium Economy", models.PremiumEconomy},
	{"Business", models.Business},
	{"First", models.First},
}

// cabinResult holds the best flight for a single cabin class.
type cabinResult struct {
	Cabin    string
	Price    float64
	Currency string
	Airline  string
	Stops    int
	Duration int // minutes
	Error    string
}

// runCabinComparison searches all 4 cabin classes in parallel and
// displays a side-by-side comparison table.
func runCabinComparison(ctx context.Context, origins, destinations []string, date string, baseOpts flights.SearchOptions, format string) error {
	results := make([]cabinResult, len(cabinClasses))
	var wg sync.WaitGroup

	for i, cs := range cabinClasses {
		wg.Add(1)
		go func(idx int, spec cabinSpec) {
			defer wg.Done()

			opts := baseOpts
			opts.CabinClass = spec.Class

			var result *models.FlightSearchResult
			var err error
			if len(origins) > 1 || len(destinations) > 1 {
				result, err = flights.SearchMultiAirport(ctx, origins, destinations, date, opts)
			} else {
				result, err = flights.SearchFlights(ctx, origins[0], destinations[0], date, opts)
			}

			if err != nil {
				results[idx] = cabinResult{Cabin: spec.Name, Error: err.Error()}
				return
			}
			if !result.Success || len(result.Flights) == 0 {
				results[idx] = cabinResult{Cabin: spec.Name, Error: "no flights"}
				return
			}

			best := result.Flights[0]
			airline := ""
			duration := 0
			if len(best.Legs) > 0 {
				airline = best.Legs[0].Airline
				duration = best.Legs[0].Duration
			}

			results[idx] = cabinResult{
				Cabin:    spec.Name,
				Price:    best.Price,
				Currency: best.Currency,
				Airline:  airline,
				Stops:    best.Stops,
				Duration: duration,
			}
		}(i, cs)
	}

	wg.Wait()

	if format == "json" {
		return models.FormatJSON(os.Stdout, results)
	}

	route := fmt.Sprintf("%s → %s", strings.Join(origins, ","), strings.Join(destinations, ","))
	models.Banner(os.Stdout, "✈️", fmt.Sprintf("Cabin Comparison · %s · %s", route, date), "All cabin classes searched in parallel")
	fmt.Println()

	headers := []string{"Cabin", "Best Price", "Airline", "Stops", "Duration"}
	var rows [][]string

	for _, r := range results {
		if r.Error != "" {
			rows = append(rows, []string{r.Cabin, "—", "—", "—", r.Error})
			continue
		}
		stopLabel := "nonstop"
		if r.Stops == 1 {
			stopLabel = "1 stop"
		} else if r.Stops > 1 {
			stopLabel = fmt.Sprintf("%d stops", r.Stops)
		}
		dur := "—"
		if r.Duration > 0 {
			dur = fmt.Sprintf("%dh %dm", r.Duration/60, r.Duration%60)
		}
		rows = append(rows, []string{
			r.Cabin,
			fmt.Sprintf("%s %.0f", r.Currency, r.Price),
			r.Airline,
			stopLabel,
			dur,
		})
	}

	models.FormatTable(os.Stdout, headers, rows)
	return nil
}
