package route

import "time"

// MinConnectionTime returns the minimum transfer time between two transport modes.
// For same-mode connections (train→train), shorter times are allowed.
// For cross-mode connections involving flights, longer times account for
// airport transit, check-in, and baggage claim.
func MinConnectionTime(prevMode, nextMode string) time.Duration {
	key := prevMode + "→" + nextMode
	if d, ok := connectionTimes[key]; ok {
		return d
	}
	// Default: 60 minutes for unknown combinations.
	return 60 * time.Minute
}

// connectionTimes maps "prevMode→nextMode" to minimum connection durations.
var connectionTimes = map[string]time.Duration{
	// Same mode
	"flight→flight": 120 * time.Minute, // self-transfer at hub airport
	"train→train":   30 * time.Minute,
	"bus→bus":        30 * time.Minute,
	"ferry→ferry":   60 * time.Minute,

	// Ground to flight (need airport transit + check-in)
	"train→flight": 120 * time.Minute,
	"bus→flight":   120 * time.Minute,
	"ferry→flight": 120 * time.Minute,

	// Flight to ground (baggage claim + transit to station)
	"flight→train": 120 * time.Minute,
	"flight→bus":   120 * time.Minute,
	"flight→ferry": 150 * time.Minute,

	// Ground cross-mode
	"train→bus":   30 * time.Minute,
	"bus→train":   30 * time.Minute,
	"train→ferry": 60 * time.Minute,
	"ferry→train": 60 * time.Minute,
	"bus→ferry":   60 * time.Minute,
	"ferry→bus":   60 * time.Minute,
}

// timeLayouts are the datetime formats used by various providers.
var timeLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
}

// parseFlexTime parses a datetime string using multiple layouts.
func parseFlexTime(s string) (time.Time, bool) {
	for _, layout := range timeLayouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// IsConnectionFeasible checks if there is enough time between two legs
// for a transfer, given the transport modes.
func IsConnectionFeasible(arriveTime, departTime string, prevMode, nextMode string) bool {
	arrive, ok := parseFlexTime(arriveTime)
	if !ok {
		return false
	}
	depart, ok := parseFlexTime(departTime)
	if !ok {
		return false
	}
	return depart.Sub(arrive) >= MinConnectionTime(prevMode, nextMode)
}
