package ground

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MikkoParkkola/trvl/internal/models"
	"golang.org/x/time/rate"
)

// httpClient is a shared HTTP client with sensible timeouts for FlixBus/RegioJet.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// Shared rate limiters for FlixBus and RegioJet (used by the shared httpClient).
var (
	flixbusLimiter  = rate.NewLimiter(rate.Limit(10), 1) // 10 req/s
	regiojetLimiter = rate.NewLimiter(rate.Limit(10), 1) // 10 req/s
)

// rateLimitedDo executes an HTTP request through the shared client after
// waiting on the provided rate limiter.
func rateLimitedDo(ctx context.Context, limiter *rate.Limiter, req *http.Request) (*http.Response, error) {
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}
	return httpClient.Do(req)
}

// SearchOptions configures a ground transport search.
type SearchOptions struct {
	Currency  string // Default: EUR
	Providers []string // Filter to specific providers; empty = all
	MaxPrice  float64  // 0 = no limit
	Type      string   // "bus", "train", or empty for all
}

// SearchByName searches all providers for ground transport between two cities
// given by name. Resolves city names to provider-specific IDs automatically.
func SearchByName(ctx context.Context, from, to, date string, opts SearchOptions) (*models.GroundSearchResult, error) {
	if opts.Currency == "" {
		opts.Currency = "EUR"
	}

	type providerResult struct {
		routes []models.GroundRoute
		err    error
		name   string
	}

	var wg sync.WaitGroup
	results := make(chan providerResult, 5)

	useProvider := func(name string) bool {
		if len(opts.Providers) == 0 {
			return true
		}
		for _, p := range opts.Providers {
			if strings.EqualFold(p, name) {
				return true
			}
		}
		return false
	}

	// FlixBus
	if useProvider("flixbus") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			routes, err := searchFlixBusByName(ctx, from, to, date, opts.Currency)
			results <- providerResult{routes: routes, err: err, name: "flixbus"}
		}()
	}

	// RegioJet
	if useProvider("regiojet") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			routes, err := searchRegioJetByName(ctx, from, to, date, opts.Currency)
			results <- providerResult{routes: routes, err: err, name: "regiojet"}
		}()
	}

	// Eurostar — only if both cities have Eurostar stations.
	// Try Snap (last-minute deals) first — if no Snap fares, fall back to regular.
	if (useProvider("eurostar") || useProvider("eurostar_snap")) && HasEurostarRoute(from, to) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Eurostar returns cheapest fares for a date range; use the single date
			// as both start and end to get that day's price.
			// Try Snap first (preferred — better value), fall back to regular.
			routes, err := SearchEurostar(ctx, from, to, date, date, opts.Currency, true)
			if err != nil || len(routes) == 0 {
				slog.Debug("no eurostar snap fares, trying regular", "err", err)
				routes, err = SearchEurostar(ctx, from, to, date, date, opts.Currency, false)
			}
			results <- providerResult{routes: routes, err: err, name: "eurostar"}
		}()
	}

	// SNCF — only if at least one city is French.
	if useProvider("sncf") && HasSNCFRoute(from, to) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			routes, err := SearchSNCF(ctx, from, to, date, opts.Currency)
			results <- providerResult{routes: routes, err: err, name: "sncf"}
		}()
	}

	// Transitous — coordinate-based, always available as a fallback.
	// Requires geocoding city names to coordinates; skipped if geocoding fails.
	if useProvider("transitous") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			routes, err := searchTransitousByName(ctx, from, to, date)
			results <- providerResult{routes: routes, err: err, name: "transitous"}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allRoutes []models.GroundRoute
	var errors []string
	for r := range results {
		if r.err != nil {
			slog.Warn("ground provider error", "provider", r.name, "error", r.err)
			errors = append(errors, fmt.Sprintf("%s: %v", r.name, r.err))
			continue
		}
		allRoutes = append(allRoutes, r.routes...)
	}

	// Filter out zero-price routes (sold-out routes from RegioJet)
	{
		filtered := allRoutes[:0]
		for _, r := range allRoutes {
			if r.Price > 0 {
				filtered = append(filtered, r)
			}
		}
		allRoutes = filtered
	}

	// Apply filters
	if opts.MaxPrice > 0 {
		filtered := allRoutes[:0]
		for _, r := range allRoutes {
			if r.Price <= opts.MaxPrice {
				filtered = append(filtered, r)
			}
		}
		allRoutes = filtered
	}
	if opts.Type != "" {
		filtered := allRoutes[:0]
		for _, r := range allRoutes {
			if strings.EqualFold(r.Type, opts.Type) {
				filtered = append(filtered, r)
			}
		}
		allRoutes = filtered
	}

	// Sort by price
	sort.Slice(allRoutes, func(i, j int) bool {
		return allRoutes[i].Price < allRoutes[j].Price
	})

	result := &models.GroundSearchResult{
		Success: len(allRoutes) > 0,
		Count:   len(allRoutes),
		Routes:  allRoutes,
	}
	if len(allRoutes) == 0 && len(errors) > 0 {
		result.Error = strings.Join(errors, "; ")
	}
	return result, nil
}

// searchFlixBusByName resolves city names and searches FlixBus.
func searchFlixBusByName(ctx context.Context, from, to, date, currency string) ([]models.GroundRoute, error) {
	fromCities, err := FlixBusAutoComplete(ctx, from)
	if err != nil {
		return nil, fmt.Errorf("resolve from city: %w", err)
	}
	if len(fromCities) == 0 {
		return nil, fmt.Errorf("no FlixBus city found for %q", from)
	}

	toCities, err := FlixBusAutoComplete(ctx, to)
	if err != nil {
		return nil, fmt.Errorf("resolve to city: %w", err)
	}
	if len(toCities) == 0 {
		return nil, fmt.Errorf("no FlixBus city found for %q", to)
	}

	routes, err := SearchFlixBus(ctx, fromCities[0].ID, toCities[0].ID, date, currency)
	if err != nil {
		return nil, err
	}

	// Enrich city names
	for i := range routes {
		if routes[i].Departure.City == "" {
			routes[i].Departure.City = fromCities[0].Name
		}
		if routes[i].Arrival.City == "" {
			routes[i].Arrival.City = toCities[0].Name
		}
	}

	return routes, nil
}

// searchRegioJetByName resolves city names and searches RegioJet.
func searchRegioJetByName(ctx context.Context, from, to, date, currency string) ([]models.GroundRoute, error) {
	fromCities, err := RegioJetAutoComplete(ctx, from)
	if err != nil {
		return nil, fmt.Errorf("resolve from city: %w", err)
	}
	if len(fromCities) == 0 {
		return nil, fmt.Errorf("no RegioJet city found for %q", from)
	}

	toCities, err := RegioJetAutoComplete(ctx, to)
	if err != nil {
		return nil, fmt.Errorf("resolve to city: %w", err)
	}
	if len(toCities) == 0 {
		return nil, fmt.Errorf("no RegioJet city found for %q", to)
	}

	return SearchRegioJet(ctx, fromCities[0].ID, toCities[0].ID, date, currency)
}

// searchTransitousByName geocodes city names to coordinates and searches Transitous.
func searchTransitousByName(ctx context.Context, from, to, date string) ([]models.GroundRoute, error) {
	fromGeo, err := geocodeCity(ctx, from)
	if err != nil {
		return nil, fmt.Errorf("geocode from city: %w", err)
	}
	toGeo, err := geocodeCity(ctx, to)
	if err != nil {
		return nil, fmt.Errorf("geocode to city: %w", err)
	}
	return SearchTransitous(ctx, fromGeo.lat, fromGeo.lon, toGeo.lat, toGeo.lon, date)
}

// geoCoord holds a latitude/longitude pair from geocoding.
type geoCoord struct {
	lat float64
	lon float64
}

// geoCityCache caches city name to coordinate lookups.
var geoCityCache = struct {
	sync.RWMutex
	entries map[string]geoCoord
}{entries: make(map[string]geoCoord)}

// geocodeCity resolves a city name to coordinates using Nominatim.
func geocodeCity(ctx context.Context, city string) (geoCoord, error) {
	key := strings.ToLower(strings.TrimSpace(city))

	geoCityCache.RLock()
	if entry, ok := geoCityCache.entries[key]; ok {
		geoCityCache.RUnlock()
		return entry, nil
	}
	geoCityCache.RUnlock()

	params := url.Values{
		"q":      {city},
		"format": {"json"},
		"limit":  {"1"},
	}
	apiURL := "https://nominatim.openstreetmap.org/search?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return geoCoord{}, err
	}
	req.Header.Set("User-Agent", "trvl/1.0 (travel agent; github.com/MikkoParkkola/trvl)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return geoCoord{}, fmt.Errorf("nominatim: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return geoCoord{}, fmt.Errorf("nominatim: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return geoCoord{}, fmt.Errorf("nominatim read: %w", err)
	}

	var results []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}
	if err := json.Unmarshal(body, &results); err != nil {
		return geoCoord{}, fmt.Errorf("nominatim decode: %w", err)
	}
	if len(results) == 0 {
		return geoCoord{}, fmt.Errorf("no geocoding results for %q", city)
	}

	var lat, lon float64
	if _, err := fmt.Sscanf(results[0].Lat, "%f", &lat); err != nil {
		return geoCoord{}, fmt.Errorf("parse lat %q: %w", results[0].Lat, err)
	}
	if _, err := fmt.Sscanf(results[0].Lon, "%f", &lon); err != nil {
		return geoCoord{}, fmt.Errorf("parse lon %q: %w", results[0].Lon, err)
	}

	coord := geoCoord{lat: lat, lon: lon}
	geoCityCache.Lock()
	geoCityCache.entries[key] = coord
	geoCityCache.Unlock()

	return coord, nil
}
