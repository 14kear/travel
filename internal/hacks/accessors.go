package hacks

// AlternativeOrigin describes a nearby airport or transport hub reachable from
// a given origin, with ground-transit cost and mode information.
type AlternativeOrigin struct {
	IATA     string  // Airport IATA code
	City     string  // City name
	Cost     float64 // Ground transit cost (EUR)
	Minutes  int     // Ground transit time
	Mode     string  // "bus", "train", "ferry", etc.
	HackType string  // "positioning" or "multimodal_positioning"
}

// NearbyAirports returns alternative origin airports for the given IATA code.
// Data comes from the positioning and multimodal positioning maps.
func NearbyAirports(origin string) []AlternativeOrigin {
	var result []AlternativeOrigin

	// From nearbyAirports (positioning.go)
	if entries, ok := nearbyAirports[origin]; ok {
		for _, e := range entries {
			result = append(result, AlternativeOrigin{
				IATA:     e.Code,
				City:     e.City,
				Cost:     e.GroundCost,
				Minutes:  e.GroundMins,
				Mode:     "ground",
				HackType: "positioning",
			})
		}
	}

	// From multiModalHubs (multimodal_positioning.go)
	if hubs, ok := multiModalHubs[origin]; ok {
		for _, h := range hubs {
			// Skip duplicates already added from nearbyAirports.
			dup := false
			for _, r := range result {
				if r.IATA == h.HubCode {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
			result = append(result, AlternativeOrigin{
				IATA:     h.HubCode,
				City:     h.HubCity,
				Cost:     h.StaticGroundEUR,
				Minutes:  0, // not tracked in multiModalHub
				Mode:     h.GroundType,
				HackType: "multimodal_positioning",
			})
		}
	}

	return result
}

// AlternativeDestination describes a cheaper alternative airport near a
// primary destination, with ground-transit details.
type AlternativeDestination struct {
	IATA    string  // Airport IATA code
	City    string  // City name
	Cost    float64 // Ground transit cost (EUR)
	Minutes int     // Transit time to city centre
	Mode    string  // "bus", "train", etc.
}

// DestinationAlternatives returns alternative destination airports for the
// given destination IATA code.
func DestinationAlternatives(dest string) []AlternativeDestination {
	alts, ok := destinationAlternatives[dest]
	if !ok {
		return nil
	}
	result := make([]AlternativeDestination, len(alts))
	for i, a := range alts {
		result[i] = AlternativeDestination{
			IATA:    a.IATA,
			City:    a.City,
			Cost:    a.TransportCost,
			Minutes: a.TransportMin,
			Mode:    a.TransportMode,
		}
	}
	return result
}

// RailFlyStationInfo describes a rail station bookable as a flight origin.
type RailFlyStationInfo struct {
	IATA        string // Rail station IATA code (e.g. "ZWE")
	City        string // City name
	HubIATA     string // Hub airport the train connects to
	Airline     string // IATA carrier code
	AirlineName string // Display name
	FareZone    string // Why it's cheaper
	TrainMins   int    // Approximate train journey time
}

// RailFlyStationsForHub returns rail+fly stations that connect to the given
// hub airport IATA code.
func RailFlyStationsForHub(hubIATA string) []RailFlyStationInfo {
	stations := railFlyStationsForHub(hubIATA)
	result := make([]RailFlyStationInfo, len(stations))
	for i, s := range stations {
		result[i] = RailFlyStationInfo{
			IATA:        s.IATA,
			City:        s.City,
			HubIATA:     s.HubIATA,
			Airline:     s.Airline,
			AirlineName: s.AirlineName,
			FareZone:    s.FareZone,
			TrainMins:   s.TrainMinutes,
		}
	}
	return result
}
