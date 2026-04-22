package main

import (
	"os"
	"strings"
)

// appHomeDir prefers explicit test/runtime overrides before falling back to
// the OS-reported home directory.
func appHomeDir() string {
	for _, key := range []string{"HOME", "USERPROFILE"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}

	home, _ := os.UserHomeDir()
	return home
}
