package optimizer

import (
	"testing"

	"github.com/MikkoParkkola/trvl/internal/models"
)

// ---------------------------------------------------------------------------
// shiftDate — covering the empty-input and bad-parse paths
// ---------------------------------------------------------------------------

func TestShiftDate_EmptyInput(t *testing.T) {
	got := shiftDate("", 5)
	if got != "" {
		t.Errorf("shiftDate empty input: got %q, want empty", got)
	}
}

func TestShiftDate_InvalidDate(t *testing.T) {
	got := shiftDate("not-a-date", 1)
	if got != "" {
		t.Errorf("shiftDate invalid date: got %q, want empty", got)
	}
}

func TestShiftDate_PositiveShift(t *testing.T) {
	got := shiftDate("2026-06-15", 3)
	if got != "2026-06-18" {
		t.Errorf("shiftDate +3: got %q, want 2026-06-18", got)
	}
}

func TestShiftDate_NegativeShift(t *testing.T) {
	got := shiftDate("2026-06-15", -5)
	if got != "2026-06-10" {
		t.Errorf("shiftDate -5: got %q, want 2026-06-10", got)
	}
}

func TestShiftDate_CrossMonthBoundary(t *testing.T) {
	got := shiftDate("2026-01-30", 3)
	if got != "2026-02-02" {
		t.Errorf("shiftDate cross month: got %q, want 2026-02-02", got)
	}
}

// ---------------------------------------------------------------------------
// defaults — covering the negative FlexDays path
// ---------------------------------------------------------------------------

func TestDefaults_NegativeFlexDays(t *testing.T) {
	in := OptimizeInput{
		FlexDays: -1,
	}
	in.defaults()
	// Negative → set to 0, then 0 → set to 3 (default).
	// Actually: FlexDays < 0 → set to 0, then FlexDays == 0 → set to 3.
	if in.FlexDays != 3 {
		t.Errorf("expected FlexDays=3 after negative input, got %d", in.FlexDays)
	}
}

func TestDefaults_NegativeGuests(t *testing.T) {
	in := OptimizeInput{
		Guests: -5,
	}
	in.defaults()
	if in.Guests != 1 {
		t.Errorf("expected Guests=1 after negative input, got %d", in.Guests)
	}
}

func TestDefaults_NegativeMaxResults(t *testing.T) {
	in := OptimizeInput{
		MaxResults: -10,
	}
	in.defaults()
	if in.MaxResults != 5 {
		t.Errorf("expected MaxResults=5 after negative input, got %d", in.MaxResults)
	}
}

func TestDefaults_NegativeMaxAPICalls(t *testing.T) {
	in := OptimizeInput{
		MaxAPICalls: -1,
	}
	in.defaults()
	if in.MaxAPICalls != 15 {
		t.Errorf("expected MaxAPICalls=15 after negative input, got %d", in.MaxAPICalls)
	}
}

// ---------------------------------------------------------------------------
// expandCandidates — hidden city candidates
// ---------------------------------------------------------------------------

func TestExpandCandidates_HiddenCity_AMS(t *testing.T) {
	input := OptimizeInput{
		Origin:      "LHR",
		Destination: "AMS",
		DepartDate:  "2026-06-01",
		FlexDays:    -1, // disable flex
	}
	input.defaults()
	candidates := expandCandidates(input)

	// AMS is a hidden city hub destination. Candidates for hidden city should
	// search to HEL, RIX, TLL, ARN (beyond cities) but skip LHR (origin).
	var hiddenCityCandidates []*candidate
	for _, c := range candidates {
		for _, h := range c.hackTypes {
			if h == "hidden_city" {
				hiddenCityCandidates = append(hiddenCityCandidates, c)
				break
			}
		}
	}

	if len(hiddenCityCandidates) == 0 {
		t.Fatal("expected hidden city candidates for LHR→AMS")
	}

	// None should have origin == dest.
	for _, c := range hiddenCityCandidates {
		if c.origin == c.dest {
			t.Errorf("hidden city candidate has same origin and dest: %s", c.origin)
		}
		// None should fly from LHR to LHR (beyond == origin check).
		if c.dest == "LHR" {
			t.Errorf("hidden city beyond should not equal origin LHR")
		}
	}
}

func TestExpandCandidates_NoHiddenCity_NonHub(t *testing.T) {
	input := OptimizeInput{
		Origin:      "HEL",
		Destination: "AGP", // Malaga, not a hub
		DepartDate:  "2026-06-01",
		FlexDays:    -1,
	}
	input.defaults()
	candidates := expandCandidates(input)

	for _, c := range candidates {
		for _, h := range c.hackTypes {
			if h == "hidden_city" {
				t.Errorf("unexpected hidden_city candidate for non-hub destination AGP: %+v", c)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// expandCandidates — date flex generates the right number of candidates
// ---------------------------------------------------------------------------

func TestExpandCandidates_DateFlexCount(t *testing.T) {
	input := OptimizeInput{
		Origin:      "XYZ",
		Destination: "QQQ",
		DepartDate:  "2026-06-15",
		FlexDays:    2,
	}
	input.defaults()
	candidates := expandCandidates(input)

	// FlexDays=2 should generate candidates for d=-2,-1,+1,+2 = 4 date-flex
	// candidates plus 1 baseline = 5 total (for unknown airports).
	dateFlexCount := 0
	for _, c := range candidates {
		for _, h := range c.hackTypes {
			if h == "date_flex" {
				dateFlexCount++
				break
			}
		}
	}
	if dateFlexCount != 4 {
		t.Errorf("expected 4 date-flex candidates for FlexDays=2, got %d", dateFlexCount)
	}
}

// ---------------------------------------------------------------------------
// priceCandidate — all zero/negative prices
// ---------------------------------------------------------------------------

func TestPriceCandidate_AllZeroPrices(t *testing.T) {
	c := &candidate{
		searched: true,
		flights: []models.FlightResult{
			{Price: 0, Currency: "EUR"},
			{Price: 0, Currency: "EUR"},
		},
	}
	input := OptimizeInput{}
	input.defaults()
	priceCandidate(c, input)

	if c.allInCost != 0 {
		t.Errorf("expected allInCost=0 for all-zero prices, got %f", c.allInCost)
	}
}

func TestPriceCandidate_NegativePrice(t *testing.T) {
	c := &candidate{
		searched: true,
		flights: []models.FlightResult{
			{Price: -10, Currency: "EUR"},
			{Price: 100, Currency: "EUR"},
		},
	}
	input := OptimizeInput{}
	input.defaults()
	priceCandidate(c, input)

	// cheapestFlight prefers positive prices.
	if c.baseCost != 100 {
		t.Errorf("expected baseCost=100, got %f", c.baseCost)
	}
}

// ---------------------------------------------------------------------------
// priceCandidate — FF savings
// ---------------------------------------------------------------------------

func TestPriceCandidate_WithFFStatus(t *testing.T) {
	c := &candidate{
		searched: true,
		flights: []models.FlightResult{
			{Price: 200, Currency: "EUR", Legs: []models.FlightLeg{
				{AirlineCode: "KL"},
			}},
		},
	}
	input := OptimizeInput{
		FFStatuses:     []FFStatus{{Alliance: "skyteam", Tier: "gold"}},
		NeedCheckedBag: true,
	}
	input.defaults()
	priceCandidate(c, input)

	// With FF status, ffSavings should be >= 0 (depends on baggage module).
	if c.ffSavings < 0 {
		t.Errorf("expected ffSavings >= 0, got %f", c.ffSavings)
	}
}

// ---------------------------------------------------------------------------
// candidateToOption — destination airport transfer leg
// ---------------------------------------------------------------------------

func TestCandidateToOption_DestinationAirportLeg(t *testing.T) {
	c := &candidate{
		origin:       "HEL",
		dest:         "GRO",
		departDate:   "2026-06-01",
		strategy:     "Fly to Girona + bus to BCN",
		hackTypes:    []string{"destination_airport"},
		transferCost: 15,
		baseCost:     100,
		currency:     "EUR",
		allInCost:    115,
		flights: []models.FlightResult{
			{Price: 100, Currency: "EUR", Duration: 180},
		},
	}

	input := OptimizeInput{Origin: "HEL", Destination: "BCN"}
	opt := candidateToOption(c, 1, input)

	// Should have: flight leg + destination transfer leg.
	// No positioning leg because origin == input.Origin.
	foundDestTransfer := false
	for _, leg := range opt.Legs {
		if leg.Type == "ground" && leg.From == "GRO" && leg.To == "BCN" {
			foundDestTransfer = true
			if leg.Notes != "Ground transfer to final destination" {
				t.Errorf("dest transfer notes: %q", leg.Notes)
			}
		}
	}
	if !foundDestTransfer {
		t.Error("expected destination transfer leg from GRO to BCN")
	}
}

func TestCandidateToOption_NoLegsEmptyFlights(t *testing.T) {
	c := &candidate{
		origin:     "HEL",
		dest:       "BCN",
		departDate: "2026-06-01",
		strategy:   "Direct booking",
		baseCost:   0,
		currency:   "EUR",
		allInCost:  0,
		flights:    nil, // no flights
	}

	input := OptimizeInput{Origin: "HEL", Destination: "BCN"}
	opt := candidateToOption(c, 1, input)

	// No transfer cost, no flights, no pre-priced: should have 0 legs.
	if len(opt.Legs) != 0 {
		t.Errorf("expected 0 legs for empty candidate, got %d", len(opt.Legs))
	}
}

func TestCandidateToOption_FlightWithAirline(t *testing.T) {
	c := &candidate{
		origin:     "HEL",
		dest:       "BCN",
		departDate: "2026-06-01",
		strategy:   "Direct booking",
		baseCost:   150,
		currency:   "EUR",
		allInCost:  150,
		flights: []models.FlightResult{
			{Price: 150, Currency: "EUR", Duration: 240, Legs: []models.FlightLeg{
				{Airline: "Finnair", AirlineCode: "AY"},
			}},
		},
	}

	input := OptimizeInput{Origin: "HEL", Destination: "BCN"}
	opt := candidateToOption(c, 1, input)

	if len(opt.Legs) < 1 {
		t.Fatal("expected at least 1 leg")
	}
	if opt.Legs[0].Airline != "Finnair" {
		t.Errorf("airline: got %q, want Finnair", opt.Legs[0].Airline)
	}
	if opt.Legs[0].Duration != 240 {
		t.Errorf("duration: got %d, want 240", opt.Legs[0].Duration)
	}
}

// ---------------------------------------------------------------------------
// rankCandidates — edge case: more candidates than MaxResults
// ---------------------------------------------------------------------------

func TestRankCandidates_MaxResultsLargerThanPriced(t *testing.T) {
	candidates := []*candidate{
		{searched: true, allInCost: 100, baseCost: 100, currency: "EUR", strategy: "A"},
	}

	input := OptimizeInput{MaxResults: 10}
	result := rankCandidates(candidates, input)

	if !result.Success {
		t.Fatal("expected success")
	}
	if len(result.Options) != 1 {
		t.Errorf("expected 1 option (fewer than MaxResults), got %d", len(result.Options))
	}
}

// ---------------------------------------------------------------------------
// rankCandidates — baseline is the first no-hack candidate
// ---------------------------------------------------------------------------

func TestRankCandidates_BaselineIsNoHack(t *testing.T) {
	candidates := []*candidate{
		{searched: true, allInCost: 150, baseCost: 150, currency: "EUR", strategy: "Alt", hackTypes: []string{"positioning"}},
		{searched: true, allInCost: 200, baseCost: 200, currency: "EUR", strategy: "Direct"},
		{searched: true, allInCost: 100, baseCost: 100, currency: "EUR", strategy: "Alt2", hackTypes: []string{"date_flex"}},
	}

	input := OptimizeInput{MaxResults: 5, Currency: "EUR"}
	result := rankCandidates(candidates, input)

	if result.Baseline == nil {
		t.Fatal("expected baseline to be set")
	}
	if result.Baseline.AllInCost != 200 {
		t.Errorf("baseline should be the direct/no-hack candidate (200), got %f", result.Baseline.AllInCost)
	}
}

// ---------------------------------------------------------------------------
// convertFFStatuses — empty input
// ---------------------------------------------------------------------------

func TestConvertFFStatuses_Empty(t *testing.T) {
	converted := convertFFStatuses(nil)
	if len(converted) != 0 {
		t.Errorf("expected 0 converted statuses for nil input, got %d", len(converted))
	}
}

// ---------------------------------------------------------------------------
// cheapestFlight — single flight
// ---------------------------------------------------------------------------

func TestCheapestFlight_SingleFlight(t *testing.T) {
	flights := []models.FlightResult{
		{Price: 99, Currency: "EUR"},
	}
	best := cheapestFlight(flights)
	if best.Price != 99 {
		t.Errorf("cheapestFlight single: want 99, got %f", best.Price)
	}
}

func TestCheapestFlight_AllZero(t *testing.T) {
	flights := []models.FlightResult{
		{Price: 0, Currency: "EUR"},
		{Price: 0, Currency: "EUR"},
	}
	best := cheapestFlight(flights)
	// When all are 0, the first is returned (no positive price found).
	if best.Price != 0 {
		t.Errorf("cheapestFlight all-zero: want 0, got %f", best.Price)
	}
}

func TestCheapestFlight_FirstZeroSecondPositive(t *testing.T) {
	flights := []models.FlightResult{
		{Price: 0, Currency: "EUR"},
		{Price: 50, Currency: "EUR"},
	}
	best := cheapestFlight(flights)
	if best.Price != 50 {
		t.Errorf("cheapestFlight first-zero: want 50, got %f", best.Price)
	}
}

// ---------------------------------------------------------------------------
// expandCandidates — round-trip date flex with return date
// ---------------------------------------------------------------------------

func TestExpandCandidates_DateFlexWithReturnDate(t *testing.T) {
	input := OptimizeInput{
		Origin:      "XYZ",
		Destination: "QQQ",
		DepartDate:  "2026-06-15",
		ReturnDate:  "2026-06-22",
		FlexDays:    1,
	}
	input.defaults()
	candidates := expandCandidates(input)

	var dateFlexCandidates []*candidate
	for _, c := range candidates {
		for _, h := range c.hackTypes {
			if h == "date_flex" {
				dateFlexCandidates = append(dateFlexCandidates, c)
				break
			}
		}
	}

	// FlexDays=1: d=-1, d=+1 = 2 date-flex candidates.
	if len(dateFlexCandidates) != 2 {
		t.Fatalf("expected 2 date-flex candidates for FlexDays=1 RT, got %d", len(dateFlexCandidates))
	}

	// Both departure and return should be shifted.
	for _, c := range dateFlexCandidates {
		if c.returnDate == "" {
			t.Error("date-flex candidate for RT should have returnDate set")
		}
		if c.returnDate == "2026-06-22" {
			t.Error("date-flex candidate returnDate should differ from original")
		}
	}
}
