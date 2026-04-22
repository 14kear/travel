package flights

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type storedFlightKeys struct {
	Duffel string `json:"duffel,omitempty"`
}

func loadStoredFlightKeys() storedFlightKeys {
	home, err := os.UserHomeDir()
	if err != nil {
		return storedFlightKeys{}
	}

	path := filepath.Join(home, ".trvl", "keys.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return storedFlightKeys{}
	}

	var keys storedFlightKeys
	if err := json.Unmarshal(data, &keys); err != nil {
		return storedFlightKeys{}
	}
	return keys
}

func loadDuffelToken() string {
	for _, envKey := range []string{"DUFFEL_API_TOKEN", "DUFFEL_TEST_TOKEN"} {
		if token := strings.TrimSpace(os.Getenv(envKey)); token != "" {
			return token
		}
	}

	return strings.TrimSpace(loadStoredFlightKeys().Duffel)
}

func hasDuffelToken() bool {
	return loadDuffelToken() != ""
}

func duffelSearchEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("TRVL_ENABLE_DUFFEL")))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
