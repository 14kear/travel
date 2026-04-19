package hacks

import (
	"context"
	"testing"
)

func TestDetectDepartureTax_emptyInput(t *testing.T) {
	hacks := detectDepartureTax(context.Background(), DetectorInput{})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for empty input, got %d", len(hacks))
	}
}

func TestDetectDepartureTax_missingOrigin(t *testing.T) {
	hacks := detectDepartureTax(context.Background(), DetectorInput{
		Destination: "BCN",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for missing origin, got %d", len(hacks))
	}
}

func TestDetectDepartureTax_missingDestination(t *testing.T) {
	hacks := detectDepartureTax(context.Background(), DetectorInput{
		Origin: "AMS",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for missing destination, got %d", len(hacks))
	}
}

func TestDetectDepartureTax_unknownAirport(t *testing.T) {
	hacks := detectDepartureTax(context.Background(), DetectorInput{
		Origin:      "XYZ",
		Destination: "ABC",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for unknown airports, got %d", len(hacks))
	}
}

func TestDetectDepartureTax_zeroTaxOrigin(t *testing.T) {
	// Helsinki (FI) has zero aviation tax — should return nil.
	hacks := detectDepartureTax(context.Background(), DetectorInput{
		Origin:      "HEL",
		Destination: "BCN",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for zero-tax origin (FI), got %d", len(hacks))
	}
}

func TestDetectDepartureTax_zeroTaxOriginPrague(t *testing.T) {
	// Prague (CZ) has zero aviation tax — should return nil.
	hacks := detectDepartureTax(context.Background(), DetectorInput{
		Origin:      "PRG",
		Destination: "BCN",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for zero-tax origin (CZ), got %d", len(hacks))
	}
}

func TestDetectDepartureTax_highTaxNoAlternative(t *testing.T) {
	// MAD (ES) — Spain is not in departureTaxEUR so should return nil.
	hacks := detectDepartureTax(context.Background(), DetectorInput{
		Origin:      "MAD",
		Destination: "HEL",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for origin country not in tax map, got %d", len(hacks))
	}
}

func TestDetectDepartureTax_cphToHel(t *testing.T) {
	// CPH (DK) has EUR 5 tax. Nearby: MMX (SE, zero tax after 2025 abolition).
	hacks := detectDepartureTax(context.Background(), DetectorInput{
		Origin:      "CPH",
		Destination: "BCN",
	})
	// CPH has nearby airports (MMX is SE = zero tax).
	if len(hacks) == 0 {
		t.Fatal("expected a hack for CPH (DK tax) with MMX (SE zero tax) alternative")
	}
	h := hacks[0]
	if h.Type != "departure_tax" {
		t.Errorf("type = %q, want departure_tax", h.Type)
	}
	if h.Savings <= 0 {
		t.Errorf("savings should be > 0, got %.0f", h.Savings)
	}
	if h.Title == "" {
		t.Error("title is empty")
	}
	if h.Description == "" {
		t.Error("description is empty")
	}
	if len(h.Steps) == 0 {
		t.Error("steps are empty")
	}
	if len(h.Risks) == 0 {
		t.Error("risks are empty")
	}
}

func TestDetectDepartureTax_caseInsensitive(t *testing.T) {
	// Lowercase input should work.
	hacks := detectDepartureTax(context.Background(), DetectorInput{
		Origin:      "cph",
		Destination: "bcn",
	})
	// CPH has nearby zero-tax alternatives.
	if len(hacks) == 0 {
		t.Fatal("expected hack for lowercase cph")
	}
}

func TestDetectDepartureTax_currencyDefault(t *testing.T) {
	hacks := detectDepartureTax(context.Background(), DetectorInput{
		Origin:      "CPH",
		Destination: "BCN",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack")
	}
	if hacks[0].Currency != "EUR" {
		t.Errorf("currency = %q, want EUR", hacks[0].Currency)
	}
}

func TestDetectDepartureTax_customCurrency(t *testing.T) {
	hacks := detectDepartureTax(context.Background(), DetectorInput{
		Origin:      "CPH",
		Destination: "BCN",
		Currency:    "DKK",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least one hack")
	}
	if hacks[0].Currency != "DKK" {
		t.Errorf("currency = %q, want DKK", hacks[0].Currency)
	}
}

// --- Static data tests ---

func TestDepartureTaxEUR_populated(t *testing.T) {
	if len(departureTaxEUR) == 0 {
		t.Fatal("departureTaxEUR is empty")
	}
}

func TestDepartureTaxEUR_zeroTaxCountries(t *testing.T) {
	zeroTax := []string{"IE", "PT", "CY", "MT", "FI", "EE", "LV", "LT", "CZ", "PL", "HU", "RO", "BG", "HR", "SE"}
	for _, cc := range zeroTax {
		tax, ok := departureTaxEUR[cc]
		if !ok {
			t.Errorf("missing zero-tax country %s from departureTaxEUR", cc)
			continue
		}
		if tax != 0 {
			t.Errorf("expected zero tax for %s, got %.0f", cc, tax)
		}
	}
}

func TestDepartureTaxEUR_highTaxCountries(t *testing.T) {
	highTax := []string{"GB", "DE", "FR", "NL", "AT", "NO"}
	for _, cc := range highTax {
		tax, ok := departureTaxEUR[cc]
		if !ok {
			t.Errorf("missing high-tax country %s from departureTaxEUR", cc)
			continue
		}
		if tax <= 0 {
			t.Errorf("expected positive tax for %s, got %.0f", cc, tax)
		}
	}
}

func TestCountryName_known(t *testing.T) {
	if got := countryName("NL"); got != "Netherlands" {
		t.Errorf("countryName(NL) = %q, want Netherlands", got)
	}
}

func TestCountryName_unknown(t *testing.T) {
	if got := countryName("XX"); got != "XX" {
		t.Errorf("countryName(XX) = %q, want XX", got)
	}
}
