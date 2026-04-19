package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestTravelModeRoundtrip verifies TravelMode marshals and unmarshals correctly.
func TestTravelModeRoundtrip(t *testing.T) {
	mode := TravelMode{
		Name:       "solo_remote",
		Companions: 0,
		Accom:      "apartment",
		AccomNeeds: []string{"wifi_fast", "laundry", "kitchen"},
		BudgetFlex: 1.2,
		Dining:     "cook",
		Transport:  "multimodal",
		Priority:   "experience",
	}

	b, err := json.Marshal(mode)
	if err != nil {
		t.Fatalf("marshal TravelMode: %v", err)
	}

	var got TravelMode
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal TravelMode: %v", err)
	}

	if got.Name != mode.Name {
		t.Errorf("Name = %q, want %q", got.Name, mode.Name)
	}
	if got.Companions != mode.Companions {
		t.Errorf("Companions = %d, want %d", got.Companions, mode.Companions)
	}
	if len(got.AccomNeeds) != len(mode.AccomNeeds) {
		t.Errorf("AccomNeeds len = %d, want %d", len(got.AccomNeeds), len(mode.AccomNeeds))
	}
	if got.BudgetFlex != mode.BudgetFlex {
		t.Errorf("BudgetFlex = %v, want %v", got.BudgetFlex, mode.BudgetFlex)
	}
}

// TestCityIntelligenceRoundtrip verifies CityIntelligence marshals and unmarshals correctly.
func TestCityIntelligenceRoundtrip(t *testing.T) {
	ci := CityIntelligence{
		City:           "Helsinki",
		KnowledgeLevel: "local",
		YearsLived:     5,
		Neighbourhoods: []string{"Kallio", "Töölö"},
		Restaurants:    []string{"Savoy", "Olo"},
		WhyVisit:       "home",
		TypicalStay:    7,
		Notes:          "home city, know it well",
	}

	b, err := json.Marshal(ci)
	if err != nil {
		t.Fatalf("marshal CityIntelligence: %v", err)
	}

	var got CityIntelligence
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal CityIntelligence: %v", err)
	}

	if got.City != ci.City {
		t.Errorf("City = %q, want %q", got.City, ci.City)
	}
	if got.KnowledgeLevel != ci.KnowledgeLevel {
		t.Errorf("KnowledgeLevel = %q, want %q", got.KnowledgeLevel, ci.KnowledgeLevel)
	}
	if got.YearsLived != ci.YearsLived {
		t.Errorf("YearsLived = %d, want %d", got.YearsLived, ci.YearsLived)
	}
	if len(got.Neighbourhoods) != len(ci.Neighbourhoods) {
		t.Errorf("Neighbourhoods len = %d, want %d", len(got.Neighbourhoods), len(ci.Neighbourhoods))
	}
	if got.TypicalStay != ci.TypicalStay {
		t.Errorf("TypicalStay = %d, want %d", got.TypicalStay, ci.TypicalStay)
	}
}

// TestBookingStrategyRoundtrip verifies BookingStrategy marshals and unmarshals correctly.
func TestBookingStrategyRoundtrip(t *testing.T) {
	bs := BookingStrategy{
		Name:        "Cheap fare + status upgrade",
		Pattern:     "cheapest_fare_plus_status",
		Description: "Book cheapest available fare, then use points or status to upgrade cabin",
		Frequency:   "often",
	}

	b, err := json.Marshal(bs)
	if err != nil {
		t.Fatalf("marshal BookingStrategy: %v", err)
	}

	var got BookingStrategy
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal BookingStrategy: %v", err)
	}

	if got.Pattern != bs.Pattern {
		t.Errorf("Pattern = %q, want %q", got.Pattern, bs.Pattern)
	}
	if got.Frequency != bs.Frequency {
		t.Errorf("Frequency = %q, want %q", got.Frequency, bs.Frequency)
	}
}

// TestPreferenceElasticityRoundtrip verifies PreferenceElasticity marshals and unmarshals correctly.
func TestPreferenceElasticityRoundtrip(t *testing.T) {
	tests := []struct {
		name       string
		elasticity PreferenceElasticity
	}{
		{
			name: "with price delta",
			elasticity: PreferenceElasticity{
				Factor:     "sauna",
				Impact:     "will_pay_more",
				PriceDelta: 1.3,
			},
		},
		{
			name: "dealbreaker zero delta",
			elasticity: PreferenceElasticity{
				Factor: "laundry",
				Impact: "dealbreaker",
				// PriceDelta intentionally zero (omitempty)
			},
		},
		{
			name: "nice to have",
			elasticity: PreferenceElasticity{
				Factor: "breakfast_quality",
				Impact: "nice_to_have",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.elasticity)
			if err != nil {
				t.Fatalf("marshal PreferenceElasticity: %v", err)
			}

			var got PreferenceElasticity
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("unmarshal PreferenceElasticity: %v", err)
			}

			if got.Factor != tc.elasticity.Factor {
				t.Errorf("Factor = %q, want %q", got.Factor, tc.elasticity.Factor)
			}
			if got.Impact != tc.elasticity.Impact {
				t.Errorf("Impact = %q, want %q", got.Impact, tc.elasticity.Impact)
			}
			if got.PriceDelta != tc.elasticity.PriceDelta {
				t.Errorf("PriceDelta = %v, want %v", got.PriceDelta, tc.elasticity.PriceDelta)
			}
		})
	}
}

// TestPreferenceElasticityOmitEmptyPriceDelta verifies that a zero PriceDelta is omitted from JSON.
func TestPreferenceElasticityOmitEmptyPriceDelta(t *testing.T) {
	pe := PreferenceElasticity{
		Factor: "laundry",
		Impact: "dealbreaker",
		// PriceDelta is zero — should be omitted
	}

	b, err := json.Marshal(pe)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	if _, ok := m["price_delta"]; ok {
		t.Error("price_delta should be omitted when zero, but it was present in JSON")
	}
}

// TestDestinationRelationshipRoundtrip verifies DestinationRelationship marshals and unmarshals correctly.
func TestDestinationRelationshipRoundtrip(t *testing.T) {
	dr := DestinationRelationship{
		City:      "Amsterdam",
		Reason:    "friends",
		Person:    "Janne",
		Frequency: "monthly",
		Sentiment: "love",
	}

	b, err := json.Marshal(dr)
	if err != nil {
		t.Fatalf("marshal DestinationRelationship: %v", err)
	}

	var got DestinationRelationship
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal DestinationRelationship: %v", err)
	}

	if got.City != dr.City {
		t.Errorf("City = %q, want %q", got.City, dr.City)
	}
	if got.Reason != dr.Reason {
		t.Errorf("Reason = %q, want %q", got.Reason, dr.Reason)
	}
	if got.Person != dr.Person {
		t.Errorf("Person = %q, want %q", got.Person, dr.Person)
	}
	if got.Frequency != dr.Frequency {
		t.Errorf("Frequency = %q, want %q", got.Frequency, dr.Frequency)
	}
	if got.Sentiment != dr.Sentiment {
		t.Errorf("Sentiment = %q, want %q", got.Sentiment, dr.Sentiment)
	}
}

// TestDestinationRelationshipPersonOmitted verifies that Person is omitted from JSON when empty.
func TestDestinationRelationshipPersonOmitted(t *testing.T) {
	dr := DestinationRelationship{
		City:      "Tokyo",
		Reason:    "aspirational",
		Frequency: "never_been",
		Sentiment: "curious",
		// Person is empty — should be omitted
	}

	b, err := json.Marshal(dr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	if _, ok := m["person"]; ok {
		t.Error("person should be omitted when empty, but it was present in JSON")
	}
}

// TestTravelProfileReasoningLayerSaveLoad verifies the full reasoning layer
// survives a save/load roundtrip, and that existing profile fields are preserved.
func TestTravelProfileReasoningLayerSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.json")

	p := &TravelProfile{
		// Existing fields — must survive unchanged.
		TotalTrips:   12,
		TotalFlights: 24,
		BudgetTier:   "mid-range",
		Bookings: []Booking{
			{Type: "flight", Provider: "Finnair", From: "HEL", To: "AMS", Price: 199},
		},
		// New reasoning layer.
		TravelModes: []TravelMode{
			{
				Name:       "solo_remote",
				Companions: 0,
				Accom:      "apartment",
				AccomNeeds: []string{"wifi_fast", "laundry"},
				BudgetFlex: 1.15,
				Dining:     "cook",
				Transport:  "multimodal",
				Priority:   "experience",
			},
			{
				Name:       "with_partner",
				Companions: 1,
				Accom:      "boutique_hotel",
				AccomNeeds: []string{"breakfast", "central_location"},
				BudgetFlex: 1.4,
				Dining:     "eat_out",
				Transport:  "flights_only",
				Priority:   "experience",
			},
		},
		CityIntelligence: []CityIntelligence{
			{
				City:           "Helsinki",
				KnowledgeLevel: "local",
				YearsLived:     8,
				Neighbourhoods: []string{"Kallio", "Töölö"},
				WhyVisit:       "home",
				TypicalStay:    7,
			},
			{
				City:           "Amsterdam",
				KnowledgeLevel: "regular",
				WhyVisit:       "friends",
				TypicalStay:    4,
				Notes:          "always stay in Jordaan",
			},
		},
		BookingStrategies: []BookingStrategy{
			{
				Name:        "Snap watching",
				Pattern:     "snap_watching",
				Description: "Watch prices for weeks, snap when a predictable low appears",
				Frequency:   "always",
			},
		},
		PriceElasticity: []PreferenceElasticity{
			{Factor: "sauna", Impact: "will_pay_more", PriceDelta: 1.25},
			{Factor: "laundry", Impact: "dealbreaker"},
			{Factor: "modern_interior", Impact: "will_pay_more", PriceDelta: 1.1},
		},
		DestinationGraph: []DestinationRelationship{
			{City: "Helsinki", Reason: "home", Frequency: "weekly", Sentiment: "practical"},
			{City: "Amsterdam", Reason: "friends", Person: "Janne", Frequency: "monthly", Sentiment: "love"},
			{City: "Tokyo", Reason: "aspirational", Frequency: "never_been", Sentiment: "curious"},
		},
		TravelIdentity:    "Multimodal optimizer who works remotely from favourite cities",
		DecisionFramework: "Price first, then filter by quality. Never overpays but will stretch for the right experience.",
	}

	if err := SaveTo(path, p); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	// Verify existing fields are intact.
	if loaded.TotalTrips != 12 {
		t.Errorf("TotalTrips = %d, want 12", loaded.TotalTrips)
	}
	if loaded.BudgetTier != "mid-range" {
		t.Errorf("BudgetTier = %q, want mid-range", loaded.BudgetTier)
	}
	if len(loaded.Bookings) != 1 {
		t.Fatalf("Bookings len = %d, want 1", len(loaded.Bookings))
	}

	// Verify TravelModes.
	if len(loaded.TravelModes) != 2 {
		t.Fatalf("TravelModes len = %d, want 2", len(loaded.TravelModes))
	}
	if loaded.TravelModes[0].Name != "solo_remote" {
		t.Errorf("TravelModes[0].Name = %q, want solo_remote", loaded.TravelModes[0].Name)
	}
	if loaded.TravelModes[0].BudgetFlex != 1.15 {
		t.Errorf("TravelModes[0].BudgetFlex = %v, want 1.15", loaded.TravelModes[0].BudgetFlex)
	}
	if len(loaded.TravelModes[0].AccomNeeds) != 2 {
		t.Errorf("TravelModes[0].AccomNeeds len = %d, want 2", len(loaded.TravelModes[0].AccomNeeds))
	}

	// Verify CityIntelligence.
	if len(loaded.CityIntelligence) != 2 {
		t.Fatalf("CityIntelligence len = %d, want 2", len(loaded.CityIntelligence))
	}
	if loaded.CityIntelligence[0].City != "Helsinki" {
		t.Errorf("CityIntelligence[0].City = %q, want Helsinki", loaded.CityIntelligence[0].City)
	}
	if loaded.CityIntelligence[0].YearsLived != 8 {
		t.Errorf("CityIntelligence[0].YearsLived = %d, want 8", loaded.CityIntelligence[0].YearsLived)
	}

	// Verify BookingStrategies.
	if len(loaded.BookingStrategies) != 1 {
		t.Fatalf("BookingStrategies len = %d, want 1", len(loaded.BookingStrategies))
	}
	if loaded.BookingStrategies[0].Pattern != "snap_watching" {
		t.Errorf("BookingStrategies[0].Pattern = %q, want snap_watching", loaded.BookingStrategies[0].Pattern)
	}

	// Verify PriceElasticity.
	if len(loaded.PriceElasticity) != 3 {
		t.Fatalf("PriceElasticity len = %d, want 3", len(loaded.PriceElasticity))
	}
	if loaded.PriceElasticity[0].Factor != "sauna" {
		t.Errorf("PriceElasticity[0].Factor = %q, want sauna", loaded.PriceElasticity[0].Factor)
	}
	if loaded.PriceElasticity[0].PriceDelta != 1.25 {
		t.Errorf("PriceElasticity[0].PriceDelta = %v, want 1.25", loaded.PriceElasticity[0].PriceDelta)
	}
	// dealbreaker entry has zero PriceDelta.
	if loaded.PriceElasticity[1].PriceDelta != 0 {
		t.Errorf("PriceElasticity[1].PriceDelta = %v, want 0", loaded.PriceElasticity[1].PriceDelta)
	}

	// Verify DestinationGraph.
	if len(loaded.DestinationGraph) != 3 {
		t.Fatalf("DestinationGraph len = %d, want 3", len(loaded.DestinationGraph))
	}
	if loaded.DestinationGraph[1].Person != "Janne" {
		t.Errorf("DestinationGraph[1].Person = %q, want Janne", loaded.DestinationGraph[1].Person)
	}
	if loaded.DestinationGraph[2].Sentiment != "curious" {
		t.Errorf("DestinationGraph[2].Sentiment = %q, want curious", loaded.DestinationGraph[2].Sentiment)
	}

	// Verify TravelIdentity and DecisionFramework.
	if loaded.TravelIdentity != p.TravelIdentity {
		t.Errorf("TravelIdentity = %q, want %q", loaded.TravelIdentity, p.TravelIdentity)
	}
	if loaded.DecisionFramework != p.DecisionFramework {
		t.Errorf("DecisionFramework = %q, want %q", loaded.DecisionFramework, p.DecisionFramework)
	}
}

// TestTravelProfileReasoningLayerZeroValue verifies that a profile with no
// reasoning fields loaded from JSON (e.g. an old profile) has nil slices and
// empty strings — not panics or garbage values.
func TestTravelProfileReasoningLayerZeroValue(t *testing.T) {
	// Minimal old-style profile JSON — no reasoning layer keys.
	raw := `{"total_trips":3,"total_flights":6,"budget_tier":"budget"}`

	dir := t.TempDir()
	path := filepath.Join(dir, "profile.json")

	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	// Existing fields must parse correctly.
	if loaded.TotalTrips != 3 {
		t.Errorf("TotalTrips = %d, want 3", loaded.TotalTrips)
	}

	// Reasoning fields should be nil/empty (not garbage).
	if loaded.TravelModes != nil {
		t.Errorf("TravelModes should be nil for old profile, got %v", loaded.TravelModes)
	}
	if loaded.CityIntelligence != nil {
		t.Errorf("CityIntelligence should be nil for old profile, got %v", loaded.CityIntelligence)
	}
	if loaded.BookingStrategies != nil {
		t.Errorf("BookingStrategies should be nil for old profile, got %v", loaded.BookingStrategies)
	}
	if loaded.PriceElasticity != nil {
		t.Errorf("PriceElasticity should be nil for old profile, got %v", loaded.PriceElasticity)
	}
	if loaded.DestinationGraph != nil {
		t.Errorf("DestinationGraph should be nil for old profile, got %v", loaded.DestinationGraph)
	}
	if loaded.TravelIdentity != "" {
		t.Errorf("TravelIdentity should be empty for old profile, got %q", loaded.TravelIdentity)
	}
	if loaded.DecisionFramework != "" {
		t.Errorf("DecisionFramework should be empty for old profile, got %q", loaded.DecisionFramework)
	}
}

// TestReasoningLayerOmittedFromEmptyProfile verifies that an empty TravelProfile
// marshals without any reasoning layer keys (omitempty in action).
func TestReasoningLayerOmittedFromEmptyProfile(t *testing.T) {
	p := &TravelProfile{}

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal empty profile: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	omitKeys := []string{
		"travel_modes", "city_intelligence", "booking_strategies",
		"price_elasticity", "destination_graph",
		"travel_identity", "decision_framework",
	}
	for _, key := range omitKeys {
		if _, ok := m[key]; ok {
			t.Errorf("key %q should be omitted from empty profile JSON, but was present", key)
		}
	}
}
