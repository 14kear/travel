package trips

import (
	"context"
	"fmt"
	"time"
)

// Monitor checks all booked/in-progress trips for notable events and returns
// any new alerts.  It does not persist the alerts — callers are responsible
// for calling Store.AddAlert for each returned alert.
//
// Currently implemented checks:
//  1. Time-based reminders: "Departure in 24h", "Check in opens in 6h"
//  2. Placeholder stubs for price-drop and weather (not live — require
//     calling into flight/weather providers).
func Monitor(_ context.Context, trips []Trip) []Alert {
	now := time.Now()
	var alerts []Alert

	for _, t := range trips {
		if !activeStatuses[t.Status] {
			continue
		}
		alerts = append(alerts, checkReminders(t, now)...)
	}
	return alerts
}

// checkReminders generates time-based alerts for a trip.
func checkReminders(t Trip, now time.Time) []Alert {
	var alerts []Alert

	for _, leg := range t.Legs {
		if leg.StartTime == "" {
			continue
		}
		start, err := parseDateTime(leg.StartTime)
		if err != nil {
			continue
		}

		until := start.Sub(now)
		if until < 0 {
			continue // already departed
		}

		switch {
		case within(until, 24*time.Hour, 30*time.Minute):
			alerts = append(alerts, Alert{
				TripID:   t.ID,
				TripName: t.Name,
				Type:     "reminder",
				Message:  fmt.Sprintf("Departure in ~24h: %s %s->%s at %s", leg.Provider, leg.From, leg.To, start.Format("15:04")),
			})
		case within(until, 6*time.Hour, 30*time.Minute):
			alerts = append(alerts, Alert{
				TripID:   t.ID,
				TripName: t.Name,
				Type:     "reminder",
				Message:  fmt.Sprintf("Check-in opens in ~6h: %s %s->%s at %s", leg.Provider, leg.From, leg.To, start.Format("15:04")),
			})
		case within(until, 2*time.Hour, 15*time.Minute):
			alerts = append(alerts, Alert{
				TripID:   t.ID,
				TripName: t.Name,
				Type:     "reminder",
				Message:  fmt.Sprintf("Departing in ~2h: %s %s->%s", leg.Provider, leg.From, leg.To),
			})
		}
	}
	return alerts
}

// within returns true when d is in the range [target-tolerance, target+tolerance].
func within(d, target, tolerance time.Duration) bool {
	return d >= target-tolerance && d <= target+tolerance
}
