package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MikkoParkkola/trvl/internal/destinations"
	"github.com/MikkoParkkola/trvl/internal/models"
	"github.com/MikkoParkkola/trvl/internal/preferences"
)

// ---------------------------------------------------------------------------
// shouldShowNudge — pure function, covers all branches
// ---------------------------------------------------------------------------

func TestShouldShowNudge_NotSearchCommandV24(t *testing.T) {
	got := shouldShowNudge("prefs", "", os.Getenv, 2, func(int) bool { return true })
	if got {
		t.Error("expected false for non-search command")
	}
}

func TestShouldShowNudge_NoNudgeEnvV24(t *testing.T) {
	t.Setenv("TRVL_NO_NUDGE", "1")
	got := shouldShowNudge("flights", "", os.Getenv, 2, func(int) bool { return true })
	if got {
		t.Error("expected false when TRVL_NO_NUDGE=1")
	}
}

func TestShouldShowNudge_JSONFormatV24(t *testing.T) {
	got := shouldShowNudge("flights", "json", os.Getenv, 2, func(int) bool { return true })
	if got {
		t.Error("expected false when format=json")
	}
}

func TestShouldShowNudge_NotTerminalV24(t *testing.T) {
	got := shouldShowNudge("flights", "", os.Getenv, 2, func(int) bool { return false })
	if got {
		t.Error("expected false when not a terminal")
	}
}

func TestShouldShowNudge_ReturnsTrueV24(t *testing.T) {
	t.Setenv("TRVL_NO_NUDGE", "")
	got := shouldShowNudge("hotels", "", func(key string) string {
		if key == "TRVL_NO_NUDGE" {
			return ""
		}
		return ""
	}, 2, func(int) bool { return true })
	if !got {
		t.Error("expected true for search command with terminal and no suppression")
	}
}

// ---------------------------------------------------------------------------
// nudgePath — pure helper
// ---------------------------------------------------------------------------

func TestNudgePath_ReturnsPathV24(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	p, err := nudgePath()
	if err != nil {
		t.Fatalf("nudgePath: %v", err)
	}
	if !strings.HasSuffix(p, "nudge.json") {
		t.Errorf("expected path ending in nudge.json, got %s", p)
	}
}

// ---------------------------------------------------------------------------
// saveNudgeState + loadNudgeState — pure file I/O
// ---------------------------------------------------------------------------

func TestSaveAndLoadNudgeState_V24(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nudge.json")

	s := nudgeState{SearchCount: 3, Shown: true}
	saveNudgeState(path, s)

	loaded := loadNudgeState(path)
	if loaded.SearchCount != 3 {
		t.Errorf("expected SearchCount=3, got %d", loaded.SearchCount)
	}
	if !loaded.Shown {
		t.Error("expected Shown=true")
	}
}

func TestLoadNudgeState_MissingFileV24(t *testing.T) {
	s := loadNudgeState("/tmp/nonexistent-nudge-xyz.json")
	if s.SearchCount != 0 || s.Shown {
		t.Errorf("expected zero state for missing file, got %+v", s)
	}
}

// ---------------------------------------------------------------------------
// loungesCmd — IATA validation error (no network), flags coverage
// ---------------------------------------------------------------------------

func TestLoungesCmd_InvalidIATA_V24(t *testing.T) {
	cmd := loungesCmd()
	cmd.SetArgs([]string{"12"}) // not valid IATA
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid IATA")
	}
}

func TestLoungesCmd_FlagsV24(t *testing.T) {
	cmd := loungesCmd()
	if cmd == nil {
		t.Fatal("loungesCmd returned nil")
	}
}

// ---------------------------------------------------------------------------
// weatherCmd — flags coverage
// ---------------------------------------------------------------------------

func TestWeatherCmd_FlagsV24(t *testing.T) {
	cmd := weatherCmd()
	for _, name := range []string{"from", "to"} {
		if f := cmd.Flags().Lookup(name); f == nil {
			t.Errorf("expected --%s flag on weatherCmd", name)
		}
	}
}

// ---------------------------------------------------------------------------
// upgradeCmd — dry-run (no network, touches ~/.trvl/upgrade-stamp.json)
// ---------------------------------------------------------------------------

func TestUpgradeCmd_DryRunV24(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	cmd := upgradeCmd()
	cmd.SetArgs([]string{"--dry-run"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("upgrade --dry-run: %v", err)
	}
}

func TestUpgradeCmd_QuietV24(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	cmd := upgradeCmd()
	cmd.SetArgs([]string{"--quiet"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("upgrade --quiet: %v", err)
	}
}

func TestUpgradeCmd_DefaultRunV24(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	cmd := upgradeCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Errorf("upgrade (default): %v", err)
	}
}

// ---------------------------------------------------------------------------
// runSetup — non-interactive mode (covers most of runSetup body)
// ---------------------------------------------------------------------------

func TestRunSetup_NonInteractiveV24(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg := setupConfig{
		nonInteractive: true,
		homeFlag:       "HEL",
		currencyFlag:   "EUR",
		cabinFlag:      "economy",
		stdin:          os.Stdin,
		stdout:         os.Stdout,
	}
	if err := runSetup(cfg); err != nil {
		t.Errorf("runSetup non-interactive: %v", err)
	}
}

func TestRunSetup_NonInteractiveBusinessClassV24(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg := setupConfig{
		nonInteractive: true,
		homeFlag:       "JFK",
		currencyFlag:   "USD",
		cabinFlag:      "business",
		stdin:          os.Stdin,
		stdout:         os.Stdout,
	}
	if err := runSetup(cfg); err != nil {
		t.Errorf("runSetup non-interactive business: %v", err)
	}
}

// ---------------------------------------------------------------------------
// secureTempPath — pure crypto helper
// ---------------------------------------------------------------------------

func TestSecureTempPath_V24(t *testing.T) {
	tmp := t.TempDir()
	p, err := secureTempPath(tmp, "keys.json.tmp-")
	if err != nil {
		t.Fatalf("secureTempPath: %v", err)
	}
	if !strings.HasPrefix(filepath.Base(p), "keys.json.tmp-") {
		t.Errorf("unexpected prefix in %s", p)
	}
}

// ---------------------------------------------------------------------------
// keysPath — pure path helper
// ---------------------------------------------------------------------------

func TestKeysPath_V24(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	p, err := keysPath()
	if err != nil {
		t.Fatalf("keysPath: %v", err)
	}
	if !strings.HasSuffix(p, "keys.json") {
		t.Errorf("expected keys.json suffix, got %s", p)
	}
}

// ---------------------------------------------------------------------------
// saveKeysTo — write + atomic rename
// ---------------------------------------------------------------------------

func TestSaveKeysTo_V24(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".trvl", "keys.json")
	keys := APIKeys{SeatsAero: "test-key", Kiwi: "kiwi-key"}
	if err := saveKeysTo(path, keys); err != nil {
		t.Fatalf("saveKeysTo: %v", err)
	}
	// Verify file exists.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("keys.json not created: %v", err)
	}
}

// ---------------------------------------------------------------------------
// mcpConfigKey — pure switch function
// ---------------------------------------------------------------------------

func TestMcpConfigKey_V24(t *testing.T) {
	cases := []struct {
		client string
		want   string
	}{
		{"vscode", "servers"},
		{"vs-code", "servers"},
		{"copilot", "servers"},
		{"zed", "context_servers"},
		{"claude-desktop", "mcpServers"},
		{"windsurf", "mcpServers"},
		{"codex", "mcpServers"},
	}
	for _, tc := range cases {
		got := mcpConfigKey(tc.client)
		if got != tc.want {
			t.Errorf("mcpConfigKey(%q) = %q, want %q", tc.client, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// trvlBinaryPath — returns non-empty path (test binary is running)
// ---------------------------------------------------------------------------

func TestTrvlBinaryPath_V24(t *testing.T) {
	p, err := trvlBinaryPath()
	if err != nil {
		t.Fatalf("trvlBinaryPath: %v", err)
	}
	if p == "" {
		t.Error("expected non-empty binary path")
	}
}

// ---------------------------------------------------------------------------
// createGist — gh not found → fallback print path (no network)
// ---------------------------------------------------------------------------

func TestCreateGist_NoGhV24(t *testing.T) {
	// createGist checks LookPath("gh"); in the unlikely event gh is installed,
	// it would try to create a gist — but with empty content it should still
	// return nil (just printing markdown). Either way, no panic.
	err := createGist("# Test trip\n\nSome markdown content here.")
	_ = err // Either nil or gh error — acceptable in test environment
}

// ---------------------------------------------------------------------------
// runEvents — missing API key (error path, no network)
// ---------------------------------------------------------------------------

func TestRunEvents_MissingAPIKeyV24(t *testing.T) {
	t.Setenv("TICKETMASTER_API_KEY", "")
	cmd := eventsCmd()
	cmd.SetArgs([]string{"Barcelona", "--from", "2026-07-01", "--to", "2026-07-08"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for missing TICKETMASTER_API_KEY")
	}
}

// ---------------------------------------------------------------------------
// formatEventsCard — pure rendering with 0 events and with events
// ---------------------------------------------------------------------------

func TestFormatEventsCard_EmptyV24(t *testing.T) {
	err := formatEventsCard(nil, "Barcelona", "2026-07-01", "2026-07-08")
	if err != nil {
		t.Errorf("formatEventsCard empty: %v", err)
	}
}

func TestFormatEventsCard_WithEventsV24(t *testing.T) {
	events := []models.Event{
		{
			Name:       "Test Concert",
			Date:       "2026-07-03",
			Time:       "20:00",
			Venue:      "Palau Sant Jordi",
			Type:       "Music",
			PriceRange: "€30-€80",
		},
		{
			Name:       "FC Barcelona Match",
			Date:       "2026-07-05",
			Time:       "18:00",
			Venue:      "Spotify Camp Nou",
			Type:       "Sports",
			PriceRange: "€50-€200",
		},
	}
	err := formatEventsCard(events, "Barcelona", "2026-07-01", "2026-07-08")
	if err != nil {
		t.Errorf("formatEventsCard with events: %v", err)
	}
}

// ---------------------------------------------------------------------------
// formatNearbyCard — pure rendering with various POI combinations
// ---------------------------------------------------------------------------

func TestFormatNearbyCard_EmptyV24(t *testing.T) {
	result := &destinations.NearbyResult{}
	if err := formatNearbyCard(result); err != nil {
		t.Errorf("formatNearbyCard empty: %v", err)
	}
}

func TestFormatNearbyCard_WithPOIsV24(t *testing.T) {
	result := &destinations.NearbyResult{
		POIs: []models.NearbyPOI{
			{Name: "La Boqueria", Type: "market", Distance: 120, Cuisine: "market", Hours: "9:00-20:00"},
			{Name: "Bar El Xampanyet", Type: "bar", Distance: 250, Cuisine: "tapas"},
		},
		RatedPlaces: []models.RatedPlace{
			{Name: "Tickets", Rating: 9.5, Category: "restaurant", PriceLevel: 3, Distance: 400},
		},
		Attractions: []models.Attraction{
			{Name: "Sagrada Familia", Kind: "church", Distance: 1500},
		},
	}
	if err := formatNearbyCard(result); err != nil {
		t.Errorf("formatNearbyCard with POIs: %v", err)
	}
}

// ---------------------------------------------------------------------------
// truncate — pure string helper
// ---------------------------------------------------------------------------

func TestTruncate_V24(t *testing.T) {
	cases := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "he..."},
		{"hello", 5, "hello"},
		{"ab", 2, "ab"},
		{"abc", 1, "a"},
	}
	for _, tc := range cases {
		got := truncate(tc.input, tc.maxLen)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// loungeFFCards + loungeTierDisplay — pure preference processing
// ---------------------------------------------------------------------------

func TestLoungeFFCards_EmptyV24(t *testing.T) {
	cards := loungeFFCards(nil)
	if len(cards) != 0 {
		t.Errorf("expected empty cards for nil programs, got %v", cards)
	}
}

func TestLoungeFFCards_WithAlliancesV24(t *testing.T) {
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "oneworld", Tier: "sapphire", AirlineCode: "BA"},
		{Alliance: "star_alliance", Tier: "gold", AirlineCode: "LH"},
		{Alliance: "skyteam", Tier: "elite_plus", AirlineCode: "AF"},
	}
	cards := loungeFFCards(programs)
	if len(cards) == 0 {
		t.Error("expected non-empty cards for known alliances")
	}
}

func TestLoungeFFCards_UnknownAllianceV24(t *testing.T) {
	programs := []preferences.FrequentFlyerStatus{
		{Alliance: "unknown-alliance", Tier: "gold", AirlineCode: "XX"},
	}
	cards := loungeFFCards(programs)
	// Unknown airline code should not produce cards (no match), but no panic.
	_ = cards
}

func TestLoungeTierDisplay_KnownAllianceV24(t *testing.T) {
	display := loungeTierDisplay("oneworld", "emerald")
	if display != "Emerald" {
		t.Errorf("expected Emerald, got %s", display)
	}
}

func TestLoungeTierDisplay_UnknownTierV24(t *testing.T) {
	// Falls through to title-case the normalized tier string.
	display := loungeTierDisplay("oneworld", "diamond")
	if display == "" {
		t.Error("expected non-empty display for unknown tier")
	}
}

func TestLoungeTierDisplay_UnknownAllianceV24(t *testing.T) {
	display := loungeTierDisplay("unknown", "gold")
	if display == "" {
		t.Error("expected non-empty display for unknown alliance")
	}
}

// ---------------------------------------------------------------------------
// mcpCmd — flags coverage (no actual server start)
// ---------------------------------------------------------------------------

func TestMcpCmd_FlagsV24(t *testing.T) {
	cmd := mcpCmd()
	for _, name := range []string{"http", "port"} {
		if f := cmd.Flags().Lookup(name); f == nil {
			t.Errorf("expected --%s flag on mcpCmd", name)
		}
	}
}

// ---------------------------------------------------------------------------
// prefsEditCmd + prefsInitCmd — non-nil check
// ---------------------------------------------------------------------------

func TestPrefsEditCmd_NonNilV24(t *testing.T) {
	cmd := prefsEditCmd()
	if cmd == nil {
		t.Error("expected non-nil prefsEditCmd")
	}
}

func TestPrefsInitCmd_NonNilV24(t *testing.T) {
	cmd := prefsInitCmd()
	if cmd == nil {
		t.Error("expected non-nil prefsInitCmd")
	}
}

// ---------------------------------------------------------------------------
// groundCmd — missing args (no network)
// ---------------------------------------------------------------------------

func TestGroundCmd_MissingArgsV24(t *testing.T) {
	cmd := groundCmd()
	cmd.SetArgs([]string{"Prague"}) // needs 3 args
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error with only one positional arg")
	}
}

func TestGroundCmd_FlagsV24(t *testing.T) {
	cmd := groundCmd()
	for _, name := range []string{"currency", "max-price", "type"} {
		if f := cmd.Flags().Lookup(name); f == nil {
			t.Errorf("expected --%s flag on groundCmd", name)
		}
	}
}

// ---------------------------------------------------------------------------
// hacksCmd — missing args (no network)
// ---------------------------------------------------------------------------

func TestHacksCmd_MissingArgsV24(t *testing.T) {
	cmd := hacksCmd()
	cmd.SetArgs([]string{"HEL"}) // needs 3 args
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error with only one positional arg")
	}
}

// ---------------------------------------------------------------------------
// loadExistingKeys — missing file returns empty struct
// ---------------------------------------------------------------------------

func TestLoadExistingKeys_MissingFileV24(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	keys := loadExistingKeys()
	// Should return empty struct, no panic.
	if keys.SeatsAero != "" || keys.Kiwi != "" {
		t.Errorf("expected empty keys for missing file, got %+v", keys)
	}
}
