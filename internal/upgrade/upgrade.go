// Package upgrade manages post-upgrade migrations and version stamping.
//
// The version stamp lives at ~/.trvl/version.stamp and contains the version
// string of the last binary that ran successfully. On startup, CheckUpgrade
// compares the stamp to the running binary's version and applies any
// registered migrations whose FromVersion falls between the two.
package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Migration describes a single post-upgrade step.
type Migration struct {
	FromVersion string // applies when upgrading past this version
	Description string
	Apply       func() error
}

// migrations is the global migration registry. Add entries here when a
// release requires post-upgrade fixups.
var migrations []Migration

// RegisterMigration appends a migration to the registry. Migrations are
// executed in registration order.
func RegisterMigration(m Migration) {
	migrations = append(migrations, m)
}

// resetMigrations clears the registry (test-only).
func resetMigrations() {
	migrations = nil
}

// trvlDir returns the ~/.trvl directory path.
func trvlDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".trvl"), nil
}

// stampPath returns the full path to the version stamp file.
func stampPath() (string, error) {
	dir, err := trvlDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "version.stamp"), nil
}

// stampPathIn returns the stamp path inside a given directory.
func stampPathIn(dir string) string {
	return filepath.Join(dir, "version.stamp")
}

// prefsPathIn returns the preferences.json path inside a given directory.
func prefsPathIn(dir string) string {
	return filepath.Join(dir, "preferences.json")
}

// ReadStamp reads the version stamp file. Returns "" if the file does not exist.
func ReadStamp(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read version stamp: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// WriteStamp writes a version string to the stamp file.
func WriteStamp(path, version string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create stamp dir: %w", err)
	}
	return os.WriteFile(path, []byte(version+"\n"), 0o644)
}

// Result holds the outcome of a CheckUpgrade or RunUpgrade call.
type Result struct {
	OldVersion        string
	NewVersion        string
	MigrationsApplied int
	FreshInstall      bool
	Downgrade         bool
}

// CheckUpgrade reads the stamp, compares to currentVersion, runs applicable
// migrations, prints what's-new info, and updates the stamp. It is safe to
// call on every startup.
//
// dir is the ~/.trvl directory. If empty, the default is used.
func CheckUpgrade(currentVersion, dir string) (*Result, error) {
	if currentVersion == "" || currentVersion == "dev" {
		return &Result{NewVersion: currentVersion}, nil
	}

	if dir == "" {
		d, err := trvlDir()
		if err != nil {
			return nil, err
		}
		dir = d
	}

	sp := stampPathIn(dir)
	old, err := ReadStamp(sp)
	if err != nil {
		return nil, err
	}

	// Fresh install: no stamp file.
	if old == "" {
		if err := WriteStamp(sp, currentVersion); err != nil {
			return nil, err
		}
		return &Result{
			NewVersion:   currentVersion,
			FreshInstall: true,
		}, nil
	}

	cmp := CompareSemver(old, currentVersion)

	if cmp == 0 {
		// Same version — no-op.
		return &Result{
			OldVersion: old,
			NewVersion: currentVersion,
		}, nil
	}

	if cmp > 0 {
		// Downgrade — warn but don't touch stamp.
		return &Result{
			OldVersion: old,
			NewVersion: currentVersion,
			Downgrade:  true,
		}, nil
	}

	// Upgrade path: old < current.
	return runMigrations(dir, sp, old, currentVersion, false)
}

// RunUpgrade is the explicit upgrade path (called by `trvl upgrade`).
// When dryRun is true, no stamp is written and migrations are not applied.
func RunUpgrade(currentVersion, dir string, dryRun bool) (*Result, error) {
	if currentVersion == "" || currentVersion == "dev" {
		return &Result{NewVersion: currentVersion}, nil
	}

	if dir == "" {
		d, err := trvlDir()
		if err != nil {
			return nil, err
		}
		dir = d
	}

	sp := stampPathIn(dir)
	old, err := ReadStamp(sp)
	if err != nil {
		return nil, err
	}

	if old == "" {
		if !dryRun {
			if err := WriteStamp(sp, currentVersion); err != nil {
				return nil, err
			}
		}
		return &Result{
			NewVersion:   currentVersion,
			FreshInstall: true,
		}, nil
	}

	cmp := CompareSemver(old, currentVersion)
	if cmp == 0 {
		return &Result{OldVersion: old, NewVersion: currentVersion}, nil
	}
	if cmp > 0 {
		return &Result{OldVersion: old, NewVersion: currentVersion, Downgrade: true}, nil
	}

	return runMigrations(dir, sp, old, currentVersion, dryRun)
}

// runMigrations backs up preferences, runs applicable migrations, and writes
// the new stamp (unless dryRun).
func runMigrations(dir, sp, old, current string, dryRun bool) (*Result, error) {
	// Backup preferences before migration.
	if !dryRun {
		backupPreferences(dir, old)
	}

	applicable := applicableMigrations(old, current)
	if !dryRun {
		for _, m := range applicable {
			if err := m.Apply(); err != nil {
				return nil, fmt.Errorf("migration %q failed: %w", m.Description, err)
			}
		}
		if err := WriteStamp(sp, current); err != nil {
			return nil, err
		}
	}

	return &Result{
		OldVersion:        old,
		NewVersion:        current,
		MigrationsApplied: len(applicable),
	}, nil
}

// applicableMigrations returns migrations whose FromVersion is >= old and < current.
func applicableMigrations(old, current string) []Migration {
	var out []Migration
	for _, m := range migrations {
		// A migration applies when its FromVersion is >= old version
		// (i.e., the user hasn't run it yet) and < current version.
		if CompareSemver(m.FromVersion, old) >= 0 && CompareSemver(m.FromVersion, current) < 0 {
			out = append(out, m)
		}
	}
	return out
}

// backupPreferences copies preferences.json to preferences.json.bak.{version}
// if it exists.
func backupPreferences(dir, oldVersion string) {
	src := prefsPathIn(dir)
	if _, err := os.Stat(src); err != nil {
		return // no prefs file, nothing to back up
	}
	dst := src + ".bak." + oldVersion
	data, err := os.ReadFile(src)
	if err != nil {
		return
	}
	_ = os.WriteFile(dst, data, 0o600)
}

// WhatsNew returns the "what's new" message for an upgrade result.
func WhatsNew(r *Result) string {
	if r.FreshInstall {
		return fmt.Sprintf("Welcome to trvl %s.", r.NewVersion)
	}
	if r.Downgrade {
		return fmt.Sprintf("Warning: running older version %s (stamp is %s). Stamp not modified.", r.NewVersion, r.OldVersion)
	}
	if r.OldVersion == "" || r.OldVersion == r.NewVersion {
		return ""
	}
	return fmt.Sprintf("Upgraded from %s to %s. %d migrations applied.", r.OldVersion, r.NewVersion, r.MigrationsApplied)
}

// --- semver comparison ---

// CompareSemver compares two semver strings (with optional "v" prefix).
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Non-semver strings are compared lexicographically as a fallback.
func CompareSemver(a, b string) int {
	am, ami, ap := parseSemver(a)
	bm, bmi, bp := parseSemver(b)

	if am == nil || bm == nil {
		// Fallback: lexicographic.
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}

	for i := 0; i < 3; i++ {
		if am[i] < bm[i] {
			return -1
		}
		if am[i] > bm[i] {
			return 1
		}
	}

	// Equal numeric parts — compare pre-release.
	// A version without pre-release has higher precedence than one with.
	if ap == "" && bp == "" {
		return 0
	}
	if ap == "" && bp != "" {
		return 1 // release > pre-release
	}
	if ap != "" && bp == "" {
		return -1
	}
	// Both have pre-release: compare identifiers per semver spec.
	return comparePreRelease(ap, bp, ami, bmi)
}

// parseSemver extracts [major, minor, patch] and pre-release from a version string.
// Returns nil parts if parsing fails.
func parseSemver(s string) (parts []int, identifiers []string, preRelease string) {
	s = strings.TrimPrefix(s, "v")

	// Split off pre-release (e.g., "1.2.3-beta.1").
	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		preRelease = s[idx+1:]
		s = s[:idx]
	}

	// Split off build metadata (ignored for comparison).
	if idx := strings.IndexByte(s, '+'); idx >= 0 {
		s = s[:idx]
	}

	segs := strings.SplitN(s, ".", 3)
	if len(segs) != 3 {
		return nil, nil, ""
	}

	parts = make([]int, 3)
	for i, seg := range segs {
		n, err := strconv.Atoi(seg)
		if err != nil {
			return nil, nil, ""
		}
		parts[i] = n
	}

	if preRelease != "" {
		identifiers = strings.Split(preRelease, ".")
	}

	return parts, identifiers, preRelease
}

// comparePreRelease compares two pre-release strings per the semver spec.
func comparePreRelease(a, b string, aIDs, bIDs []string) int {
	max := len(aIDs)
	if len(bIDs) < max {
		max = len(bIDs)
	}

	for i := 0; i < max; i++ {
		ai, aIsNum := strconv.Atoi(aIDs[i])
		bi, bIsNum := strconv.Atoi(bIDs[i])

		switch {
		case aIsNum == nil && bIsNum == nil:
			// Both numeric — compare as integers.
			if ai < bi {
				return -1
			}
			if ai > bi {
				return 1
			}
		case aIsNum == nil && bIsNum != nil:
			// Numeric < string.
			return -1
		case aIsNum != nil && bIsNum == nil:
			return 1
		default:
			// Both strings — lexicographic.
			if aIDs[i] < bIDs[i] {
				return -1
			}
			if aIDs[i] > bIDs[i] {
				return 1
			}
		}
	}

	// Shorter set has lower precedence.
	if len(aIDs) < len(bIDs) {
		return -1
	}
	if len(aIDs) > len(bIDs) {
		return 1
	}
	return 0
}
