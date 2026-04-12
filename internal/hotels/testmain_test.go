package hotels

import (
	"os"
	"testing"
)

// TestMain runs before all tests in the hotels package. It disables live
// Trivago HTTP calls so that unit/integration tests that mock the Google
// Hotels transport do not accidentally fire real requests to mcp.trivago.com.
// Individual Trivago tests that need live or mock-server calls restore the
// flag themselves (or use their own mock transport).
func TestMain(m *testing.M) {
	trivagoEnabled = false
	os.Exit(m.Run())
}
