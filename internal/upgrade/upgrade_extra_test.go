package upgrade

import (
	"testing"
)

// --- comparePreRelease ---

func TestComparePreRelease_BothNumeric(t *testing.T) {
	tests := []struct {
		aIDs, bIDs []string
		want       int
	}{
		{[]string{"1"}, []string{"2"}, -1},
		{[]string{"2"}, []string{"1"}, 1},
		{[]string{"1"}, []string{"1"}, 0},
		{[]string{"10"}, []string{"9"}, 1}, // numeric, not lexicographic
	}
	for _, tt := range tests {
		got := comparePreRelease("", "", tt.aIDs, tt.bIDs)
		if got != tt.want {
			t.Errorf("comparePreRelease(%v, %v) = %d, want %d", tt.aIDs, tt.bIDs, got, tt.want)
		}
	}
}

func TestComparePreRelease_NumericVsString(t *testing.T) {
	// Numeric < string per semver spec.
	got := comparePreRelease("", "", []string{"1"}, []string{"alpha"})
	if got != -1 {
		t.Errorf("expected numeric < string, got %d", got)
	}
	got = comparePreRelease("", "", []string{"alpha"}, []string{"1"})
	if got != 1 {
		t.Errorf("expected string > numeric, got %d", got)
	}
}

func TestComparePreRelease_BothStrings_Lex(t *testing.T) {
	tests := []struct {
		aIDs, bIDs []string
		want       int
	}{
		{[]string{"alpha"}, []string{"beta"}, -1},
		{[]string{"beta"}, []string{"alpha"}, 1},
		{[]string{"alpha"}, []string{"alpha"}, 0},
	}
	for _, tt := range tests {
		got := comparePreRelease("", "", tt.aIDs, tt.bIDs)
		if got != tt.want {
			t.Errorf("comparePreRelease(%v, %v) = %d, want %d", tt.aIDs, tt.bIDs, got, tt.want)
		}
	}
}

func TestComparePreRelease_DifferentLengths(t *testing.T) {
	// Shorter has lower precedence.
	got := comparePreRelease("", "", []string{"alpha"}, []string{"alpha", "1"})
	if got != -1 {
		t.Errorf("expected shorter to be less, got %d", got)
	}
	got = comparePreRelease("", "", []string{"alpha", "1"}, []string{"alpha"})
	if got != 1 {
		t.Errorf("expected longer to be greater, got %d", got)
	}
}

func TestComparePreRelease_Empty(t *testing.T) {
	got := comparePreRelease("", "", []string{}, []string{})
	if got != 0 {
		t.Errorf("expected 0 for both empty, got %d", got)
	}
}

// --- RunUpgrade ---

func TestRunUpgrade_DevVersion(t *testing.T) {
	result, err := RunUpgrade("dev", t.TempDir(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.NewVersion != "dev" {
		t.Errorf("expected dev version, got %q", result.NewVersion)
	}
}

func TestRunUpgrade_EmptyVersion(t *testing.T) {
	result, err := RunUpgrade("", t.TempDir(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestRunUpgrade_FreshInstall(t *testing.T) {
	dir := t.TempDir()
	result, err := RunUpgrade("1.0.0", dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.FreshInstall {
		t.Error("expected FreshInstall=true")
	}
	if result.NewVersion != "1.0.0" {
		t.Errorf("expected 1.0.0, got %q", result.NewVersion)
	}
}

func TestRunUpgrade_SameVersion(t *testing.T) {
	dir := t.TempDir()
	// First run — fresh install.
	_, err := RunUpgrade("1.0.0", dir, false)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}

	// Second run — same version, no migration.
	result, err := RunUpgrade("1.0.0", dir, false)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if result.FreshInstall {
		t.Error("expected FreshInstall=false on second run")
	}
	if result.Downgrade {
		t.Error("expected no downgrade")
	}
}

func TestRunUpgrade_Upgrade(t *testing.T) {
	dir := t.TempDir()
	// Start at 1.0.0.
	_, err := RunUpgrade("1.0.0", dir, false)
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	// Upgrade to 1.1.0.
	result, err := RunUpgrade("1.1.0", dir, false)
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if result.OldVersion != "1.0.0" {
		t.Errorf("expected OldVersion=1.0.0, got %q", result.OldVersion)
	}
	if result.NewVersion != "1.1.0" {
		t.Errorf("expected NewVersion=1.1.0, got %q", result.NewVersion)
	}
	if result.Downgrade {
		t.Error("expected no downgrade")
	}
}

func TestRunUpgrade_Downgrade(t *testing.T) {
	dir := t.TempDir()
	// Start at 2.0.0.
	_, err := RunUpgrade("2.0.0", dir, false)
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	// Try running with 1.0.0 (downgrade).
	result, err := RunUpgrade("1.0.0", dir, false)
	if err != nil {
		t.Fatalf("downgrade check: %v", err)
	}
	if !result.Downgrade {
		t.Error("expected Downgrade=true")
	}
}

func TestRunUpgrade_DryRunIdempotent(t *testing.T) {
	dir := t.TempDir()
	// dryRun should not write stamp.
	result, err := RunUpgrade("1.0.0", dir, true)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if !result.FreshInstall {
		t.Error("expected FreshInstall in dry run (no stamp written)")
	}

	// Run again — should still see fresh install (stamp not persisted).
	result2, err := RunUpgrade("1.0.0", dir, true)
	if err != nil {
		t.Fatalf("second dry run: %v", err)
	}
	if !result2.FreshInstall {
		t.Error("expected FreshInstall on second dry run (stamp never written)")
	}
}

// --- RunUpgrade_DryRunDontWrite verifies stamp not written ---

func TestRunUpgrade_DryRunNoStamp(t *testing.T) {
	dir := t.TempDir()
	sp := stampPathIn(dir)

	// dryRun: should not write stamp file.
	_, err := RunUpgrade("1.0.0", dir, true)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}

	// Stamp file should not exist.
	v, err := ReadStamp(sp)
	if err != nil {
		t.Fatalf("ReadStamp: %v", err)
	}
	if v != "" {
		t.Errorf("expected no stamp after dry run, got %q", v)
	}
}
