package hacks

import (
	"context"
	"fmt"
	"strings"
)

// regionalPass describes a flat-rate transport pass in a European country.
type regionalPass struct {
	Name     string
	Country  string
	PriceEUR float64
	Period   string   // "monthly", "annual", "day"
	Coverage string   // what it covers
	ValidFor []string // ISO 3166-1 alpha-2 country codes
	Notes    string
}

// regionalPasses is a curated list of European flat-rate transport passes.
var regionalPasses = []regionalPass{
	{Name: "Deutschlandticket", Country: "DE", PriceEUR: 49, Period: "monthly",
		Coverage: "All regional trains, buses, trams in Germany",
		ValidFor: []string{"DE"}, Notes: "Not valid on ICE/IC long-distance trains"},
	{Name: "Klimaticket", Country: "AT", PriceEUR: 1095.0 / 12, Period: "monthly",
		Coverage: "All public transport in Austria (trains, buses, trams)",
		ValidFor: []string{"AT"}, Notes: "Annual pass, EUR 1095/year = EUR 91/month"},
	{Name: "Swiss Half Fare Card", Country: "CH", PriceEUR: 170, Period: "annual",
		Coverage: "50% off all Swiss trains, buses, boats, cable cars",
		ValidFor: []string{"CH"}, Notes: "Pays for itself in 1-2 long-distance journeys"},
	{Name: "OV-chipkaart + Dal Vrij", Country: "NL", PriceEUR: 5, Period: "monthly",
		Coverage: "40% off Dutch trains during off-peak hours",
		ValidFor: []string{"NL"}, Notes: "EUR 5/month for off-peak discount"},
	{Name: "OBB Vorteilscard", Country: "AT", PriceEUR: 66, Period: "annual",
		Coverage: "50% off OBB trains, including cross-border to Munich/Zurich",
		ValidFor: []string{"AT", "DE", "CH"}, Notes: "Pays for itself in 1 Vienna-Munich trip"},
	{Name: "BahnCard 25", Country: "DE", PriceEUR: 62, Period: "annual",
		Coverage: "25% off all DB flex fares",
		ValidFor: []string{"DE"}, Notes: "Introductory price for first year"},
	{Name: "BahnCard 50", Country: "DE", PriceEUR: 244, Period: "annual",
		Coverage: "50% off all DB flex fares, 25% off advance fares",
		ValidFor: []string{"DE"}, Notes: "Pays for itself in 3-4 long-distance trips"},
}

// iataToCountry maps IATA airport codes to ISO 3166-1 alpha-2 country codes.
// Covers major European airports relevant to regional pass suggestions.
var iataToCountry = map[string]string{
	// Germany
	"BER": "DE", "FRA": "DE", "MUC": "DE", "DUS": "DE", "HAM": "DE",
	"STR": "DE", "CGN": "DE", "HHN": "DE", "FMM": "DE", "NUE": "DE",
	"HAJ": "DE", "LEJ": "DE", "DTM": "DE", "PAD": "DE", "BRE": "DE",
	// Austria
	"VIE": "AT", "INN": "AT", "SZG": "AT", "GRZ": "AT", "LNZ": "AT", "KLU": "AT",
	// Switzerland
	"ZRH": "CH", "GVA": "CH", "BSL": "CH", "BRN": "CH",
	// Netherlands
	"AMS": "NL", "EIN": "NL", "RTM": "NL", "MST": "NL",
	// Belgium
	"BRU": "BE", "CRL": "BE", "ANR": "BE", "LGG": "BE",
	// France
	"CDG": "FR", "ORY": "FR", "BVA": "FR", "LYS": "FR", "NCE": "FR",
	"MRS": "FR", "TLS": "FR", "NTE": "FR", "BOD": "FR",
	// Italy
	"FCO": "IT", "MXP": "IT", "LIN": "IT", "BGY": "IT", "VCE": "IT",
	"NAP": "IT", "CIA": "IT", "BLQ": "IT", "PSA": "IT", "TRN": "IT",
	"TSF": "IT",
	// Spain
	"MAD": "ES", "BCN": "ES", "AGP": "ES", "PMI": "ES", "IBZ": "ES",
	"MAH": "ES", "ALC": "ES", "SVQ": "ES", "VLC": "ES", "BIO": "ES",
	"TFS": "ES", "LPA": "ES", "ACE": "ES", "FUE": "ES", "GRO": "ES",
	// UK
	"LHR": "GB", "LGW": "GB", "STN": "GB", "LTN": "GB", "MAN": "GB",
	"EDI": "GB", "BHX": "GB", "BRS": "GB", "GLA": "GB",
	// Nordics
	"HEL": "FI", "TMP": "FI", "TKU": "FI", "OUL": "FI", "RVN": "FI",
	"ARN": "SE", "GOT": "SE", "NYO": "SE", "MMX": "SE",
	"CPH": "DK", "AAL": "DK", "BLL": "DK",
	"OSL": "NO", "BGO": "NO", "TRD": "NO", "SVG": "NO", "TRF": "NO",
	// Eastern Europe
	"PRG": "CZ", "BRQ": "CZ",
	"WAW": "PL", "KRK": "PL", "GDN": "PL", "WRO": "PL", "KTW": "PL",
	"WMI": "PL", "RZE": "PL",
	"BUD": "HU", "DEB": "HU",
	"OTP": "RO", "CLJ": "RO",
	"SOF": "BG",
	"BEG": "RS",
	// Baltics
	"RIX": "LV", "TLL": "EE", "VNO": "LT", "KUN": "LT",
	// Greece
	"ATH": "GR", "SKG": "GR", "HER": "GR", "RHO": "GR", "CFU": "GR",
	"CHQ": "GR", "JMK": "GR", "JTR": "GR", "ZTH": "GR", "KGS": "GR",
	// Turkey
	"IST": "TR", "AYT": "TR", "ADB": "TR", "DLM": "TR", "BJV": "TR",
	// Croatia
	"ZAG": "HR", "SPU": "HR", "DBV": "HR", "PUY": "HR", "ZAD": "HR",
	// Portugal
	"LIS": "PT", "OPO": "PT", "FAO": "PT",
	// Ireland
	"DUB": "IE", "SNN": "IE", "ORK": "IE",
	// Iceland
	"KEF": "IS",
}

// detectRegionalPass suggests relevant European flat-rate transport passes
// when the origin or destination is in a country with such a pass. This is
// purely advisory — zero API calls.
func detectRegionalPass(_ context.Context, in DetectorInput) []Hack {
	if in.Origin == "" && in.Destination == "" {
		return nil
	}

	// Collect country codes from origin and destination.
	countries := make(map[string]bool)
	if cc, ok := iataToCountry[strings.ToUpper(in.Origin)]; ok {
		countries[cc] = true
	}
	if cc, ok := iataToCountry[strings.ToUpper(in.Destination)]; ok {
		countries[cc] = true
	}

	if len(countries) == 0 {
		return nil
	}

	// Find all applicable passes.
	var hacks []Hack
	seen := make(map[string]bool) // deduplicate by pass name

	for _, pass := range regionalPasses {
		if seen[pass.Name] {
			continue
		}
		// Check if any of the trip's countries are in the pass's validity.
		applicable := false
		for _, vc := range pass.ValidFor {
			if countries[vc] {
				applicable = true
				break
			}
		}
		if !applicable {
			continue
		}

		seen[pass.Name] = true

		currency := in.currency()
		priceStr := fmt.Sprintf("EUR %.0f/%s", pass.PriceEUR, pass.Period)

		hacks = append(hacks, Hack{
			Type:     "regional_pass",
			Title:    fmt.Sprintf("%s — %s", pass.Name, priceStr),
			Currency: currency,
			Savings:  0, // advisory — no concrete savings estimate
			Description: fmt.Sprintf(
				"%s (%s): %s. %s. Valid in: %s.",
				pass.Name, priceStr, pass.Coverage, pass.Notes,
				strings.Join(pass.ValidFor, ", "),
			),
			Risks: []string{
				"Pass may not cover all transport modes (check restrictions)",
				fmt.Sprintf("Only useful if spending enough time in %s to justify the cost", pass.Country),
			},
			Steps: []string{
				fmt.Sprintf("Check if %s covers the routes you need: %s", pass.Name, pass.Coverage),
				fmt.Sprintf("Compare %s cost against buying individual tickets", priceStr),
				fmt.Sprintf("Note restrictions: %s", pass.Notes),
			},
		})
	}

	return hacks
}
