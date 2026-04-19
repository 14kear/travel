package hacks

import (
	"context"
	"testing"
)

func TestDetectMileageRun_emptyInput(t *testing.T) {
	hacks := detectMileageRun(context.Background(), DetectorInput{})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for empty input, got %d", len(hacks))
	}
}

func TestDetectMileageRun_missingOrigin(t *testing.T) {
	hacks := detectMileageRun(context.Background(), DetectorInput{
		Destination: "BCN",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks with missing origin, got %d", len(hacks))
	}
}

func TestDetectMileageRun_unknownAirport(t *testing.T) {
	// Airport not in any mileage run route.
	hacks := detectMileageRun(context.Background(), DetectorInput{
		Origin: "XYZ",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for unknown airport, got %d", len(hacks))
	}
}

func TestDetectMileageRun_noRoutesFromOrigin(t *testing.T) {
	// Airport that exists but has no mileage run routes.
	hacks := detectMileageRun(context.Background(), DetectorInput{
		Origin: "DUB",
	})
	if len(hacks) != 0 {
		t.Errorf("expected no hacks for airport without mileage runs, got %d", len(hacks))
	}
}

func TestDetectMileageRun_fromIstanbul(t *testing.T) {
	// IST is the From end of IST→AYT (Star Alliance).
	hacks := detectMileageRun(context.Background(), DetectorInput{
		Origin: "IST",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least 1 hack from IST")
	}
	found := false
	for _, h := range hacks {
		if h.Type != "mileage_run" {
			t.Errorf("expected type mileage_run, got %q", h.Type)
		}
		if h.Title != "" && h.Description != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected at least one hack with non-empty title and description")
	}
}

func TestDetectMileageRun_fromAYT(t *testing.T) {
	// AYT is the To end of IST→AYT; should match via reverse lookup.
	hacks := detectMileageRun(context.Background(), DetectorInput{
		Origin: "AYT",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least 1 hack from AYT (reverse of IST→AYT)")
	}
	if hacks[0].Type != "mileage_run" {
		t.Errorf("expected type mileage_run, got %q", hacks[0].Type)
	}
}

func TestDetectMileageRun_fromHEL(t *testing.T) {
	// HEL is the From end of HEL→ARN (Oneworld, Finnair).
	hacks := detectMileageRun(context.Background(), DetectorInput{
		Origin: "HEL",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least 1 hack from HEL")
	}
	h := hacks[0]
	if h.Type != "mileage_run" {
		t.Errorf("expected type mileage_run, got %q", h.Type)
	}
	if h.Savings != 0 {
		t.Errorf("advisory hack should have 0 savings, got %.0f", h.Savings)
	}
	if h.Currency != "EUR" {
		t.Errorf("expected EUR currency, got %q", h.Currency)
	}
	if len(h.Steps) == 0 {
		t.Error("expected non-empty steps")
	}
	if len(h.Risks) == 0 {
		t.Error("expected non-empty risks")
	}
}

func TestDetectMileageRun_fromMAD(t *testing.T) {
	// MAD is the From end of MAD→BCN (Oneworld, Iberia).
	hacks := detectMileageRun(context.Background(), DetectorInput{
		Origin: "MAD",
	})
	if len(hacks) == 0 {
		t.Fatal("expected at least 1 hack from MAD")
	}
}

func TestDetectMileageRun_maxThreeResults(t *testing.T) {
	// FCO is in both SkyTeam (FCO→MXP) routes. Ensure at most 3 returned.
	hacks := detectMileageRun(context.Background(), DetectorInput{
		Origin: "FCO",
	})
	if len(hacks) > 3 {
		t.Errorf("expected at most 3 hacks, got %d", len(hacks))
	}
}

func TestDetectMileageRun_caseInsensitive(t *testing.T) {
	// Lowercase origin should be normalised.
	hacks := detectMileageRun(context.Background(), DetectorInput{
		Origin: "hel",
	})
	// "hel" uppercases to "HEL" which should match.
	if len(hacks) == 0 {
		t.Fatal("expected hacks for lowercase 'hel'")
	}
}

func TestDetectMileageRun_allRoutesHaveData(t *testing.T) {
	// Verify all static routes have consistent data.
	for _, r := range cheapMileageRuns {
		if r.From == "" || r.To == "" {
			t.Error("mileage run route has empty From or To")
		}
		if r.Airline == "" {
			t.Errorf("route %s→%s has empty airline", r.From, r.To)
		}
		if r.Alliance == "" {
			t.Errorf("route %s→%s has empty alliance", r.From, r.To)
		}
		if r.CostEUR <= 0 {
			t.Errorf("route %s→%s has invalid cost: %.2f", r.From, r.To, r.CostEUR)
		}
		if r.MilesEarned <= 0 {
			t.Errorf("route %s→%s has invalid miles: %d", r.From, r.To, r.MilesEarned)
		}
		if r.CostPerMile <= 0 {
			t.Errorf("route %s→%s has invalid cost-per-mile: %.2f", r.From, r.To, r.CostPerMile)
		}
	}
}

func TestDetectMileageRun_allianceDisplayNames(t *testing.T) {
	for _, alliance := range []string{"star_alliance", "skyteam", "oneworld"} {
		name := allianceDisplayNames[alliance]
		if name == "" {
			t.Errorf("missing display name for alliance %q", alliance)
		}
	}
}
