package testutil

import (
	"os"
	"strings"
	"testing"
)

const LiveIntegrationEnv = "TRVL_TEST_LIVE_INTEGRATIONS"

// RequireLiveIntegration keeps live-network integration tests out of the
// default suite while preserving an explicit opt-in for provider and MCP
// contract checks.
func RequireLiveIntegration(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping live integration test in short mode")
	}
	if strings.TrimSpace(os.Getenv(LiveIntegrationEnv)) == "" {
		t.Skip("set TRVL_TEST_LIVE_INTEGRATIONS=1 to run live integration tests")
	}
}
