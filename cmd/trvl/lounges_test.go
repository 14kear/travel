package main

import (
	"testing"

	"github.com/MikkoParkkola/trvl/internal/lounges"
	"github.com/MikkoParkkola/trvl/internal/preferences"
)

func TestLoungesCmd_NonNil(t *testing.T) {
	cmd := loungesCmd()
	if cmd == nil {
		t.Fatal("loungesCmd() returned nil")
	}
}

func TestLoungesCmd_Use(t *testing.T) {
	cmd := loungesCmd()
	want := "lounges AIRPORT"
	if cmd.Use != want {
		t.Errorf("loungesCmd Use = %q, want %q", cmd.Use, want)
	}
}

func TestLoungesCmd_RequiresExactlyOneArg(t *testing.T) {
	cmd := loungesCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error with 0 args")
	}
}

func TestLoungesCmd_HasValidArgsFunction(t *testing.T) {
	cmd := loungesCmd()
	if cmd.ValidArgsFunction == nil {
		t.Error("loungesCmd should have ValidArgsFunction for airport completion")
	}
}

func TestLoungeFFCards_Empty(t *testing.T) {
	cards := loungeFFCards(nil)
	if cards != nil {
		t.Errorf("expected nil for empty programs, got %v", cards)
	}
}

func TestLoungeFFCards_OneworldSapphire(t *testing.T) {
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "oneworld", Tier: "sapphire", AirlineCode: "AY"},
	}
	cards := loungeFFCards(programs)

	want := map[string]bool{"Oneworld Sapphire": false, "Finnair Plus Sapphire": false}
	for _, c := range cards {
		if _, ok := want[c]; ok {
			want[c] = true
		}
	}
	for card, found := range want {
		if !found {
			t.Errorf("missing expected card %q in %v", card, cards)
		}
	}
}

func TestLoungeFFCards_StarAllianceGold(t *testing.T) {
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "star_alliance", Tier: "gold", AirlineCode: "LH"},
	}
	cards := loungeFFCards(programs)

	found := false
	for _, c := range cards {
		if c == "Star Alliance Gold" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'Star Alliance Gold' in %v", cards)
	}
}

func TestLoungeFFCards_Dedup(t *testing.T) {
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "oneworld", Tier: "emerald", AirlineCode: "BA"},
		{Alliance: "oneworld", Tier: "emerald", AirlineCode: "QF"},
	}
	cards := loungeFFCards(programs)

	count := 0
	for _, c := range cards {
		if c == "Oneworld Emerald" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 'Oneworld Emerald' exactly once, got %d in %v", count, cards)
	}
}

func TestLoungeTierDisplay(t *testing.T) {
	tests := []struct {
		alliance string
		tier     string
		want     string
	}{
		{"oneworld", "sapphire", "Sapphire"},
		{"star_alliance", "gold", "Gold"},
		{"skyteam", "elite_plus", "Elite Plus"},
		{"unknown", "platinum", "Platinum"},
	}
	for _, tt := range tests {
		got := loungeTierDisplay(tt.alliance, tt.tier)
		if got != tt.want {
			t.Errorf("loungeTierDisplay(%q, %q) = %q, want %q", tt.alliance, tt.tier, got, tt.want)
		}
	}
}

func TestPrintLoungesTable_EmptyResult(t *testing.T) {
	result := &lounges.SearchResult{
		Success: true,
		Airport: "TST",
		Count:   0,
		Lounges: nil,
	}
	// Should not panic with zero lounges.
	err := printLoungesTable(result, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrintLoungesTable_WithAccess(t *testing.T) {
	result := &lounges.SearchResult{
		Success: true,
		Airport: "HEL",
		Count:   2,
		Lounges: []lounges.Lounge{
			{Name: "A", AccessibleWith: []string{"Priority Pass"}},
			{Name: "B"},
		},
	}
	prefs := &preferences.Preferences{
		LoungeCards: []string{"Priority Pass"},
	}
	// Should not panic.
	err := printLoungesTable(result, prefs)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
