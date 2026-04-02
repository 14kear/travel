package mcp

import "fmt"

// registerResources adds all resource definitions to the server.
func registerResources(s *Server) {
	s.resources = []ResourceDef{
		{
			URI:         "trvl://airports/popular",
			Name:        "Popular Airports",
			Description: "List of 50 popular airport codes with city names",
			MimeType:    "text/plain",
		},
		{
			URI:         "trvl://help/flights",
			Name:        "Flight Search Guide",
			Description: "Flight search usage guide with examples",
			MimeType:    "text/markdown",
		},
		{
			URI:         "trvl://help/hotels",
			Name:        "Hotel Search Guide",
			Description: "Hotel search usage guide with examples",
			MimeType:    "text/markdown",
		},
	}
}

// readResource returns the content for a resource URI.
func readResource(uri string) (*ResourcesReadResult, error) {
	switch uri {
	case "trvl://airports/popular":
		return &ResourcesReadResult{
			Contents: []ResourceContent{{
				URI:      uri,
				MimeType: "text/plain",
				Text:     popularAirports,
			}},
		}, nil
	case "trvl://help/flights":
		return &ResourcesReadResult{
			Contents: []ResourceContent{{
				URI:      uri,
				MimeType: "text/markdown",
				Text:     flightSearchGuide,
			}},
		}, nil
	case "trvl://help/hotels":
		return &ResourcesReadResult{
			Contents: []ResourceContent{{
				URI:      uri,
				MimeType: "text/markdown",
				Text:     hotelSearchGuide,
			}},
		}, nil
	default:
		return nil, fmt.Errorf("resource not found: %s", uri)
	}
}

const popularAirports = `HEL - Helsinki, Finland
JFK - New York (John F. Kennedy), USA
LHR - London Heathrow, UK
NRT - Tokyo Narita, Japan
CDG - Paris Charles de Gaulle, France
LAX - Los Angeles, USA
SIN - Singapore Changi, Singapore
DXB - Dubai, UAE
FRA - Frankfurt, Germany
AMS - Amsterdam Schiphol, Netherlands
HND - Tokyo Haneda, Japan
ICN - Seoul Incheon, South Korea
SFO - San Francisco, USA
ORD - Chicago O'Hare, USA
BKK - Bangkok Suvarnabhumi, Thailand
IST - Istanbul, Turkey
MUC - Munich, Germany
FCO - Rome Fiumicino, Italy
MAD - Madrid Barajas, Spain
BCN - Barcelona El Prat, Spain
ZRH - Zurich, Switzerland
HKG - Hong Kong, China
SYD - Sydney Kingsford Smith, Australia
MIA - Miami, USA
EWR - Newark Liberty, USA
ARN - Stockholm Arlanda, Sweden
OSL - Oslo Gardermoen, Norway
CPH - Copenhagen, Denmark
LIS - Lisbon, Portugal
VIE - Vienna, Austria
ATL - Atlanta Hartsfield-Jackson, USA
DEN - Denver, USA
SEA - Seattle-Tacoma, USA
BOS - Boston Logan, USA
DOH - Doha Hamad, Qatar
DEL - Delhi Indira Gandhi, India
BOM - Mumbai, India
PEK - Beijing Capital, China
PVG - Shanghai Pudong, China
KUL - Kuala Lumpur, Malaysia
MEX - Mexico City, Mexico
GRU - Sao Paulo Guarulhos, Brazil
JNB - Johannesburg, South Africa
CAI - Cairo, Egypt
DUS - Dusseldorf, Germany
HAM - Hamburg, Germany
MXP - Milan Malpensa, Italy
PMI - Palma de Mallorca, Spain
TLV - Tel Aviv Ben Gurion, Israel
AKL - Auckland, New Zealand`

const flightSearchGuide = `# Flight Search Guide

## search_flights

Search for flights on a specific date. Returns real-time pricing from Google Flights.

### Required Parameters
- **origin**: IATA airport code (e.g., HEL, JFK, NRT)
- **destination**: IATA airport code
- **departure_date**: Date in YYYY-MM-DD format

### Optional Parameters
- **return_date**: For round-trip searches (YYYY-MM-DD)
- **cabin_class**: economy (default), premium_economy, business, first
- **max_stops**: any (default), nonstop, one_stop, two_plus
- **sort_by**: cheapest (default), duration, departure, arrival

### Examples

**One-way flight:**
` + "```json" + `
{"origin": "HEL", "destination": "NRT", "departure_date": "2026-06-15"}
` + "```" + `

**Round-trip flight:**
` + "```json" + `
{"origin": "HEL", "destination": "NRT", "departure_date": "2026-06-15", "return_date": "2026-06-22"}
` + "```" + `

**Nonstop business class:**
` + "```json" + `
{"origin": "JFK", "destination": "LHR", "departure_date": "2026-06-15", "cabin_class": "business", "max_stops": "nonstop"}
` + "```" + `

## search_dates

Find the cheapest flight prices across a date range.

### Required Parameters
- **origin**: IATA airport code
- **destination**: IATA airport code
- **start_date**: Start of range (YYYY-MM-DD)
- **end_date**: End of range (YYYY-MM-DD)

### Optional Parameters
- **trip_duration**: Days for round-trip (e.g., 7)
- **is_round_trip**: true/false (default: false)

### Examples

**Cheapest one-way dates in June:**
` + "```json" + `
{"origin": "HEL", "destination": "NRT", "start_date": "2026-06-01", "end_date": "2026-06-30"}
` + "```" + `

**Cheapest round-trip week in June:**
` + "```json" + `
{"origin": "HEL", "destination": "NRT", "start_date": "2026-06-01", "end_date": "2026-06-30", "is_round_trip": true, "trip_duration": 7}
` + "```" + `

## Tips

- Use **search_dates** first to find the cheapest day, then **search_flights** for full details on that day
- Airport codes are case-insensitive (hel = HEL)
- Prices are real-time from Google Flights and may change
- Round-trip searches often show lower per-leg prices than one-way
`

const hotelSearchGuide = `# Hotel Search Guide

## search_hotels

Search for hotels in a location. Returns real-time pricing from Google Hotels.

### Required Parameters
- **location**: City name, neighborhood, or address (e.g., "Helsinki", "Shibuya Tokyo", "Manhattan New York")
- **check_in**: Check-in date (YYYY-MM-DD)
- **check_out**: Check-out date (YYYY-MM-DD)

### Optional Parameters
- **guests**: Number of guests (default: 2)
- **stars**: Minimum star rating 1-5 (default: no filter)
- **sort**: relevance (default), price, rating, distance

### Examples

**Basic hotel search:**
` + "```json" + `
{"location": "Helsinki", "check_in": "2026-06-15", "check_out": "2026-06-18"}
` + "```" + `

**4+ star hotels sorted by price:**
` + "```json" + `
{"location": "Tokyo Shinjuku", "check_in": "2026-06-15", "check_out": "2026-06-22", "stars": 4, "sort": "price"}
` + "```" + `

## hotel_prices

Compare prices across booking providers for a specific hotel.

### Required Parameters
- **hotel_id**: Google Hotels property ID (from search_hotels results)
- **check_in**: Check-in date (YYYY-MM-DD)
- **check_out**: Check-out date (YYYY-MM-DD)

### Example

` + "```json" + `
{"hotel_id": "/g/11b6d4_v_4", "check_in": "2026-06-15", "check_out": "2026-06-18"}
` + "```" + `

## Tips

- Use **search_hotels** to find options, then **hotel_prices** to compare booking providers for the best deal
- Location can be specific ("Shibuya Tokyo") or general ("Tokyo")
- Prices shown are per night
- The hotel_id from search results is needed for price comparison
- Star ratings and guest ratings are different: stars = hotel class, rating = guest reviews out of 5
`
