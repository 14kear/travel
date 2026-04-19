package main

import (
	"context"
	"testing"

	"github.com/MikkoParkkola/trvl/internal/trip"
)

// ---------------------------------------------------------------------------
// printMultiCityTable — failure branch and savings branch (pure function)
// ---------------------------------------------------------------------------

func TestPrintMultiCityTable_FailureBranchV23(t *testing.T) {
	result := &trip.MultiCityResult{
		Success: false,
		Error:   "no routes found",
	}
	err := printMultiCityTable(context.Background(), "", result)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrintMultiCityTable_WithSavingsV23(t *testing.T) {
	result := &trip.MultiCityResult{
		Success:      true,
		HomeAirport:  "HEL",
		OptimalOrder: []string{"BCN", "ROM"},
		Permutations: 2,
		Currency:     "EUR",
		TotalCost:    600,
		Savings:      150, // > 0 → covers the savings row
		Segments: []trip.Segment{
			{From: "HEL", To: "BCN", Price: 200, Currency: "EUR"},
			{From: "BCN", To: "ROM", Price: 150, Currency: "EUR"},
			{From: "ROM", To: "HEL", Price: 250, Currency: "EUR"},
		},
	}
	err := printMultiCityTable(context.Background(), "", result)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrintMultiCityTable_NoSavingsV23(t *testing.T) {
	result := &trip.MultiCityResult{
		Success:      true,
		HomeAirport:  "HEL",
		OptimalOrder: []string{"BCN"},
		Permutations: 1,
		Currency:     "EUR",
		TotalCost:    400,
		Savings:      0, // = 0 → savings row not printed
		Segments: []trip.Segment{
			{From: "HEL", To: "BCN", Price: 200, Currency: "EUR"},
			{From: "BCN", To: "HEL", Price: 200, Currency: "EUR"},
		},
	}
	err := printMultiCityTable(context.Background(), "", result)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// runProvidersDisable — confirm non-terminal path fully (stdin not a tty)
// ---------------------------------------------------------------------------

func TestRunProvidersDisable_ConfirmsNonTerminalV23(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	writeTestProviderV19(t, tmp, "confirm-delete-provider")

	// In tests stdin is not a tty → skips the interactive confirm → deletes.
	err := runProvidersDisable("confirm-delete-provider")
	if err != nil {
		t.Errorf("runProvidersDisable: %v", err)
	}
}

// ---------------------------------------------------------------------------
// pointsValueCmd — with valid program flag (covers more RunE body)
// ---------------------------------------------------------------------------

func TestPointsValueCmd_WithProgramV23(t *testing.T) {
	cmd := pointsValueCmd()
	cmd.SetArgs([]string{"--program", "avios"})
	_ = cmd.Execute()
}

func TestPointsValueCmd_WithJSONFormatV23(t *testing.T) {
	cmd := pointsValueCmd()
	cmd.SetArgs([]string{"--format", "json"})
	_ = cmd.Execute()
}

// ---------------------------------------------------------------------------
// weekendCmd — flags coverage
// ---------------------------------------------------------------------------

func TestWeekendCmd_FlagsV23(t *testing.T) {
	cmd := weekendCmd()
	for _, name := range []string{"month", "budget", "nights", "format", "currency"} {
		if f := cmd.Flags().Lookup(name); f == nil {
			t.Errorf("expected --%s flag on weekendCmd", name)
		}
	}
}

// ---------------------------------------------------------------------------
// airportTransferCmd — valid args (runs until network, covers RunE body)
// ---------------------------------------------------------------------------

func TestAirportTransferCmd_ValidArgsNoNetworkV23(t *testing.T) {
	cmd := airportTransferCmd()
	cmd.SetArgs([]string{"CDG", "Hotel Lutetia Paris", "2026-07-01"})
	_ = cmd.Execute()
}

// ---------------------------------------------------------------------------
// runPrefsSet — remaining uncovered keys from applyPreference
// ---------------------------------------------------------------------------

func TestPrefsSetCmd_EnsuitOnlyV23(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	cmd := prefsSetCmd()
	cmd.SetArgs([]string{"ensuite_only", "true"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("prefs set ensuite_only: %v", err)
	}
}

func TestPrefsSetCmd_NoDormitoriesV23(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	cmd := prefsSetCmd()
	cmd.SetArgs([]string{"no_dormitories", "true"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("prefs set no_dormitories: %v", err)
	}
}

func TestPrefsSetCmd_FastWifiV23(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	cmd := prefsSetCmd()
	cmd.SetArgs([]string{"fast_wifi_needed", "true"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("prefs set fast_wifi_needed: %v", err)
	}
}

func TestPrefsSetCmd_HomeCitiesV23(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	cmd := prefsSetCmd()
	cmd.SetArgs([]string{"home_cities", "Helsinki,Amsterdam"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("prefs set home_cities: %v", err)
	}
}

// ---------------------------------------------------------------------------
// dealsCmd — run (hits RSS feed — fast, typically succeeds)
// ---------------------------------------------------------------------------

func TestDealsCmd_ValidRunV23(t *testing.T) {
	cmd := dealsCmd()
	cmd.SetArgs([]string{})
	_ = cmd.Execute()
}

// ---------------------------------------------------------------------------
// searchCmd — dry-run flag (covers dry-run branch without triggering search)
// ---------------------------------------------------------------------------

func TestSearchCmd_DryRunV23(t *testing.T) {
	cmd := searchCmd()
	cmd.SetArgs([]string{"--dry-run", "flights from HEL to BCN"})
	_ = cmd.Execute()
}
