package deals

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// rssChannel represents the top-level RSS channel.
type rssChannel struct {
	XMLName xml.Name  `xml:"rss"`
	Channel rssInner  `xml:"channel"`
}

type rssInner struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

// shared rate limiter: 10 requests per minute across all feeds.
var limiter = rate.NewLimiter(rate.Every(6*time.Second), 4)

// httpClient is the shared HTTP client for RSS fetches.
var httpClient = &http.Client{Timeout: 15 * time.Second}

// FetchDeals fetches deals from the given sources in parallel, applies the filter,
// and returns the aggregated result sorted by published date (newest first).
func FetchDeals(ctx context.Context, sources []string, filter DealFilter) (*DealsResult, error) {
	if len(sources) == 0 {
		sources = AllSources
	}

	type feedResult struct {
		deals []Deal
		err   error
	}

	results := make([]feedResult, len(sources))
	var wg sync.WaitGroup

	for i, src := range sources {
		feedURL, ok := SourceFeeds[src]
		if !ok {
			results[i] = feedResult{err: fmt.Errorf("unknown source: %s", src)}
			continue
		}
		wg.Add(1)
		go func(idx int, source, url string) {
			defer wg.Done()
			deals, err := fetchFeed(ctx, source, url)
			results[idx] = feedResult{deals: deals, err: err}
		}(i, src, feedURL)
	}

	wg.Wait()

	var allDeals []Deal
	var errs []string
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, r.err.Error())
			continue
		}
		allDeals = append(allDeals, r.deals...)
	}

	// Apply filters.
	filtered := FilterDeals(allDeals, filter)

	// Sort by published date descending.
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Published.After(filtered[j].Published)
	})

	result := &DealsResult{
		Success: true,
		Count:   len(filtered),
		Deals:   filtered,
	}
	if len(errs) > 0 && len(filtered) == 0 {
		result.Success = false
		result.Error = strings.Join(errs, "; ")
	}
	return result, nil
}

// fetchFeed fetches and parses a single RSS feed.
func fetchFeed(ctx context.Context, source, url string) ([]Deal, error) {
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%s: rate limit: %w", source, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}
	req.Header.Set("User-Agent", "trvl/0.3.0 (+https://github.com/MikkoParkkola/trvl)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: fetch: %w", source, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", source, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024)) // 2MB limit
	if err != nil {
		return nil, fmt.Errorf("%s: read: %w", source, err)
	}

	return ParseRSS(body, source)
}

// ParseRSS parses RSS XML bytes into Deal structs for the given source.
func ParseRSS(data []byte, source string) ([]Deal, error) {
	var rss rssChannel
	if err := xml.Unmarshal(data, &rss); err != nil {
		return nil, fmt.Errorf("parse RSS (%s): %w", source, err)
	}

	var deals []Deal
	for _, item := range rss.Channel.Items {
		d := Deal{
			Title:  html.UnescapeString(strings.TrimSpace(item.Title)),
			Source: source,
			URL:    strings.TrimSpace(item.Link),
		}

		// Parse published date (RFC 1123/RFC 2822 variants).
		d.Published = parseRSSDate(item.PubDate)

		// Extract summary from description (strip HTML).
		d.Summary = stripHTML(item.Description)
		if len(d.Summary) > 200 {
			d.Summary = d.Summary[:200] + "..."
		}

		// Extract price and route from title.
		extractPriceAndRoute(&d)

		// Classify deal type.
		classifyDeal(&d)

		deals = append(deals, d)
	}

	return deals, nil
}

// --- Price and route extraction ---

// Regex patterns for extracting price, origin, destination from RSS titles.
var (
	// "$299" or "EUR595" or "299 EUR" or "US$199"
	pricePatterns = []*regexp.Regexp{
		// "$299" or "US$199" or "CA$450"
		regexp.MustCompile(`(?:US|CA|AU|NZ|HK|SG)?\$\s*(\d+(?:\.\d{2})?)`),
		// "EUR 595" or "EUR595" or "595 EUR"
		regexp.MustCompile(`(?i)(EUR|GBP|CHF|SEK|NOK|DKK|PLN|CZK)\s*(\d+(?:\.\d{2})?)`),
		regexp.MustCompile(`(\d+(?:\.\d{2})?)\s*(EUR|GBP|CHF|SEK|NOK|DKK|PLN|CZK)`),
		// Pounds: "£199"
		regexp.MustCompile(`\x{00a3}\s*(\d+(?:\.\d{2})?)`),
	}

	// "from Rome to Taiwan" or "Helsinki to Tokyo" -- captures single or two-word city names around "to".
	routeFromToAnchored = regexp.MustCompile(`(?i)\bfrom\s+([A-Z][a-z]+(?:\s[A-Z][a-z]+)?)\s+to\s+([A-Z][a-z]+)`)
	routeToOnly         = regexp.MustCompile(`(?i)\b([A-Z][a-z]+)\s+to\s+([A-Z][a-z]+)\b`)

	// "Helsinki-Tokyo" or "HEL-NRT"
	routeDash = regexp.MustCompile(`\b([A-Z]{3})\s*[-–]\s*([A-Z]{3})\b`)

	// Airlines in titles
	airlinePattern = regexp.MustCompile(`(?i)\b(Finnair|Lufthansa|Ryanair|easyJet|Norwegian|SAS|KLM|British Airways|Air France|Swiss|TAP|Wizz Air|Vueling|Eurowings|Iberia|Turkish Airlines|Emirates|Qatar Airways|Singapore Airlines|ANA|JAL|Delta|United|American Airlines|JetBlue|Southwest|Spirit|Frontier|Alaska Airlines|Air Canada|WestJet)\b`)
)

// extractPriceAndRoute parses the deal title to extract price, currency, origin, and destination.
func extractPriceAndRoute(d *Deal) {
	title := d.Title

	// Extract price.
	for _, pat := range pricePatterns {
		m := pat.FindStringSubmatch(title)
		if m == nil {
			continue
		}
		switch {
		case strings.Contains(pat.String(), `\$`):
			// Dollar pattern: group 1 is the amount.
			d.Price = parseFloat(m[1])
			d.Currency = "USD"
			if strings.HasPrefix(m[0], "CA") {
				d.Currency = "CAD"
			} else if strings.HasPrefix(m[0], "AU") {
				d.Currency = "AUD"
			}
		case strings.Contains(pat.String(), `\x{00a3}`):
			// Pound pattern.
			d.Price = parseFloat(m[1])
			d.Currency = "GBP"
		default:
			// EUR/other currency patterns.
			if len(m) >= 3 {
				cur := strings.ToUpper(m[1])
				amt := m[2]
				// Check if the amount is in group 1 (number first pattern).
				if _, err := fmt.Sscanf(m[1], "%f", new(float64)); err == nil {
					amt = m[1]
					cur = strings.ToUpper(m[2])
				}
				d.Price = parseFloat(amt)
				d.Currency = cur
			}
		}
		if d.Price > 0 {
			break
		}
	}

	// Extract route.
	if m := routeDash.FindStringSubmatch(title); m != nil {
		d.Origin = m[1]
		d.Destination = m[2]
	} else if m := routeFromToAnchored.FindStringSubmatch(title); m != nil {
		d.Origin = strings.TrimSpace(m[1])
		d.Destination = strings.TrimSpace(m[2])
	} else if m := routeToOnly.FindStringSubmatch(title); m != nil {
		orig := strings.TrimSpace(m[1])
		dest := strings.TrimSpace(m[2])
		// Skip common false positives: words like "Non-stop", "Flights", etc.
		if isLikelyCity(orig) && isLikelyCity(dest) {
			d.Origin = orig
			d.Destination = dest
		}
	}

	// Extract airline.
	if m := airlinePattern.FindStringSubmatch(title); m != nil {
		d.Airline = m[1]
	}
}

// classifyDeal sets the deal Type based on title keywords.
func classifyDeal(d *Deal) {
	lower := strings.ToLower(d.Title)
	switch {
	case strings.Contains(lower, "error fare") || strings.Contains(lower, "mistake fare"):
		d.Type = "error_fare"
	case strings.Contains(lower, "flash sale") || strings.Contains(lower, "flash deal"):
		d.Type = "flash_sale"
	case strings.Contains(lower, "package") || strings.Contains(lower, "hotel +") || strings.Contains(lower, "holiday"):
		d.Type = "package"
	default:
		d.Type = "deal"
	}
}

// isLikelyCity returns true if the string looks like a city name rather than
// a common English word that appears in deal titles.
func isLikelyCity(s string) bool {
	notCities := map[string]bool{
		"flights": true, "stop": true, "nonstop": true, "cheap": true,
		"return": true, "trip": true, "way": true, "fare": true,
		"deal": true, "sale": true, "error": true, "mistake": true,
		"flash": true, "direct": true, "travel": true, "book": true,
		"holiday": true, "package": true, "airline": true, "airport": true,
	}
	return !notCities[strings.ToLower(s)]
}

// --- Helpers ---

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return math.Round(f*100) / 100
}

func parseRSSDate(s string) time.Time {
	s = strings.TrimSpace(s)
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 MST",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func stripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	// Collapse whitespace.
	parts := strings.Fields(s)
	return strings.Join(parts, " ")
}
