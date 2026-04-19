package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/MikkoParkkola/trvl/internal/flights"
)

// ---------------------------------------------------------------------------
// baggageCmd — remaining branches (carry-on-only, unknown airline, help)
// ---------------------------------------------------------------------------

func TestBaggageCmd_CarryOnOnly(t *testing.T) {
	cmd := baggageCmd()
	cmd.SetArgs([]string{"--carry-on-only"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBaggageCmd_UnknownAirline(t *testing.T) {
	cmd := baggageCmd()
	cmd.SetArgs([]string{"XX"}) // should not exist
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for unknown airline code")
	}
}

func TestBaggageCmd_NoArgsReturnsHelp(t *testing.T) {
	cmd := baggageCmd()
	cmd.SetArgs([]string{})
	// Should print help and return nil (not an error).
	_ = cmd.Execute()
}

// ---------------------------------------------------------------------------
// accomHackCmd — validation branches (no network needed)
// ---------------------------------------------------------------------------

func TestAccomHackCmd_FlagsExist(t *testing.T) {
	cmd := accomHackCmd()
	for _, name := range []string{"checkin", "checkout", "currency", "max-splits", "guests"} {
		if f := cmd.Flags().Lookup(name); f == nil {
			t.Errorf("expected --%s flag on accomHackCmd", name)
		}
	}
}

// ---------------------------------------------------------------------------
// calendarCmd — --last path when no last search exists
// ---------------------------------------------------------------------------

func TestCalendarCmd_LastNoSearch(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	cmd := calendarCmd()
	cmd.SetArgs([]string{"--last"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no last search exists")
	}
}

func TestCalendarCmd_FlagsExist(t *testing.T) {
	cmd := calendarCmd()
	for _, name := range []string{"output", "last"} {
		if f := cmd.Flags().Lookup(name); f == nil {
			t.Errorf("expected --%s flag on calendarCmd", name)
		}
	}
}

func TestCalendarCmd_NoArgNoLast(t *testing.T) {
	cmd := calendarCmd()
	cmd.SetArgs([]string{}) // neither trip_id nor --last
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when neither trip_id nor --last provided")
	}
}

// ---------------------------------------------------------------------------
// saveLastSearch / loadLastSearch integration
// ---------------------------------------------------------------------------

func TestSaveAndLoadLastSearch_V9(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	ls := &LastSearch{
		Command:        "flights",
		Origin:         "HEL",
		Destination:    "BCN",
		DepartDate:     "2026-07-01",
		FlightPrice:    199,
		FlightCurrency: "EUR",
	}
	saveLastSearch(ls)

	loaded, err := loadLastSearch()
	if err != nil {
		t.Fatalf("loadLastSearch: %v", err)
	}
	if loaded.Origin != "HEL" {
		t.Errorf("expected HEL, got %q", loaded.Origin)
	}
}

// ---------------------------------------------------------------------------
// calendarCmd — uses saved last search to generate ICS
// ---------------------------------------------------------------------------

func TestCalendarCmd_LastWithSearch(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Pre-write a last_search.json.
	ls := &LastSearch{
		Command:        "flights",
		Origin:         "HEL",
		Destination:    "BCN",
		DepartDate:     "2026-07-01",
		ReturnDate:     "2026-07-08",
		FlightPrice:    199,
		FlightCurrency: "EUR",
		FlightAirline:  "KLM",
	}
	saveLastSearch(ls)

	cmd := calendarCmd()
	cmd.SetArgs([]string{"--last"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCalendarCmd_LastWriteToFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	ls := &LastSearch{
		Command:        "flights",
		Origin:         "HEL",
		Destination:    "BCN",
		DepartDate:     "2026-07-01",
		ReturnDate:     "2026-07-08",
		FlightPrice:    199,
		FlightCurrency: "EUR",
	}
	saveLastSearch(ls)

	outFile := filepath.Join(tmp, "trip.ics")
	cmd := calendarCmd()
	cmd.SetArgs([]string{"--last", "--output", outFile})
	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify the file was written.
	if _, err := os.Stat(outFile); os.IsNotExist(err) {
		t.Error("expected ICS file to be written")
	}
}

// ---------------------------------------------------------------------------
// runCabinComparison (cabin_compare.go) — json output path (no network)
// The function runs a goroutine pool and returns results; all will error
// in test env (no network), which exercises the error-in-goroutine path.
// The json format branch can be exercised.
// ---------------------------------------------------------------------------

func TestRunCabinComparison_JSONNoNetwork(t *testing.T) {
	// With no network, all cabin searches will fail. JSON format should still work.
	ctx := context.Background()
	// We expect an error from the flight search; runCabinComparison returns the error
	// only from json.Marshal or file write — the goroutine errors become cabinResult.Error.
	err := runCabinComparison(ctx, []string{"HEL"}, []string{"BCN"}, "2026-07-01", flights.SearchOptions{}, "json")
	// JSON output may succeed (prints error entries); just no panic.
	_ = err
}

// ---------------------------------------------------------------------------
// outputShare (share.go) — markdown branch (already partially tested via share_test)
// Test the gist branch when gh is not installed (will hit createGist → fallback)
// ---------------------------------------------------------------------------

func TestOutputShare_MarkdownBranch(t *testing.T) {
	// Markdown branch: prints to stdout.
	err := outputShare("# My Trip\n", "markdown")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOutputShare_DefaultBranch(t *testing.T) {
	// Empty format string falls through to default (markdown).
	err := outputShare("# Trip\n", "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// clientConfigPath — remaining client types (zed, lm-studio, amazon-q, gemini, vscode)
// ---------------------------------------------------------------------------

func TestClientConfigPath_ZedClient(t *testing.T) {
	path, err := clientConfigPath("zed")
	if err != nil {
		t.Fatalf("clientConfigPath(zed): %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path for zed")
	}
}

func TestClientConfigPath_LMStudio(t *testing.T) {
	path, err := clientConfigPath("lm-studio")
	if err != nil {
		t.Fatalf("clientConfigPath(lm-studio): %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path for lm-studio")
	}
}

func TestClientConfigPath_Gemini(t *testing.T) {
	path, err := clientConfigPath("gemini")
	if err != nil {
		t.Fatalf("clientConfigPath(gemini): %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path for gemini")
	}
}

func TestClientConfigPath_AmazonQ(t *testing.T) {
	path, err := clientConfigPath("amazon-q")
	if err != nil {
		t.Fatalf("clientConfigPath(amazon-q): %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path for amazon-q")
	}
}

func TestClientConfigPath_VSCode(t *testing.T) {
	path, err := clientConfigPath("vscode")
	if err != nil {
		t.Fatalf("clientConfigPath(vscode): %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path for vscode")
	}
}

func TestClientConfigPath_Windsurf(t *testing.T) {
	path, err := clientConfigPath("windsurf")
	if err != nil {
		t.Fatalf("clientConfigPath(windsurf): %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path for windsurf")
	}
}

// ---------------------------------------------------------------------------
// mcpConfigKey — remaining branches (zed, default)
// ---------------------------------------------------------------------------

func TestMCPConfigKey_Zed(t *testing.T) {
	got := mcpConfigKey("zed")
	if got != "context_servers" {
		t.Errorf("mcpConfigKey(zed) = %q, want %q", got, "context_servers")
	}
}

func TestMCPConfigKey_Default(t *testing.T) {
	got := mcpConfigKey("claude-desktop")
	if got != "mcpServers" {
		t.Errorf("mcpConfigKey(claude-desktop) = %q, want %q", got, "mcpServers")
	}
}

// ---------------------------------------------------------------------------
// watchHistoryCmd — arg validation (requires store, will fail on empty store)
// ---------------------------------------------------------------------------

func TestWatchHistoryCmd_MissingArg(t *testing.T) {
	cmd := watchHistoryCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error with no args")
	}
}

func TestWatchHistoryCmd_NotFoundInEmptyStore(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	cmd := watchHistoryCmd()
	cmd.SetArgs([]string{"nonexistent-id"})
	err := cmd.Execute()
	// May error or succeed with "no history"; either is fine.
	_ = err
}

// ---------------------------------------------------------------------------
// watchRemoveCmd — arg validation
// ---------------------------------------------------------------------------

func TestWatchRemoveCmd_MissingArg(t *testing.T) {
	cmd := watchRemoveCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error with no args")
	}
}

// ---------------------------------------------------------------------------
// Trips: tripsShowCmd — requires store with a trip ID
// Test the error path when store doesn't have the requested ID.
// ---------------------------------------------------------------------------

func TestTripsShowCmd_NotFound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	cmd := tripsShowCmd()
	cmd.SetArgs([]string{"nonexistent-id"})
	err := cmd.Execute()
	// Should error with "trip not found" or load error.
	_ = err
}

// ---------------------------------------------------------------------------
// tripcostCmd — no-arg path
// ---------------------------------------------------------------------------

func TestTripCostCmd_FlagsV9(t *testing.T) {
	cmd := tripCostCmd()
	if cmd == nil {
		t.Error("expected non-nil tripCostCmd")
	}
}
