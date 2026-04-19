package optimizer

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MikkoParkkola/trvl/internal/baggage"
	"github.com/MikkoParkkola/trvl/internal/batchexec"
	"github.com/MikkoParkkola/trvl/internal/flights"
	"github.com/MikkoParkkola/trvl/internal/hacks"
	"github.com/MikkoParkkola/trvl/internal/models"
)

// OptimizeInput configures a trip optimization search.
type OptimizeInput struct {
	Origin      string // primary origin IATA (e.g., "HEL")
	Destination string // primary destination IATA or city (e.g., "BCN")
	DepartDate  string // target departure (YYYY-MM-DD)
	ReturnDate  string // target return (YYYY-MM-DD), empty = one-way
	FlexDays    int    // date flexibility +/-N days (default 3)
	Guests      int    // passengers (default 1)
	Currency    string // display currency
	MaxResults  int    // top N results to return (default 5)
	MaxAPICalls int    // API call budget (default 15)

	// User context (from preferences).
	FFStatuses     []FFStatus // frequent flyer statuses
	NeedCheckedBag bool
	CarryOnOnly    bool
	HomeAirports   []string // user's home airports
}

// FFStatus represents a frequent flyer programme membership.
type FFStatus struct {
	Alliance string
	Tier     string
}

// BookingOption is a ranked booking strategy with all-in cost breakdown.
type BookingOption struct {
	Rank              int      `json:"rank"`
	Strategy          string   `json:"strategy"`
	Legs              []Leg    `json:"legs"`
	BaseCost          float64  `json:"base_cost"`
	BagCost           float64  `json:"bag_cost"`
	FFSavings         float64  `json:"ff_savings"`
	TransferCost      float64  `json:"transfer_cost"`
	AllInCost         float64  `json:"all_in_cost"`
	Currency          string   `json:"currency"`
	SavingsVsBaseline float64  `json:"savings_vs_baseline"`
	HacksApplied      []string `json:"hacks_applied"`
}

// Leg is a single transport segment in a booking option.
type Leg struct {
	Type     string  `json:"type"`
	From     string  `json:"from"`
	To       string  `json:"to"`
	Date     string  `json:"date"`
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
	Airline  string  `json:"airline,omitempty"`
	Duration int     `json:"duration_min,omitempty"`
	Notes    string  `json:"notes,omitempty"`
}

// OptimizeResult is the output of the optimization engine.
type OptimizeResult struct {
	Success  bool           `json:"success"`
	Options  []BookingOption `json:"options"`
	Baseline *BookingOption  `json:"baseline,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// candidate is an internal search candidate generated during the EXPAND phase.
type candidate struct {
	origin       string
	dest         string
	departDate   string
	returnDate   string
	strategy     string
	hackTypes    []string
	transferCost float64
	transferTime int // minutes

	// Populated during SEARCH phase.
	searched bool
	flights  []models.FlightResult
	currency string

	// Populated during PRICE phase.
	baseCost  float64
	bagCost   float64
	ffSavings float64
	allInCost float64
}

// defaults fills zero-value fields with sensible defaults.
func (in *OptimizeInput) defaults() {
	if in.Guests <= 0 {
		in.Guests = 1
	}
	if in.FlexDays < 0 {
		in.FlexDays = 0
	}
	if in.FlexDays == 0 {
		in.FlexDays = 3
	}
	if in.MaxResults <= 0 {
		in.MaxResults = 5
	}
	if in.MaxAPICalls <= 0 {
		in.MaxAPICalls = 15
	}
	if in.Currency == "" {
		in.Currency = "EUR"
	}
}

// Optimize runs the 4-phase trip optimization engine.
// It expands candidates from pricing primitives, searches them in parallel,
// applies all-in cost adjustments, and ranks the results.
func Optimize(ctx context.Context, input OptimizeInput) (*OptimizeResult, error) {
	if err := validateInput(input); err != nil {
		return &OptimizeResult{Error: err.Error()}, err
	}

	input.defaults()

	// Apply 30s total timeout.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Phase 1: EXPAND candidates.
	candidates := expandCandidates(input)

	// Phase 2: SEARCH candidates with budget.
	client := batchexec.NewClient()
	searchCandidates(ctx, candidates, client, input)

	// Phase 3: PRICE candidates.
	for _, c := range candidates {
		if c.searched {
			priceCandidate(c, input)
		}
	}

	// Phase 4: RANK by all-in cost.
	return rankCandidates(candidates, input), nil
}

func validateInput(in OptimizeInput) error {
	if in.Origin == "" {
		return fmt.Errorf("origin is required")
	}
	if in.Destination == "" {
		return fmt.Errorf("destination is required")
	}
	if in.DepartDate == "" {
		return fmt.Errorf("departure date is required")
	}
	if _, err := time.Parse("2006-01-02", in.DepartDate); err != nil {
		return fmt.Errorf("invalid departure date: %s", in.DepartDate)
	}
	if in.ReturnDate != "" {
		if _, err := time.Parse("2006-01-02", in.ReturnDate); err != nil {
			return fmt.Errorf("invalid return date: %s", in.ReturnDate)
		}
	}
	if strings.EqualFold(in.Origin, in.Destination) {
		return fmt.Errorf("origin and destination must differ")
	}
	return nil
}

// expandCandidates generates all candidate search parameters from applicable
// pricing primitives.
func expandCandidates(input OptimizeInput) []*candidate {
	origin := strings.ToUpper(input.Origin)
	dest := strings.ToUpper(input.Destination)

	var candidates []*candidate

	// 1. Baseline: direct search with given parameters.
	candidates = append(candidates, &candidate{
		origin:     origin,
		dest:       dest,
		departDate: input.DepartDate,
		returnDate: input.ReturnDate,
		strategy:   "Direct booking",
	})

	// 2. Alternative origins (positioning).
	for _, alt := range hacks.NearbyAirports(origin) {
		if alt.IATA == dest {
			continue
		}
		candidates = append(candidates, &candidate{
			origin:       alt.IATA,
			dest:         dest,
			departDate:   input.DepartDate,
			returnDate:   input.ReturnDate,
			strategy:     fmt.Sprintf("Fly from %s (%s via %s)", alt.City, alt.IATA, alt.Mode),
			hackTypes:    []string{alt.HackType},
			transferCost: alt.Cost,
			transferTime: alt.Minutes,
		})
	}

	// 3. Alternative destinations.
	for _, alt := range hacks.DestinationAlternatives(dest) {
		if alt.IATA == origin {
			continue
		}
		candidates = append(candidates, &candidate{
			origin:       origin,
			dest:         alt.IATA,
			departDate:   input.DepartDate,
			returnDate:   input.ReturnDate,
			strategy:     fmt.Sprintf("Fly to %s (%s) + %s to %s", alt.City, alt.IATA, alt.Mode, dest),
			hackTypes:    []string{"destination_airport"},
			transferCost: alt.Cost,
		})
	}

	// 4. Rail+fly stations (fare zone arbitrage).
	for _, station := range hacks.RailFlyStationsForHub(origin) {
		candidates = append(candidates, &candidate{
			origin:     station.IATA,
			dest:       dest,
			departDate: input.DepartDate,
			returnDate: input.ReturnDate,
			strategy:   fmt.Sprintf("Book via %s (%s fare zone, %s)", station.City, station.FareZone, station.AirlineName),
			hackTypes:  []string{"rail_fly_arbitrage"},
			// Rail segment is free — included in the airline ticket.
			transferCost: 0,
			transferTime: station.TrainMins,
		})
	}

	return candidates
}

// searchCandidates executes flight searches for candidates within the API budget.
// It prioritizes the baseline (direct) search first, then alternatives.
func searchCandidates(ctx context.Context, candidates []*candidate, client *batchexec.Client, input OptimizeInput) {
	budget := int64(input.MaxAPICalls)
	var used atomic.Int64

	// Sort: baseline (no hacks) first, then by expected lower transfer cost.
	sorted := make([]*candidate, len(candidates))
	copy(sorted, candidates)
	sort.SliceStable(sorted, func(i, j int) bool {
		// Baseline first.
		if len(sorted[i].hackTypes) == 0 {
			return true
		}
		if len(sorted[j].hackTypes) == 0 {
			return false
		}
		return sorted[i].transferCost < sorted[j].transferCost
	})

	// Use semaphore for concurrency control (max 4 parallel searches).
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup

	for _, c := range sorted {
		if used.Load() >= budget {
			break
		}
		if ctx.Err() != nil {
			break
		}

		c := c
		wg.Add(1)
		sem <- struct{}{} // acquire

		go func() {
			defer wg.Done()
			defer func() { <-sem }() // release

			if used.Add(1) > budget {
				return
			}

			opts := flights.SearchOptions{
				SortBy: models.SortCheapest,
				Adults: input.Guests,
			}
			if c.returnDate != "" {
				opts.ReturnDate = c.returnDate
			}

			result, err := flights.SearchFlightsWithClient(ctx, client, c.origin, c.dest, c.departDate, opts)
			if err != nil || result == nil || !result.Success || len(result.Flights) == 0 {
				return
			}

			c.searched = true
			c.flights = result.Flights
			if len(result.Flights) > 0 && result.Flights[0].Currency != "" {
				c.currency = result.Flights[0].Currency
			}
		}()
	}

	wg.Wait()
}

// priceCandidate computes all-in cost for a searched candidate.
func priceCandidate(c *candidate, input OptimizeInput) {
	if len(c.flights) == 0 {
		return
	}

	// Find cheapest flight.
	bestFlight := c.flights[0]
	for _, f := range c.flights[1:] {
		if f.Price > 0 && (bestFlight.Price <= 0 || f.Price < bestFlight.Price) {
			bestFlight = f
		}
	}
	if bestFlight.Price <= 0 {
		return
	}

	c.baseCost = bestFlight.Price
	c.currency = bestFlight.Currency

	// Compute baggage costs via AllInCost.
	ffStatuses := convertFFStatuses(input.FFStatuses)
	airlineCode := ""
	if len(bestFlight.Legs) > 0 {
		airlineCode = bestFlight.Legs[0].AirlineCode
	}

	allIn, _ := baggage.AllInCost(
		bestFlight.Price,
		airlineCode,
		input.NeedCheckedBag,
		!input.CarryOnOnly, // needCarryOn = opposite of carryOnOnly
		ffStatuses,
	)

	c.bagCost = allIn - bestFlight.Price
	if c.bagCost < 0 {
		c.bagCost = 0
	}

	// FF savings: difference between cost without FF and cost with FF.
	allInNoFF, _ := baggage.AllInCost(
		bestFlight.Price,
		airlineCode,
		input.NeedCheckedBag,
		!input.CarryOnOnly,
		nil, // no FF statuses
	)
	c.ffSavings = allInNoFF - allIn
	if c.ffSavings < 0 {
		c.ffSavings = 0
	}

	// All-in = base + bags - FF savings + transfer cost
	c.allInCost = allIn + c.transferCost
}

// rankCandidates sorts by all-in cost and returns the top N options.
func rankCandidates(candidates []*candidate, input OptimizeInput) *OptimizeResult {
	// Filter to only searched + priced candidates.
	var priced []*candidate
	for _, c := range candidates {
		if c.searched && c.allInCost > 0 {
			priced = append(priced, c)
		}
	}

	if len(priced) == 0 {
		return &OptimizeResult{
			Error: "no results found for any candidate strategy",
		}
	}

	// Sort by all-in cost ascending.
	sort.Slice(priced, func(i, j int) bool {
		return priced[i].allInCost < priced[j].allInCost
	})

	// Identify baseline (the direct booking candidate).
	var baseline *candidate
	for _, c := range priced {
		if len(c.hackTypes) == 0 {
			baseline = c
			break
		}
	}

	// Build result.
	n := input.MaxResults
	if n > len(priced) {
		n = len(priced)
	}

	result := &OptimizeResult{
		Success: true,
		Options: make([]BookingOption, n),
	}

	for i := 0; i < n; i++ {
		c := priced[i]
		opt := candidateToOption(c, i+1, input)

		// Compute savings vs baseline.
		if baseline != nil && baseline.allInCost > 0 {
			opt.SavingsVsBaseline = math.Round(baseline.allInCost - opt.AllInCost)
		}

		result.Options[i] = opt
	}

	// Set baseline if it was found and priced.
	if baseline != nil {
		bl := candidateToOption(baseline, 0, input)
		result.Baseline = &bl
	}

	return result
}

// candidateToOption converts an internal candidate to the public BookingOption.
func candidateToOption(c *candidate, rank int, input OptimizeInput) BookingOption {
	var legs []Leg

	// Add transfer leg if there is a positioning cost.
	if c.transferCost > 0 {
		legs = append(legs, Leg{
			Type:     "ground",
			From:     strings.ToUpper(input.Origin),
			To:       c.origin,
			Date:     c.departDate,
			Price:    c.transferCost,
			Currency: c.currency,
			Notes:    "Ground transfer",
		})
	}

	// Add outbound flight leg.
	if len(c.flights) > 0 {
		best := cheapestFlight(c.flights)
		airline := ""
		duration := best.Duration
		if len(best.Legs) > 0 {
			airline = best.Legs[0].Airline
		}
		legs = append(legs, Leg{
			Type:     "flight",
			From:     c.origin,
			To:       c.dest,
			Date:     c.departDate,
			Price:    best.Price,
			Currency: best.Currency,
			Airline:  airline,
			Duration: duration,
		})
	}

	// Add destination transfer leg if it's an alternative destination.
	for _, h := range c.hackTypes {
		if h == "destination_airport" && c.transferCost > 0 {
			legs = append(legs, Leg{
				Type:     "ground",
				From:     c.dest,
				To:       strings.ToUpper(input.Destination),
				Date:     c.departDate,
				Price:    c.transferCost,
				Currency: c.currency,
				Notes:    "Ground transfer to final destination",
			})
			break
		}
	}

	return BookingOption{
		Rank:         rank,
		Strategy:     c.strategy,
		Legs:         legs,
		BaseCost:     math.Round(c.baseCost),
		BagCost:      math.Round(c.bagCost),
		FFSavings:    math.Round(c.ffSavings),
		TransferCost: math.Round(c.transferCost),
		AllInCost:    math.Round(c.allInCost),
		Currency:     c.currency,
		HacksApplied: c.hackTypes,
	}
}

// cheapestFlight returns the flight with the lowest positive price.
func cheapestFlight(flts []models.FlightResult) models.FlightResult {
	best := flts[0]
	for _, f := range flts[1:] {
		if f.Price > 0 && (best.Price <= 0 || f.Price < best.Price) {
			best = f
		}
	}
	return best
}

// convertFFStatuses converts optimizer FFStatus to baggage.FFStatus.
func convertFFStatuses(statuses []FFStatus) []baggage.FFStatus {
	out := make([]baggage.FFStatus, len(statuses))
	for i, s := range statuses {
		out[i] = baggage.FFStatus{
			Alliance: s.Alliance,
			Tier:     s.Tier,
		}
	}
	return out
}
