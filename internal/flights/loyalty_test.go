package flights

import (
	"testing"

	"github.com/MikkoParkkola/trvl/internal/models"
	"github.com/MikkoParkkola/trvl/internal/preferences"
)

// helpers — underscore-assign to satisfy staticcheck (kept for future tests)
var (
	_ = boolPtr
	_ = intPtr
)

func boolPtr(b bool) *bool { return &b }
func intPtr(n int) *int    { return &n }

func makeFlight(airlineCode string, checkedBags *int) models.FlightResult {
	return models.FlightResult{
		Price:    200,
		Currency: "EUR",
		Legs: []models.FlightLeg{
			{AirlineCode: airlineCode, Airline: "Test Airline"},
		},
		CheckedBagsIncluded: checkedBags,
	}
}

// --- normalizeTier ---

func TestNormalizeTier(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Gold", "gold"},
		{"ELITE_PLUS", "elite_plus"},
		{"Elite Plus", "elite_plus"},
		{"elite-plus", "elite_plus"},
		{"  Sapphire  ", "sapphire"},
	}
	for _, tc := range cases {
		got := normalizeTier(tc.in)
		if got != tc.want {
			t.Errorf("normalizeTier(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// --- bagBenefit ---

func TestBagBenefit_KnownTiers(t *testing.T) {
	cases := []struct {
		alliance string
		tier     string
		want     int
	}{
		{"oneworld", "ruby", 0},
		{"oneworld", "sapphire", 1},
		{"oneworld", "emerald", 1},
		{"skyteam", "elite", 1},
		{"skyteam", "elite_plus", 1},
		{"star_alliance", "silver", 1},
		{"star_alliance", "gold", 1},
	}
	for _, tc := range cases {
		got := bagBenefit(tc.alliance, tc.tier)
		if got != tc.want {
			t.Errorf("bagBenefit(%q, %q) = %d, want %d", tc.alliance, tc.tier, got, tc.want)
		}
	}
}

func TestBagBenefit_UnknownAlliance(t *testing.T) {
	if got := bagBenefit("unknown_alliance", "gold"); got != 0 {
		t.Errorf("expected 0 for unknown alliance, got %d", got)
	}
}

func TestBagBenefit_UnknownTier(t *testing.T) {
	if got := bagBenefit("oneworld", "platinum"); got != 0 {
		t.Errorf("expected 0 for unknown tier, got %d", got)
	}
}

// --- freeBagGranted ---

func TestFreeBagGranted_AllianceMatch(t *testing.T) {
	// AY is Oneworld; user has Sapphire -> free bag
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "oneworld", Tier: "sapphire"},
	}
	if !freeBagGranted("AY", programs) {
		t.Error("expected free bag for Oneworld Sapphire on AY (Oneworld member)")
	}
}

func TestFreeBagGranted_NoAllianceMatch(t *testing.T) {
	// LH is Star Alliance; user has Oneworld Sapphire -> no benefit
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "oneworld", Tier: "sapphire"},
	}
	if freeBagGranted("LH", programs) {
		t.Error("expected no free bag for Oneworld status on LH (Star Alliance member)")
	}
}

func TestFreeBagGranted_LowTierNoFreeBag(t *testing.T) {
	// Oneworld Ruby does not grant a free bag
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "oneworld", Tier: "ruby"},
	}
	if freeBagGranted("BA", programs) {
		t.Error("expected no free bag for Oneworld Ruby")
	}
}

func TestFreeBagGranted_DirectCarrierMatch(t *testing.T) {
	// User has Gold status with LH (Star Alliance Gold via direct carrier).
	// The flight is also operated by LH.
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "star_alliance", Tier: "gold", AirlineCode: "LH"},
	}
	if !freeBagGranted("LH", programs) {
		t.Error("expected free bag for direct carrier match (LH Gold on LH flight)")
	}
}

func TestFreeBagGranted_DirectCarrierMismatch(t *testing.T) {
	// User has status with LH; flight is operated by UA (also Star Alliance).
	// Direct carrier code specified — only direct match counts here.
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "star_alliance", Tier: "gold", AirlineCode: "LH"},
	}
	// UA is star_alliance, and user's alliance is star_alliance -> still matches via alliance.
	if !freeBagGranted("UA", programs) {
		t.Error("expected free bag: user's alliance star_alliance matches UA (Star Alliance member)")
	}
}

func TestFreeBagGranted_EmptyPrograms(t *testing.T) {
	if freeBagGranted("BA", nil) {
		t.Error("expected no free bag when programs list is empty")
	}
}

func TestFreeBagGranted_UnknownAirline(t *testing.T) {
	// Non-alliance carrier: no membership entry, so alliance match won't fire.
	// Only direct carrier code match could work.
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "oneworld", Tier: "emerald"},
	}
	if freeBagGranted("W6", programs) {
		t.Error("expected no free bag for non-alliance carrier W6")
	}
}

func TestFreeBagGranted_CaseInsensitive(t *testing.T) {
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "SKYTEAM", Tier: "Elite Plus"},
	}
	if !freeBagGranted("kl", programs) {
		t.Error("expected free bag: SkyTeam Elite Plus on KL, case-insensitive match")
	}
}

// --- AdjustBagAllowance ---

func TestAdjustBagAllowance_NoPrograms(t *testing.T) {
	flights := []models.FlightResult{makeFlight("BA", nil)}
	got := AdjustBagAllowance(flights, nil)
	if got[0].CheckedBagsIncluded != nil {
		t.Error("expected nil checked bags when no programs configured")
	}
}

func TestAdjustBagAllowance_AlreadyHasBag(t *testing.T) {
	// Flight already has 2 bags — should not be downgraded.
	two := 2
	flights := []models.FlightResult{makeFlight("BA", &two)}
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "oneworld", Tier: "sapphire"},
	}
	got := AdjustBagAllowance(flights, programs)
	if got[0].CheckedBagsIncluded == nil || *got[0].CheckedBagsIncluded != 2 {
		t.Error("expected existing bag count preserved when already ≥1")
	}
}

func TestAdjustBagAllowance_BagGranted(t *testing.T) {
	// BA is Oneworld; Sapphire grants 1 bag.
	flights := []models.FlightResult{makeFlight("BA", nil)}
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "oneworld", Tier: "sapphire"},
	}
	got := AdjustBagAllowance(flights, programs)
	if got[0].CheckedBagsIncluded == nil || *got[0].CheckedBagsIncluded != 1 {
		t.Errorf("expected CheckedBagsIncluded=1, got %v", got[0].CheckedBagsIncluded)
	}
}

func TestAdjustBagAllowance_NoBagGrantedLowTier(t *testing.T) {
	// BA is Oneworld; Ruby does not grant a free bag.
	flights := []models.FlightResult{makeFlight("BA", nil)}
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "oneworld", Tier: "ruby"},
	}
	got := AdjustBagAllowance(flights, programs)
	if got[0].CheckedBagsIncluded != nil {
		t.Error("expected no bag for Oneworld Ruby tier")
	}
}

func TestAdjustBagAllowance_NoBagGrantedWrongAlliance(t *testing.T) {
	// LH is Star Alliance; user only has Oneworld status.
	flights := []models.FlightResult{makeFlight("LH", nil)}
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "oneworld", Tier: "emerald"},
	}
	got := AdjustBagAllowance(flights, programs)
	if got[0].CheckedBagsIncluded != nil {
		t.Error("expected no bag for Oneworld status on Star Alliance flight")
	}
}

func TestAdjustBagAllowance_NoLegs(t *testing.T) {
	// Flight with no legs — should not panic or mutate.
	f := models.FlightResult{Price: 100, Currency: "EUR"}
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "oneworld", Tier: "emerald"},
	}
	got := AdjustBagAllowance([]models.FlightResult{f}, programs)
	if got[0].CheckedBagsIncluded != nil {
		t.Error("expected no bag for flight with no legs")
	}
}

func TestAdjustBagAllowance_DoesNotMutateInput(t *testing.T) {
	original := []models.FlightResult{makeFlight("BA", nil)}
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "oneworld", Tier: "sapphire"},
	}
	_ = AdjustBagAllowance(original, programs)
	// Original slice element should be unchanged.
	if original[0].CheckedBagsIncluded != nil {
		t.Error("AdjustBagAllowance must not mutate the input slice")
	}
}

func TestAdjustBagAllowance_MultipleFlightsMixed(t *testing.T) {
	// 3 flights: BA (Oneworld), LH (Star Alliance), W6 (no alliance).
	// User has Oneworld Sapphire.
	flights := []models.FlightResult{
		makeFlight("BA", nil), // should get bag
		makeFlight("LH", nil), // should NOT get bag
		makeFlight("W6", nil), // should NOT get bag
	}
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "oneworld", Tier: "sapphire"},
	}
	got := AdjustBagAllowance(flights, programs)
	if got[0].CheckedBagsIncluded == nil || *got[0].CheckedBagsIncluded != 1 {
		t.Error("expected bag for BA (Oneworld member)")
	}
	if got[1].CheckedBagsIncluded != nil {
		t.Error("expected no bag for LH (Star Alliance)")
	}
	if got[2].CheckedBagsIncluded != nil {
		t.Error("expected no bag for W6 (no alliance)")
	}
}

func TestAdjustBagAllowance_SkyTeamElite(t *testing.T) {
	// KL is SkyTeam; Elite grants 1 bag.
	flights := []models.FlightResult{makeFlight("KL", nil)}
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "skyteam", Tier: "elite"},
	}
	got := AdjustBagAllowance(flights, programs)
	if got[0].CheckedBagsIncluded == nil || *got[0].CheckedBagsIncluded != 1 {
		t.Errorf("expected bag for SkyTeam Elite on KL, got %v", got[0].CheckedBagsIncluded)
	}
}

func TestAdjustBagAllowance_StarAllianceSilver(t *testing.T) {
	// UA is Star Alliance; Silver grants 1 bag.
	flights := []models.FlightResult{makeFlight("UA", nil)}
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "star_alliance", Tier: "silver"},
	}
	got := AdjustBagAllowance(flights, programs)
	if got[0].CheckedBagsIncluded == nil || *got[0].CheckedBagsIncluded != 1 {
		t.Errorf("expected bag for Star Alliance Silver on UA, got %v", got[0].CheckedBagsIncluded)
	}
}

func TestAdjustBagAllowance_ZeroCheckedBagsIsUpgraded(t *testing.T) {
	// CheckedBagsIncluded=0 means explicitly no bags — we should upgrade to 1
	// when the user has status.
	zero := 0
	flights := []models.FlightResult{makeFlight("BA", &zero)}
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "oneworld", Tier: "emerald"},
	}
	got := AdjustBagAllowance(flights, programs)
	if got[0].CheckedBagsIncluded == nil || *got[0].CheckedBagsIncluded != 1 {
		t.Errorf("expected CheckedBagsIncluded upgraded from 0 to 1, got %v", got[0].CheckedBagsIncluded)
	}
}

// --- allianceMembership coverage ---

func TestAllianceMembership_Oneworld(t *testing.T) {
	for _, code := range []string{"AA", "BA", "IB", "AY", "QF", "CX", "JL", "QR"} {
		if got := allianceMembership[code]; got != "oneworld" {
			t.Errorf("allianceMembership[%q] = %q, want oneworld", code, got)
		}
	}
}

func TestAllianceMembership_SkyTeam(t *testing.T) {
	for _, code := range []string{"AF", "KL", "DL", "KE"} {
		if got := allianceMembership[code]; got != "skyteam" {
			t.Errorf("allianceMembership[%q] = %q, want skyteam", code, got)
		}
	}
}

func TestAllianceMembership_StarAlliance(t *testing.T) {
	for _, code := range []string{"UA", "LH", "AC", "SQ", "NH", "TK", "LX"} {
		if got := allianceMembership[code]; got != "star_alliance" {
			t.Errorf("allianceMembership[%q] = %q, want star_alliance", code, got)
		}
	}
}
