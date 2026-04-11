package testutil

import (
	"os"
	"strings"
	"testing"
)

const LiveProbeEnv = "TRVL_TEST_LIVE_PROBES"

// RequireLiveProbe keeps live-network probe tests out of the default unit path
// while preserving an explicit opt-in for wire-format verification work.
func RequireLiveProbe(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping live probe in short mode")
	}
	if strings.TrimSpace(os.Getenv(LiveProbeEnv)) == "" {
		t.Skip("set TRVL_TEST_LIVE_PROBES=1 to run live probe tests")
	}
}
