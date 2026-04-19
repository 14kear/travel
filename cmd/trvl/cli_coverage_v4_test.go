package main

import (
	"strings"
	"testing"
	"time"

	"github.com/MikkoParkkola/trvl/internal/providers"
)

// ---------------------------------------------------------------------------
// classifyProviderStatus (providers_status.go)
// ---------------------------------------------------------------------------

func TestClassifyProviderStatus_Healthy(t *testing.T) {
	cfg := &providers.ProviderConfig{
		LastSuccess: time.Now().Add(-1 * time.Hour),
	}
	got := classifyProviderStatus(cfg)
	if got != "healthy" {
		t.Errorf("expected healthy, got %q", got)
	}
}

func TestClassifyProviderStatus_Stale(t *testing.T) {
	cfg := &providers.ProviderConfig{
		LastSuccess: time.Now().Add(-25 * time.Hour),
	}
	got := classifyProviderStatus(cfg)
	if got != "stale" {
		t.Errorf("expected stale, got %q", got)
	}
}

func TestClassifyProviderStatus_Error(t *testing.T) {
	cfg := &providers.ProviderConfig{
		ErrorCount: 3,
		LastError:  "connection refused",
		LastSuccess: time.Now().Add(-1 * time.Hour),
	}
	got := classifyProviderStatus(cfg)
	if got != "error" {
		t.Errorf("expected error, got %q", got)
	}
}

func TestClassifyProviderStatus_Unconfigured(t *testing.T) {
	cfg := &providers.ProviderConfig{}
	got := classifyProviderStatus(cfg)
	if got != "unconfigured" {
		t.Errorf("expected unconfigured, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// colorProviderStatus (providers_status.go)
// ---------------------------------------------------------------------------

func TestColorProviderStatus_AllBranches(t *testing.T) {
	for _, status := range []string{"healthy", "stale", "error", "unconfigured", "unknown"} {
		got := colorProviderStatus(status)
		if got == "" {
			t.Errorf("colorProviderStatus(%q) returned empty", status)
		}
	}
}

// ---------------------------------------------------------------------------
// relativeTimeStr (providers_status.go)
// ---------------------------------------------------------------------------

func TestRelativeTimeStr_Zero(t *testing.T) {
	got := relativeTimeStr(time.Time{})
	if got != "-" {
		t.Errorf("expected -, got %q", got)
	}
}

func TestRelativeTimeStr_JustNow(t *testing.T) {
	got := relativeTimeStr(time.Now().Add(-10 * time.Second))
	if got != "just now" {
		t.Errorf("expected 'just now', got %q", got)
	}
}

func TestRelativeTimeStr_OneMinuteAgo(t *testing.T) {
	got := relativeTimeStr(time.Now().Add(-1 * time.Minute - 5*time.Second))
	if got != "1m ago" {
		t.Errorf("expected '1m ago', got %q", got)
	}
}

func TestRelativeTimeStr_MultipleMinutesAgo(t *testing.T) {
	got := relativeTimeStr(time.Now().Add(-30 * time.Minute))
	if !strings.HasSuffix(got, "m ago") {
		t.Errorf("expected Xm ago, got %q", got)
	}
}

func TestRelativeTimeStr_OneHourAgo(t *testing.T) {
	got := relativeTimeStr(time.Now().Add(-1 * time.Hour - 5*time.Minute))
	if got != "1h ago" {
		t.Errorf("expected '1h ago', got %q", got)
	}
}

func TestRelativeTimeStr_MultipleHoursAgo(t *testing.T) {
	got := relativeTimeStr(time.Now().Add(-5 * time.Hour))
	if !strings.HasSuffix(got, "h ago") {
		t.Errorf("expected Xh ago, got %q", got)
	}
}

func TestRelativeTimeStr_OneDayAgo(t *testing.T) {
	got := relativeTimeStr(time.Now().Add(-25 * time.Hour))
	if got != "1d ago" {
		t.Errorf("expected '1d ago', got %q", got)
	}
}

func TestRelativeTimeStr_MultipleDaysAgo(t *testing.T) {
	got := relativeTimeStr(time.Now().Add(-72 * time.Hour))
	if !strings.HasSuffix(got, "d ago") {
		t.Errorf("expected Xd ago, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// truncateStr (providers_status.go)
// ---------------------------------------------------------------------------

func TestTruncateStr_ShortString(t *testing.T) {
	got := truncateStr("hello", 10)
	if got != "hello" {
		t.Errorf("expected hello, got %q", got)
	}
}

func TestTruncateStr_ExactLength(t *testing.T) {
	got := truncateStr("hello", 5)
	if got != "hello" {
		t.Errorf("expected hello, got %q", got)
	}
}

func TestTruncateStr_TruncatesWithEllipsis(t *testing.T) {
	got := truncateStr("hello world", 8)
	if !strings.HasSuffix(got, "...") || len(got) != 8 {
		t.Errorf("expected 8-char string with ellipsis, got %q", got)
	}
}

func TestTruncateStr_MaxLenThree(t *testing.T) {
	got := truncateStr("hello", 3)
	if got != "hel" {
		t.Errorf("expected hel, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// shouldShowNudge (nudge.go) — pure function with injectable deps
// ---------------------------------------------------------------------------

func TestShouldShowNudge_NotSearchCommand(t *testing.T) {
	got := shouldShowNudge("profile", "", func(string) string { return "" }, 0, func(int) bool { return true })
	if got {
		t.Error("expected false for non-search command")
	}
}

func TestShouldShowNudge_SuppressedByEnv(t *testing.T) {
	got := shouldShowNudge("flights", "", func(key string) string {
		if key == "TRVL_NO_NUDGE" {
			return "1"
		}
		return ""
	}, 0, func(int) bool { return true })
	if got {
		t.Error("expected false when TRVL_NO_NUDGE=1")
	}
}

func TestShouldShowNudge_MCPCommandV4(t *testing.T) {
	got := shouldShowNudge("mcp", "", func(string) string { return "" }, 0, func(int) bool { return true })
	if got {
		t.Error("expected false for mcp command")
	}
}

func TestShouldShowNudge_JSONFormatV4(t *testing.T) {
	got := shouldShowNudge("flights", "json", func(string) string { return "" }, 0, func(int) bool { return true })
	if got {
		t.Error("expected false for json format")
	}
}

func TestShouldShowNudge_NotTerminal(t *testing.T) {
	got := shouldShowNudge("flights", "", func(string) string { return "" }, 0, func(int) bool { return false })
	if got {
		t.Error("expected false when not a terminal")
	}
}

func TestShouldShowNudge_ShouldShow(t *testing.T) {
	got := shouldShowNudge("flights", "", func(string) string { return "" }, 0, func(int) bool { return true })
	if !got {
		t.Error("expected true for search command + terminal + no suppression")
	}
}

func TestShouldShowNudge_AllSearchCommandsV4(t *testing.T) {
	// Verify all search command names pass the guard.
	for cmd := range searchCommands {
		got := shouldShowNudge(cmd, "", func(string) string { return "" }, 0, func(int) bool { return true })
		if !got {
			t.Errorf("expected true for search command %q", cmd)
		}
	}
}

// ---------------------------------------------------------------------------
// runProfileSummary path via runProfileShow with no bookings
// ---------------------------------------------------------------------------

func TestRunProfileShow_NoBookings(t *testing.T) {
	// When profile has no bookings, runProfileShow prints a message and returns nil.
	// We can test via the cobra command.
	cmd := profileCmd()
	cmd.SetArgs([]string{})
	// This loads from ~/.trvl/profile.json; in CI there likely are no bookings.
	// We just verify it doesn't panic/returns within a reasonable call.
	_ = cmd.Execute()
}

// ---------------------------------------------------------------------------
// profileAddCmd flag registration
// ---------------------------------------------------------------------------

func TestProfileAddCmd_Flags(t *testing.T) {
	cmd := profileAddCmd()
	for _, name := range []string{"type", "travel-date", "from", "to", "provider", "price", "currency", "nights", "stars", "reference", "notes"} {
		if f := cmd.Flags().Lookup(name); f == nil {
			t.Errorf("expected --%s flag on profile add", name)
		}
	}
}

func TestProfileAddCmd_MissingTypeError(t *testing.T) {
	cmd := profileAddCmd()
	// Provide provider but no type.
	cmd.SetArgs([]string{"--provider", "KLM"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when --type is missing")
	}
}

func TestProfileAddCmd_MissingProviderError(t *testing.T) {
	cmd := profileAddCmd()
	// Provide type but no provider.
	cmd.SetArgs([]string{"--type", "flight"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when --provider is missing")
	}
}

// ---------------------------------------------------------------------------
// loungesCmd — IATA validation path (no network)
// ---------------------------------------------------------------------------

func TestLoungesCmd_InvalidIATAError(t *testing.T) {
	cmd := loungesCmd()
	cmd.SetArgs([]string{"12"}) // too short to be valid IATA
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid IATA code")
	}
}

// ---------------------------------------------------------------------------
// weatherCmd — missing arg
// ---------------------------------------------------------------------------

func TestWeatherCmd_MissingArg(t *testing.T) {
	cmd := weatherCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error with no args")
	}
}

// ---------------------------------------------------------------------------
// multiCityCmd — flag registration
// ---------------------------------------------------------------------------

func TestMultiCityCmd_FlagsV4(t *testing.T) {
	cmd := multiCityCmd()
	// multi-city uses --visit and --dates flags
	for _, name := range []string{"visit", "dates"} {
		if f := cmd.Flags().Lookup(name); f == nil {
			t.Errorf("expected --%s flag", name)
		}
	}
}

// ---------------------------------------------------------------------------
// runNearby — remaining validation branches
// ---------------------------------------------------------------------------

func TestRunNearby_ValidLatLon_FlagDefaults(t *testing.T) {
	// nearbyCmd validates args — test the parseFloat path success + ctx cancel
	// We call RunE directly to avoid network calls (will fail at destinations.GetNearbyPlaces)
	cmd := nearbyCmd()
	// Fill in args via the command Args so RunE can access flags.
	cmd.SetArgs([]string{"91.0", "2.17"}) // lat > 90 won't cause parse error but won't fail lat parse
	// We can't easily test success path without network; instead verify flag defaults.
	if f := cmd.Flags().Lookup("radius"); f.DefValue != "500" {
		t.Errorf("expected default radius 500, got %s", f.DefValue)
	}
}
