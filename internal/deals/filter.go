package deals

import (
	"strings"
	"time"
)

// FilterDeals applies the given filter to a slice of deals.
func FilterDeals(deals []Deal, f DealFilter) []Deal {
	hoursAgo := f.HoursAgo
	if hoursAgo <= 0 {
		hoursAgo = 48
	}
	cutoff := time.Now().Add(-time.Duration(hoursAgo) * time.Hour)

	originsUpper := make(map[string]bool, len(f.Origins))
	for _, o := range f.Origins {
		originsUpper[strings.ToUpper(strings.TrimSpace(o))] = true
	}

	var result []Deal
	for _, d := range deals {
		// Time filter.
		if !d.Published.IsZero() && d.Published.Before(cutoff) {
			continue
		}
		// Origin filter.
		if len(originsUpper) > 0 && d.Origin != "" {
			if !originsUpper[strings.ToUpper(d.Origin)] {
				continue
			}
		}
		// If origins filter is set but deal has no origin, skip it.
		if len(originsUpper) > 0 && d.Origin == "" {
			continue
		}
		// Price filter.
		if f.MaxPrice > 0 && d.Price > 0 && d.Price > f.MaxPrice {
			continue
		}
		// Type filter.
		if f.Type != "" && !strings.EqualFold(d.Type, f.Type) {
			continue
		}
		result = append(result, d)
	}
	return result
}
