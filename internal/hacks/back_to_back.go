package hacks

import (
	"context"
	"fmt"
)

// detectBackToBack suggests the back-to-back ticketing strategy for frequent
// travellers on the same route. Two overlapping round-trips are typically
// 20-40% cheaper than individual one-ways because airlines discount return
// fares. Purely advisory — zero API calls.
func detectBackToBack(_ context.Context, in DetectorInput) []Hack {
	// Only fire when there's a return date (user is booking round-trip).
	if in.Origin == "" || in.Destination == "" || in.ReturnDate == "" {
		return nil
	}
	if in.Date == "" {
		return nil
	}

	// Calculate trip duration — only suggest for short trips (1-14 days)
	// where business-style back-to-back is common.
	depart, err1 := parseDate(in.Date)
	ret, err2 := parseDate(in.ReturnDate)
	if err1 != nil || err2 != nil {
		return nil
	}
	nights := int(ret.Sub(depart).Hours() / 24)
	if nights < 1 || nights > 14 {
		return nil
	}

	return []Hack{{
		Type: "back_to_back",
		Title: "Frequent route? Back-to-back round-trips beat one-ways",
		Description: fmt.Sprintf(
			"If you travel %s↔%s regularly, buying two overlapping round-trips "+
				"(outbound trip 1 + outbound trip 2, discard both returns) is typically "+
				"20-40%% cheaper than individual one-ways because airlines discount returns.",
			in.Origin, in.Destination),
		Savings:  0, // advisory — no concrete savings estimate
		Currency: in.currency(),
		Steps: []string{
			fmt.Sprintf("For your next 2 trips %s→%s:", in.Origin, in.Destination),
			"Ticket A: round-trip starting from " + in.Origin + " (use outbound only)",
			"Ticket B: round-trip starting from " + in.Destination + " (use outbound only)",
			"Each ticket's return leg is unused — but the round-trip price is cheaper than one-way",
			"Also exploits Saturday-night-stay discounting if trips span weekends",
		},
		Risks: []string{
			"Unused return legs waste carbon (ethical consideration)",
			"Airlines may flag accounts with systematic no-shows on returns",
			"Book on different booking references to avoid pattern detection",
		},
	}}
}
