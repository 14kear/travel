package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/MikkoParkkola/trvl/internal/flights"
	"github.com/MikkoParkkola/trvl/internal/models"
	"github.com/spf13/cobra"
)

func flightsCmd() *cobra.Command {
	var (
		returnDate string
		cabin      string
		maxStops   string
		sortBy     string
		airlines   []string
		adults     int
		format     string
	)

	cmd := &cobra.Command{
		Use:   "flights ORIGIN DESTINATION DATE",
		Short: "Search flights between two airports",
		Long: `Search flights between two airports on a specific date.

ORIGIN and DESTINATION are IATA airport codes (e.g. HEL, NRT, JFK).
DATE is the departure date in YYYY-MM-DD format.

Examples:
  trvl flights HEL NRT 2026-06-15
  trvl flights HEL NRT 2026-06-15 --return 2026-06-22
  trvl flights HEL NRT 2026-06-15 --cabin business --stops nonstop
  trvl flights HEL NRT 2026-06-15 --format json`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			origin := strings.ToUpper(args[0])
			destination := strings.ToUpper(args[1])
			date := args[2]

			cabinClass, err := models.ParseCabinClass(cabin)
			if err != nil {
				return fmt.Errorf("invalid cabin class: %w", err)
			}

			stops, err := models.ParseMaxStops(maxStops)
			if err != nil {
				return fmt.Errorf("invalid max stops: %w", err)
			}

			sort, err := models.ParseSortBy(sortBy)
			if err != nil {
				return fmt.Errorf("invalid sort order: %w", err)
			}

			opts := flights.SearchOptions{
				ReturnDate: returnDate,
				CabinClass: cabinClass,
				MaxStops:   stops,
				SortBy:     sort,
				Airlines:   airlines,
				Adults:     adults,
			}

			result, err := flights.SearchFlights(cmd.Context(), origin, destination, date, opts)
			if err != nil {
				return err
			}

			if format == "json" {
				return models.FormatJSON(os.Stdout, result)
			}

			return printFlightsTable(result)
		},
	}

	cmd.Flags().StringVar(&returnDate, "return", "", "Return date for round-trip (YYYY-MM-DD)")
	cmd.Flags().StringVar(&cabin, "cabin", "economy", "Cabin class: economy, premium_economy, business, first")
	cmd.Flags().StringVar(&maxStops, "stops", "any", "Max stops: any, nonstop, one_stop, two_plus")
	cmd.Flags().StringVar(&sortBy, "sort", "", "Sort by: cheapest, duration, departure, arrival")
	cmd.Flags().StringSliceVar(&airlines, "airline", nil, "Filter by airline IATA code (repeatable)")
	cmd.Flags().IntVar(&adults, "adults", 1, "Number of adult passengers")
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")

	cmd.ValidArgsFunction = airportCompletion

	return cmd
}

// printFlightsTable renders flight results as an ASCII table.
func printFlightsTable(result *models.FlightSearchResult) error {
	if !result.Success {
		fmt.Fprintf(os.Stderr, "Search failed: %s\n", result.Error)
		return nil
	}

	if result.Count == 0 {
		fmt.Println("No flights found.")
		return nil
	}

	models.Banner(os.Stdout, "✈️", fmt.Sprintf("Flights · %s", result.TripType),
		fmt.Sprintf("Found %d flights", result.Count))
	fmt.Println()

	headers := []string{"Price", "Duration", "Stops", "Route", "Airline", "Flight", "Departs", "Arrives"}
	var rows [][]string

	for _, f := range result.Flights {
		route := flightRoute(f)
		airline := ""
		flightNum := ""
		departs := ""
		arrives := ""

		if len(f.Legs) > 0 {
			airline = f.Legs[0].Airline
			flightNum = f.Legs[0].FlightNumber
			departs = f.Legs[0].DepartureTime
			arrives = f.Legs[len(f.Legs)-1].ArrivalTime
		}

		rows = append(rows, []string{
			formatPrice(f.Price, f.Currency),
			formatDuration(f.Duration),
			formatStops(f.Stops),
			route,
			airline,
			flightNum,
			departs,
			arrives,
		})
	}

	models.FormatTable(os.Stdout, headers, rows)

	// Summary: cheapest flight
	if len(result.Flights) > 0 {
		cheapest := result.Flights[0]
		for _, f := range result.Flights[1:] {
			if f.Price > 0 && f.Price < cheapest.Price {
				cheapest = f
			}
		}
		airline := ""
		if len(cheapest.Legs) > 0 {
			airline = cheapest.Legs[0].Airline
		}
		models.Summary(os.Stdout, fmt.Sprintf("Cheapest: %s %.0f (%s, %s)",
			cheapest.Currency, cheapest.Price, airline, formatStops(cheapest.Stops)))
		models.BookingHint(os.Stdout)
	}
	return nil
}

// flightRoute builds a route string like "HEL -> FRA -> NRT".
func flightRoute(f models.FlightResult) string {
	if len(f.Legs) == 0 {
		return ""
	}

	parts := []string{f.Legs[0].DepartureAirport.Code}
	for _, leg := range f.Legs {
		parts = append(parts, leg.ArrivalAirport.Code)
	}
	return strings.Join(parts, " -> ")
}

// formatPrice formats a price with currency.
func formatPrice(amount float64, currency string) string {
	if amount == 0 {
		return "-"
	}
	return fmt.Sprintf("%s %.0f", currency, amount)
}

// formatDuration converts minutes to a human-readable duration string.
func formatDuration(minutes int) string {
	if minutes == 0 {
		return "-"
	}
	h := minutes / 60
	m := minutes % 60
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

// formatStops returns a human-readable stops string.
func formatStops(stops int) string {
	switch stops {
	case 0:
		return "Direct"
	case 1:
		return "1 stop"
	default:
		return fmt.Sprintf("%d stops", stops)
	}
}
