package flights

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MikkoParkkola/trvl/internal/models"
	"golang.org/x/time/rate"
)

const (
	duffelOfferRequestsURL = "https://api.duffel.com/air/offer_requests?return_offers=true&supplier_timeout=5000"
	duffelVersionHeader    = "v2"
)

var (
	duffelLimiter = rate.NewLimiter(rate.Every(500*time.Millisecond), 1)
	duffelClient  = &http.Client{Timeout: 30 * time.Second}
)

type duffelOfferRequestPayload struct {
	Data duffelOfferRequestData `json:"data"`
}

type duffelOfferRequestData struct {
	Slices         []duffelSliceRequest     `json:"slices"`
	Passengers     []duffelPassengerRequest `json:"passengers"`
	CabinClass     string                   `json:"cabin_class,omitempty"`
	MaxConnections *int                     `json:"max_connections,omitempty"`
}

type duffelSliceRequest struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
}

type duffelPassengerRequest struct {
	Type string `json:"type"`
}

type duffelOfferRequestResponse struct {
	Data duffelOfferRequest `json:"data"`
}

type duffelOfferRequest struct {
	Offers []duffelOffer `json:"offers"`
}

type duffelOffer struct {
	TotalAmount      string        `json:"total_amount"`
	TotalCurrency    string        `json:"total_currency"`
	TotalEmissionsKG string        `json:"total_emissions_kg"`
	Slices           []duffelSlice `json:"slices"`
}

type duffelSlice struct {
	Duration      string          `json:"duration"`
	FareBrandName string          `json:"fare_brand_name"`
	Segments      []duffelSegment `json:"segments"`
}

type duffelSegment struct {
	Origin                       duffelPlace              `json:"origin"`
	Destination                  duffelPlace              `json:"destination"`
	DepartingAt                  string                   `json:"departing_at"`
	ArrivingAt                   string                   `json:"arriving_at"`
	Duration                     string                   `json:"duration"`
	Aircraft                     *duffelAircraft          `json:"aircraft"`
	OperatingCarrier             *duffelCarrier           `json:"operating_carrier"`
	MarketingCarrier             *duffelCarrier           `json:"marketing_carrier"`
	OperatingCarrierFlightNumber string                   `json:"operating_carrier_flight_number"`
	MarketingCarrierFlightNumber string                   `json:"marketing_carrier_flight_number"`
	Passengers                   []duffelSegmentPassenger `json:"passengers"`
}

type duffelPlace struct {
	IATACode string `json:"iata_code"`
	Name     string `json:"name"`
	CityName string `json:"city_name"`
}

type duffelAircraft struct {
	IATACode string `json:"iata_code"`
	Name     string `json:"name"`
}

type duffelCarrier struct {
	IATACode string `json:"iata_code"`
	Name     string `json:"name"`
}

type duffelSegmentPassenger struct {
	Baggages []duffelBaggage `json:"baggages"`
}

type duffelBaggage struct {
	Type     string `json:"type"`
	Quantity int    `json:"quantity"`
}

func SearchDuffelFlights(ctx context.Context, origin, destination, date string, opts SearchOptions) ([]models.FlightResult, error) {
	token := loadDuffelToken()
	if token == "" {
		return nil, nil
	}

	if err := duffelLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("duffel: rate limiter: %w", err)
	}

	payload := duffelOfferRequestPayload{
		Data: duffelOfferRequestData{
			Slices:     duffelRequestSlices(origin, destination, date, opts.ReturnDate),
			Passengers: duffelPassengers(opts.Adults),
		},
	}
	if cabinClass := duffelCabinClass(opts.CabinClass); cabinClass != "" {
		payload.Data.CabinClass = cabinClass
	}
	if maxConnections, ok := duffelMaxConnections(opts.MaxStops); ok {
		payload.Data.MaxConnections = &maxConnections
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("duffel: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, duffelOfferRequestsURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("duffel: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Duffel-Version", duffelVersionHeader)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := duffelClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("duffel: HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("duffel: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("duffel: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded duffelOfferRequestResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("duffel: decode response: %w", err)
	}

	results := make([]models.FlightResult, 0, len(decoded.Data.Offers))
	for _, offer := range decoded.Data.Offers {
		if !duffelOfferMatchesOptions(offer, opts) {
			continue
		}
		result, ok := mapDuffelOffer(offer)
		if !ok {
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

func duffelRequestSlices(origin, destination, date, returnDate string) []duffelSliceRequest {
	slices := []duffelSliceRequest{{
		Origin:        origin,
		Destination:   destination,
		DepartureDate: date,
	}}
	if returnDate != "" {
		slices = append(slices, duffelSliceRequest{
			Origin:        destination,
			Destination:   origin,
			DepartureDate: returnDate,
		})
	}
	return slices
}

func duffelPassengers(adults int) []duffelPassengerRequest {
	if adults <= 0 {
		adults = 1
	}

	passengers := make([]duffelPassengerRequest, adults)
	for i := range passengers {
		passengers[i] = duffelPassengerRequest{Type: "adult"}
	}
	return passengers
}

func duffelCabinClass(cabin models.CabinClass) string {
	switch cabin {
	case models.Business:
		return "business"
	case models.First:
		return "first"
	case models.PremiumEconomy:
		return "premium_economy"
	case models.Economy, 0:
		return "economy"
	default:
		return ""
	}
}

func duffelMaxConnections(maxStops models.MaxStops) (int, bool) {
	switch maxStops {
	case models.NonStop:
		return 0, true
	case models.OneStop:
		return 1, true
	case models.TwoPlusStops:
		return 2, true
	default:
		return 0, false
	}
}

func duffelOfferMatchesOptions(offer duffelOffer, opts SearchOptions) bool {
	if !opts.ExcludeBasic {
		return true
	}

	for _, slice := range offer.Slices {
		if strings.Contains(strings.ToLower(strings.TrimSpace(slice.FareBrandName)), "basic") {
			return false
		}
	}
	return true
}

func mapDuffelOffer(offer duffelOffer) (models.FlightResult, bool) {
	price, err := strconv.ParseFloat(offer.TotalAmount, 64)
	if err != nil || price <= 0 {
		return models.FlightResult{}, false
	}

	legs := make([]models.FlightLeg, 0)
	totalDuration := 0
	totalStops := 0
	for _, slice := range offer.Slices {
		sliceLegs := mapDuffelSliceLegs(slice)
		if len(sliceLegs) == 0 {
			continue
		}
		totalDuration += duffelSliceDurationMinutes(slice, sliceLegs)
		totalStops += max(len(sliceLegs)-1, 0)
		legs = append(legs, sliceLegs...)
	}
	if len(legs) == 0 {
		return models.FlightResult{}, false
	}

	carryOnIncluded, checkedBagsIncluded := duffelBaggageAllowance(offer)

	return models.FlightResult{
		Price:               price,
		Currency:            strings.TrimSpace(offer.TotalCurrency),
		Duration:            totalDuration,
		Stops:               totalStops,
		Provider:            "duffel",
		Legs:                legs,
		CarryOnIncluded:     carryOnIncluded,
		CheckedBagsIncluded: checkedBagsIncluded,
		Emissions:           duffelEmissionsGrams(offer.TotalEmissionsKG),
	}, true
}

func mapDuffelSliceLegs(slice duffelSlice) []models.FlightLeg {
	if len(slice.Segments) == 0 {
		return nil
	}

	legs := make([]models.FlightLeg, 0, len(slice.Segments))
	for _, segment := range slice.Segments {
		carrier := segment.MarketingCarrier
		flightNumber := strings.TrimSpace(segment.MarketingCarrierFlightNumber)
		if carrier == nil {
			carrier = segment.OperatingCarrier
			flightNumber = strings.TrimSpace(segment.OperatingCarrierFlightNumber)
		}

		airlineCode := ""
		airlineName := ""
		if carrier != nil {
			airlineCode = strings.TrimSpace(carrier.IATACode)
			airlineName = strings.TrimSpace(carrier.Name)
		}

		legs = append(legs, models.FlightLeg{
			DepartureAirport: models.AirportInfo{
				Code: strings.TrimSpace(segment.Origin.IATACode),
				Name: firstNonEmpty(strings.TrimSpace(segment.Origin.Name), strings.TrimSpace(segment.Origin.CityName), strings.TrimSpace(segment.Origin.IATACode)),
			},
			ArrivalAirport: models.AirportInfo{
				Code: strings.TrimSpace(segment.Destination.IATACode),
				Name: firstNonEmpty(strings.TrimSpace(segment.Destination.Name), strings.TrimSpace(segment.Destination.CityName), strings.TrimSpace(segment.Destination.IATACode)),
			},
			DepartureTime: duffelDisplayTime(segment.DepartingAt),
			ArrivalTime:   duffelDisplayTime(segment.ArrivingAt),
			Duration:      duffelDurationMinutes(segment.Duration),
			Airline:       airlineName,
			AirlineCode:   airlineCode,
			FlightNumber:  duffelFlightNumber(airlineCode, flightNumber),
			Aircraft:      duffelAircraftName(segment.Aircraft),
		})
	}

	computeLayovers(legs)
	return legs
}

func duffelSliceDurationMinutes(slice duffelSlice, legs []models.FlightLeg) int {
	if minutes := duffelDurationMinutes(slice.Duration); minutes > 0 {
		return minutes
	}

	total := 0
	for _, leg := range legs {
		total += leg.Duration
		if leg.LayoverMinutes > 0 {
			total += leg.LayoverMinutes
		}
	}
	return total
}

func duffelDisplayTime(raw string) string {
	t, ok := parseFlexibleFlightTime(raw)
	if !ok {
		return ""
	}
	return t.Format(flightTimeLayout)
}

func duffelDurationMinutes(raw string) int {
	raw = strings.TrimSpace(strings.ToUpper(raw))
	if raw == "" || !strings.HasPrefix(raw, "P") {
		return 0
	}

	total := 0
	seenTimeSection := false
	value := 0
	for _, ch := range raw[1:] {
		switch {
		case ch >= '0' && ch <= '9':
			value = value*10 + int(ch-'0')
		case ch == 'T':
			seenTimeSection = true
		case ch == 'D':
			total += value * 24 * 60
			value = 0
		case ch == 'H':
			total += value * 60
			value = 0
		case ch == 'M':
			if seenTimeSection {
				total += value
			}
			value = 0
		case ch == 'S':
			if value > 0 {
				total += int(math.Ceil(float64(value) / 60.0))
			}
			value = 0
		default:
			return 0
		}
	}

	return total
}

func duffelFlightNumber(airlineCode, number string) string {
	airlineCode = strings.TrimSpace(airlineCode)
	number = strings.TrimSpace(number)
	switch {
	case airlineCode == "" && number == "":
		return ""
	case airlineCode == "":
		return number
	case number == "":
		return airlineCode
	default:
		return airlineCode + " " + number
	}
}

func duffelAircraftName(aircraft *duffelAircraft) string {
	if aircraft == nil {
		return ""
	}
	return strings.TrimSpace(firstNonEmpty(aircraft.Name, aircraft.IATACode))
}

func duffelEmissionsGrams(raw string) int {
	kg, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || kg <= 0 {
		return 0
	}
	return int(math.Round(kg * 1000))
}

func duffelBaggageAllowance(offer duffelOffer) (*bool, *int) {
	carryKnown := false
	checkedKnown := false
	carryMin := math.MaxInt
	checkedMin := math.MaxInt

	for _, slice := range offer.Slices {
		for _, segment := range slice.Segments {
			for _, passenger := range segment.Passengers {
				if len(passenger.Baggages) == 0 {
					continue
				}

				carryQty := 0
				checkedQty := 0
				for _, baggage := range passenger.Baggages {
					switch strings.ToLower(strings.TrimSpace(baggage.Type)) {
					case "carry_on":
						carryQty += max(baggage.Quantity, 0)
					case "checked":
						checkedQty += max(baggage.Quantity, 0)
					}
				}

				carryKnown = true
				checkedKnown = true
				carryMin = min(carryMin, carryQty)
				checkedMin = min(checkedMin, checkedQty)
			}
		}
	}

	var carryOnIncluded *bool
	if carryKnown && carryMin != math.MaxInt {
		included := carryMin > 0
		carryOnIncluded = &included
	}

	var checkedBagsIncluded *int
	if checkedKnown && checkedMin != math.MaxInt {
		included := checkedMin
		checkedBagsIncluded = &included
	}

	return carryOnIncluded, checkedBagsIncluded
}
