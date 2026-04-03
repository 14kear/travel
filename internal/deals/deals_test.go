package deals

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// --- RSS XML parsing ---

const sampleRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>Non-stop flights from Rome to Taiwan from EUR595</title>
      <link>https://example.com/deal1</link>
      <pubDate>Thu, 03 Apr 2026 10:00:00 +0000</pubDate>
      <description>&lt;p&gt;Great deal on flights!&lt;/p&gt;</description>
    </item>
    <item>
      <title>Error fare: Helsinki to Tokyo $299 round trip</title>
      <link>https://example.com/deal2</link>
      <pubDate>Thu, 03 Apr 2026 09:00:00 +0000</pubDate>
      <description>Grab this error fare before it is gone</description>
    </item>
    <item>
      <title>$89 — Barcelona to Prague (nonstop)</title>
      <link>https://example.com/deal3</link>
      <pubDate>Thu, 03 Apr 2026 08:00:00 +0000</pubDate>
      <description>Budget flight deal</description>
    </item>
    <item>
      <title>Flash sale: Ryanair HEL-BCN from EUR29</title>
      <link>https://example.com/deal4</link>
      <pubDate>Thu, 03 Apr 2026 07:00:00 +0000</pubDate>
      <description>Flash sale ending soon</description>
    </item>
    <item>
      <title>Holiday package to Bali including hotel + flights from GBP499</title>
      <link>https://example.com/deal5</link>
      <pubDate>Thu, 03 Apr 2026 06:00:00 +0000</pubDate>
      <description>All inclusive package</description>
    </item>
  </channel>
</rss>`

func TestParseRSS_Basic(t *testing.T) {
	deals, err := ParseRSS([]byte(sampleRSS), "test")
	if err != nil {
		t.Fatalf("ParseRSS error: %v", err)
	}
	if len(deals) != 5 {
		t.Fatalf("expected 5 deals, got %d", len(deals))
	}

	// All deals should have source set.
	for _, d := range deals {
		if d.Source != "test" {
			t.Errorf("source = %q, want test", d.Source)
		}
		if d.URL == "" {
			t.Error("URL should not be empty")
		}
	}
}

func TestParseRSS_PriceExtraction(t *testing.T) {
	deals, err := ParseRSS([]byte(sampleRSS), "test")
	if err != nil {
		t.Fatalf("ParseRSS error: %v", err)
	}

	tests := []struct {
		idx      int
		price    float64
		currency string
	}{
		{0, 595, "EUR"},  // "from EUR595"
		{1, 299, "USD"},  // "$299"
		{2, 89, "USD"},   // "$89"
		{3, 29, "EUR"},   // "EUR29"
		{4, 499, "GBP"},  // "GBP499"
	}

	for _, tt := range tests {
		d := deals[tt.idx]
		if d.Price != tt.price {
			t.Errorf("deal[%d] (%q): price = %.2f, want %.2f", tt.idx, d.Title, d.Price, tt.price)
		}
		if d.Currency != tt.currency {
			t.Errorf("deal[%d] (%q): currency = %q, want %q", tt.idx, d.Title, d.Currency, tt.currency)
		}
	}
}

func TestParseRSS_RouteExtraction(t *testing.T) {
	deals, err := ParseRSS([]byte(sampleRSS), "test")
	if err != nil {
		t.Fatalf("ParseRSS error: %v", err)
	}

	tests := []struct {
		idx    int
		origin string
		dest   string
	}{
		{0, "Rome", "Taiwan"},
		{1, "Helsinki", "Tokyo"},
		{2, "Barcelona", "Prague"},
		{3, "HEL", "BCN"},
	}

	for _, tt := range tests {
		d := deals[tt.idx]
		if d.Origin != tt.origin {
			t.Errorf("deal[%d] (%q): origin = %q, want %q", tt.idx, d.Title, d.Origin, tt.origin)
		}
		if d.Destination != tt.dest {
			t.Errorf("deal[%d] (%q): destination = %q, want %q", tt.idx, d.Title, d.Destination, tt.dest)
		}
	}
}

func TestParseRSS_TypeClassification(t *testing.T) {
	deals, err := ParseRSS([]byte(sampleRSS), "test")
	if err != nil {
		t.Fatalf("ParseRSS error: %v", err)
	}

	tests := []struct {
		idx      int
		dealType string
	}{
		{0, "deal"},
		{1, "error_fare"},
		{2, "deal"},
		{3, "flash_sale"},
		{4, "package"},
	}

	for _, tt := range tests {
		d := deals[tt.idx]
		if d.Type != tt.dealType {
			t.Errorf("deal[%d] (%q): type = %q, want %q", tt.idx, d.Title, d.Type, tt.dealType)
		}
	}
}

func TestParseRSS_AirlineExtraction(t *testing.T) {
	deals, err := ParseRSS([]byte(sampleRSS), "test")
	if err != nil {
		t.Fatalf("ParseRSS error: %v", err)
	}

	// Deal 3 mentions Ryanair.
	if deals[3].Airline != "Ryanair" {
		t.Errorf("deal[3] airline = %q, want Ryanair", deals[3].Airline)
	}
}

func TestParseRSS_DateParsing(t *testing.T) {
	deals, err := ParseRSS([]byte(sampleRSS), "test")
	if err != nil {
		t.Fatalf("ParseRSS error: %v", err)
	}

	for i, d := range deals {
		if d.Published.IsZero() {
			t.Errorf("deal[%d] published date is zero", i)
		}
	}

	// First deal should be the earliest (10:00).
	if deals[0].Published.Hour() != 10 {
		t.Errorf("deal[0] hour = %d, want 10", deals[0].Published.Hour())
	}
}

func TestParseRSS_HTMLStripping(t *testing.T) {
	deals, err := ParseRSS([]byte(sampleRSS), "test")
	if err != nil {
		t.Fatalf("ParseRSS error: %v", err)
	}

	// First item has HTML in description.
	if strings.Contains(deals[0].Summary, "<p>") {
		t.Error("summary should not contain HTML tags")
	}
	if !strings.Contains(deals[0].Summary, "Great deal") {
		t.Error("summary should contain stripped text")
	}
}

func TestParseRSS_InvalidXML(t *testing.T) {
	_, err := ParseRSS([]byte("not xml at all"), "test")
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

func TestParseRSS_EmptyFeed(t *testing.T) {
	xml := `<?xml version="1.0"?><rss version="2.0"><channel><title>Empty</title></channel></rss>`
	deals, err := ParseRSS([]byte(xml), "test")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(deals) != 0 {
		t.Errorf("expected 0 deals, got %d", len(deals))
	}
}

// --- Filtering ---

func TestFilterDeals_ByOrigin(t *testing.T) {
	deals := []Deal{
		{Origin: "HEL", Destination: "BCN", Price: 100, Published: time.Now()},
		{Origin: "AMS", Destination: "BCN", Price: 200, Published: time.Now()},
		{Origin: "HEL", Destination: "ROM", Price: 300, Published: time.Now()},
	}

	filtered := FilterDeals(deals, DealFilter{Origins: []string{"HEL"}})
	if len(filtered) != 2 {
		t.Errorf("expected 2 deals from HEL, got %d", len(filtered))
	}
	for _, d := range filtered {
		if d.Origin != "HEL" {
			t.Errorf("origin = %q, want HEL", d.Origin)
		}
	}
}

func TestFilterDeals_ByOriginCaseInsensitive(t *testing.T) {
	deals := []Deal{
		{Origin: "hel", Destination: "BCN", Price: 100, Published: time.Now()},
	}

	filtered := FilterDeals(deals, DealFilter{Origins: []string{"HEL"}})
	if len(filtered) != 1 {
		t.Errorf("expected 1 deal (case-insensitive), got %d", len(filtered))
	}
}

func TestFilterDeals_ByMaxPrice(t *testing.T) {
	deals := []Deal{
		{Price: 50, Published: time.Now()},
		{Price: 150, Published: time.Now()},
		{Price: 250, Published: time.Now()},
	}

	filtered := FilterDeals(deals, DealFilter{MaxPrice: 200})
	if len(filtered) != 2 {
		t.Errorf("expected 2 deals under 200, got %d", len(filtered))
	}
}

func TestFilterDeals_ByMaxPrice_NoPriceDealsIncluded(t *testing.T) {
	deals := []Deal{
		{Price: 0, Published: time.Now()},   // no price
		{Price: 150, Published: time.Now()},  // under max
		{Price: 250, Published: time.Now()},  // over max
	}

	filtered := FilterDeals(deals, DealFilter{MaxPrice: 200})
	if len(filtered) != 2 {
		t.Errorf("expected 2 deals (no-price + under-max), got %d", len(filtered))
	}
}

func TestFilterDeals_ByType(t *testing.T) {
	deals := []Deal{
		{Type: "error_fare", Published: time.Now()},
		{Type: "deal", Published: time.Now()},
		{Type: "error_fare", Published: time.Now()},
	}

	filtered := FilterDeals(deals, DealFilter{Type: "error_fare"})
	if len(filtered) != 2 {
		t.Errorf("expected 2 error_fare deals, got %d", len(filtered))
	}
}

func TestFilterDeals_ByHoursAgo(t *testing.T) {
	now := time.Now()
	deals := []Deal{
		{Published: now.Add(-1 * time.Hour)},
		{Published: now.Add(-25 * time.Hour)},
		{Published: now.Add(-72 * time.Hour)},
	}

	filtered := FilterDeals(deals, DealFilter{HoursAgo: 24})
	if len(filtered) != 1 {
		t.Errorf("expected 1 deal within 24h, got %d", len(filtered))
	}
}

func TestFilterDeals_DefaultHoursAgo(t *testing.T) {
	now := time.Now()
	deals := []Deal{
		{Published: now.Add(-1 * time.Hour)},
		{Published: now.Add(-72 * time.Hour)},
	}

	// HoursAgo=0 should default to 48.
	filtered := FilterDeals(deals, DealFilter{})
	if len(filtered) != 1 {
		t.Errorf("expected 1 deal within default 48h, got %d", len(filtered))
	}
}

func TestFilterDeals_OriginSkipsNoOriginDeals(t *testing.T) {
	deals := []Deal{
		{Origin: "HEL", Published: time.Now()},
		{Origin: "", Published: time.Now()}, // no origin
	}

	filtered := FilterDeals(deals, DealFilter{Origins: []string{"HEL"}})
	if len(filtered) != 1 {
		t.Errorf("expected 1 deal (skip no-origin), got %d", len(filtered))
	}
}

func TestFilterDeals_NoFilter(t *testing.T) {
	deals := []Deal{
		{Published: time.Now()},
		{Published: time.Now()},
	}

	filtered := FilterDeals(deals, DealFilter{})
	if len(filtered) != 2 {
		t.Errorf("expected 2 deals with no filter, got %d", len(filtered))
	}
}

// --- Price extraction edge cases ---

func TestExtractPrice_DollarVariants(t *testing.T) {
	tests := []struct {
		title    string
		price    float64
		currency string
	}{
		{"Flights from $299", 299, "USD"},
		{"CA$450 to Tokyo", 450, "CAD"},
		{"AU$599 return flights", 599, "AUD"},
		{"From EUR 100 one way", 100, "EUR"},
		{"Flights 299 EUR round trip", 299, "EUR"},
		{"From \u00a3199 return", 199, "GBP"},
	}

	for _, tt := range tests {
		d := Deal{Title: tt.title}
		extractPriceAndRoute(&d)
		if d.Price != tt.price {
			t.Errorf("%q: price = %.2f, want %.2f", tt.title, d.Price, tt.price)
		}
		if d.Currency != tt.currency {
			t.Errorf("%q: currency = %q, want %q", tt.title, d.Currency, tt.currency)
		}
	}
}

func TestExtractRoute_DashPattern(t *testing.T) {
	d := Deal{Title: "Flash sale HEL-BCN from EUR 29"}
	extractPriceAndRoute(&d)
	if d.Origin != "HEL" {
		t.Errorf("origin = %q, want HEL", d.Origin)
	}
	if d.Destination != "BCN" {
		t.Errorf("destination = %q, want BCN", d.Destination)
	}
}

func TestExtractRoute_FromToPattern(t *testing.T) {
	d := Deal{Title: "Cheap flights from London to Barcelona from $99"}
	extractPriceAndRoute(&d)
	if d.Origin != "London" {
		t.Errorf("origin = %q, want London", d.Origin)
	}
	if d.Destination != "Barcelona" {
		t.Errorf("destination = %q, want Barcelona", d.Destination)
	}
}

func TestClassifyDeal_ErrorFare(t *testing.T) {
	d := Deal{Title: "Mistake fare: New York to Paris $150"}
	classifyDeal(&d)
	if d.Type != "error_fare" {
		t.Errorf("type = %q, want error_fare", d.Type)
	}
}

func TestClassifyDeal_FlashSale(t *testing.T) {
	d := Deal{Title: "Flash deal: London to Rome from EUR49"}
	classifyDeal(&d)
	if d.Type != "flash_sale" {
		t.Errorf("type = %q, want flash_sale", d.Type)
	}
}

func TestClassifyDeal_Package(t *testing.T) {
	d := Deal{Title: "Holiday in Greece including hotel + flights"}
	classifyDeal(&d)
	if d.Type != "package" {
		t.Errorf("type = %q, want package", d.Type)
	}
}

func TestClassifyDeal_Default(t *testing.T) {
	d := Deal{Title: "Cheap flights to Tokyo"}
	classifyDeal(&d)
	if d.Type != "deal" {
		t.Errorf("type = %q, want deal", d.Type)
	}
}

// --- Concurrent fetch (mock HTTP) ---

func TestFetchDeals_MockHTTP(t *testing.T) {
	// Create a mock RSS server.
	var hitCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitCount.Add(1)
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, sampleRSS)
	}))
	defer srv.Close()

	// Override source feeds to point to mock server.
	origFeeds := make(map[string]string)
	for k, v := range SourceFeeds {
		origFeeds[k] = v
	}
	for k := range SourceFeeds {
		SourceFeeds[k] = srv.URL
	}
	defer func() {
		for k, v := range origFeeds {
			SourceFeeds[k] = v
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := FetchDeals(ctx, AllSources, DealFilter{HoursAgo: 999999})
	if err != nil {
		t.Fatalf("FetchDeals error: %v", err)
	}
	if !result.Success {
		t.Fatalf("result not successful: %s", result.Error)
	}

	// Should have fetched from all 4 sources.
	if hitCount.Load() != 4 {
		t.Errorf("expected 4 HTTP requests, got %d", hitCount.Load())
	}

	// Each source returns 5 items, total = 20.
	if result.Count != 20 {
		t.Errorf("expected 20 deals, got %d", result.Count)
	}

	// Verify deals are sorted by published date descending.
	for i := 1; i < len(result.Deals); i++ {
		if result.Deals[i].Published.After(result.Deals[i-1].Published) {
			t.Error("deals should be sorted newest first")
			break
		}
	}
}

func TestFetchDeals_UnknownSource(t *testing.T) {
	ctx := context.Background()
	result, err := FetchDeals(ctx, []string{"nonexistent"}, DealFilter{HoursAgo: 999999})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Success {
		t.Error("should not be successful with unknown source and no deals")
	}
}

func TestFetchDeals_EmptySourcesDefaultsToAll(t *testing.T) {
	// Create a mock server that returns empty feed.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<?xml version="1.0"?><rss version="2.0"><channel></channel></rss>`)
	}))
	defer srv.Close()

	origFeeds := make(map[string]string)
	for k, v := range SourceFeeds {
		origFeeds[k] = v
	}
	for k := range SourceFeeds {
		SourceFeeds[k] = srv.URL
	}
	defer func() {
		for k, v := range origFeeds {
			SourceFeeds[k] = v
		}
	}()

	ctx := context.Background()
	result, err := FetchDeals(ctx, nil, DealFilter{HoursAgo: 999999})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Should succeed even with empty feeds.
	if !result.Success {
		t.Error("should succeed with empty feeds")
	}
}

// --- Helper tests ---

func TestStripHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<p>Hello</p>", "Hello"},
		{"<b>Bold</b> &amp; <i>italic</i>", "Bold & italic"},
		{"No tags here", "No tags here"},
		{"  Multiple   spaces  ", "Multiple spaces"},
	}

	for _, tt := range tests {
		got := stripHTML(tt.input)
		if got != tt.want {
			t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseRSSDate(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"Thu, 03 Apr 2026 10:00:00 +0000", true},
		{"Mon, 2 Jan 2006 15:04:05 -0700", true},
		{"2026-04-03T10:00:00Z", true},
		{"not a date", false},
		{"", false},
	}

	for _, tt := range tests {
		got := parseRSSDate(tt.input)
		if tt.valid && got.IsZero() {
			t.Errorf("parseRSSDate(%q) returned zero time, expected valid", tt.input)
		}
		if !tt.valid && !got.IsZero() {
			t.Errorf("parseRSSDate(%q) returned non-zero time, expected zero", tt.input)
		}
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"299", 299},
		{"99.99", 99.99},
		{"0", 0},
		{"abc", 0},
	}

	for _, tt := range tests {
		got := parseFloat(tt.input)
		if got != tt.want {
			t.Errorf("parseFloat(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

// --- Data types ---

func TestAllSources(t *testing.T) {
	if len(AllSources) != 4 {
		t.Errorf("AllSources length = %d, want 4", len(AllSources))
	}

	for _, src := range AllSources {
		if _, ok := SourceFeeds[src]; !ok {
			t.Errorf("source %q missing from SourceFeeds", src)
		}
		if _, ok := SourceNames[src]; !ok {
			t.Errorf("source %q missing from SourceNames", src)
		}
	}
}

func TestDealFilter_ZeroValue(t *testing.T) {
	f := DealFilter{}
	if f.MaxPrice != 0 || f.Type != "" || f.HoursAgo != 0 || len(f.Origins) != 0 {
		t.Error("zero-value DealFilter should have all defaults")
	}
}

// --- Category extraction ---

func TestExtractFromCategories_Route(t *testing.T) {
	d := Deal{Title: "Cheap flights"}
	extractFromCategories(&d, []string{"Burbank, USA → Vancouver, Canada"})
	if d.Origin != "Burbank" {
		t.Errorf("origin = %q, want Burbank", d.Origin)
	}
	if d.Destination != "Vancouver" {
		t.Errorf("destination = %q, want Vancouver", d.Destination)
	}
}

func TestExtractFromCategories_RouteDoesNotOverwriteTitle(t *testing.T) {
	d := Deal{Title: "Flights from Rome to Tokyo $299"}
	extractPriceAndRoute(&d)
	// Title already extracted Rome->Tokyo.
	extractFromCategories(&d, []string{"Rome, Italy → Tokyo, Japan"})
	if d.Origin != "Rome" {
		t.Errorf("origin = %q, want Rome (should not overwrite)", d.Origin)
	}
	if d.Destination != "Tokyo" {
		t.Errorf("destination = %q, want Tokyo (should not overwrite)", d.Destination)
	}
}

func TestExtractFromCategories_Airline(t *testing.T) {
	d := Deal{Title: "Cheap flights"}
	extractFromCategories(&d, []string{"United", "Deal"})
	// "United" alone doesn't match the full airline regex; the category should be exact.
	// But "United Airlines" style entries should work:
	d2 := Deal{Title: "Cheap flights"}
	extractFromCategories(&d2, []string{"Qatar Airways", "Deal"})
	if d2.Airline != "Qatar Airways" {
		t.Errorf("airline = %q, want Qatar Airways", d2.Airline)
	}
}

func TestExtractFromCategories_Stops(t *testing.T) {
	tests := []struct {
		cat  string
		want string
	}{
		{"Non-stop", "nonstop"},
		{"Nonstop", "nonstop"},
		{"1 Stop", "1 stop"},
		{"2 Stops", "2 stops"},
	}
	for _, tt := range tests {
		d := Deal{Title: "Flights"}
		extractFromCategories(&d, []string{tt.cat})
		if d.Stops != tt.want {
			t.Errorf("category %q: stops = %q, want %q", tt.cat, d.Stops, tt.want)
		}
	}
}

func TestExtractFromCategories_CabinClass(t *testing.T) {
	tests := []struct {
		cat  string
		want string
	}{
		{"Economy", "economy"},
		{"Business Class", "business"},
		{"First Class", "first"},
		{"Premium Economy", "premium_economy"},
	}
	for _, tt := range tests {
		d := Deal{Title: "Flights"}
		extractFromCategories(&d, []string{tt.cat})
		if d.CabinClass != tt.want {
			t.Errorf("category %q: cabin = %q, want %q", tt.cat, d.CabinClass, tt.want)
		}
	}
}

func TestExtractFromCategories_ErrorFareType(t *testing.T) {
	d := Deal{Title: "Flights to Paris"}
	extractFromCategories(&d, []string{"Error Fare"})
	classifyDeal(&d)
	if d.Type != "error_fare" {
		t.Errorf("type = %q, want error_fare", d.Type)
	}
}

func TestExtractStopsFromTitle(t *testing.T) {
	d := Deal{Title: "Non-stop flights from Rome to Taiwan from EUR595"}
	extractFromCategories(&d, nil)
	if d.Stops != "nonstop" {
		t.Errorf("stops = %q, want nonstop", d.Stops)
	}
}

func TestExtractCabinFromTitle(t *testing.T) {
	d := Deal{Title: "Business class flights HEL-NRT from EUR1200"}
	extractFromCategories(&d, nil)
	if d.CabinClass != "business" {
		t.Errorf("cabin = %q, want business", d.CabinClass)
	}
}

// --- Date range extraction ---

func TestExtractDateRange(t *testing.T) {
	tests := []struct {
		desc string
		want string
	}{
		{
			"Great deal! Travel from April 2026 to January 2027 for cheap.",
			"April 2026 to January 2027",
		},
		{
			"Fly from May to September 2026.",
			"May to September 2026",
		},
		{
			"No date info here",
			"",
		},
	}
	for _, tt := range tests {
		d := Deal{}
		extractDateRange(&d, tt.desc)
		if d.DateRange != tt.want {
			t.Errorf("desc=%q: dateRange = %q, want %q", tt.desc, d.DateRange, tt.want)
		}
	}
}

// --- RSS with categories ---

const sampleRSSWithCategories = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>Cheap flights from $399</title>
      <link>https://example.com/deal1</link>
      <pubDate>Thu, 03 Apr 2026 10:00:00 +0000</pubDate>
      <description>Travel April 2026 to January 2027</description>
      <category>Burbank, USA → Vancouver, Canada</category>
      <category>Qatar Airways</category>
      <category>Non-stop</category>
      <category>Business Class</category>
      <category>Error Fare</category>
    </item>
  </channel>
</rss>`

func TestParseRSS_WithCategories(t *testing.T) {
	deals, err := ParseRSS([]byte(sampleRSSWithCategories), "test")
	if err != nil {
		t.Fatalf("ParseRSS error: %v", err)
	}
	if len(deals) != 1 {
		t.Fatalf("expected 1 deal, got %d", len(deals))
	}

	d := deals[0]
	if d.Origin != "Burbank" {
		t.Errorf("origin = %q, want Burbank", d.Origin)
	}
	if d.Destination != "Vancouver" {
		t.Errorf("destination = %q, want Vancouver", d.Destination)
	}
	if d.Airline != "Qatar Airways" {
		t.Errorf("airline = %q, want Qatar Airways", d.Airline)
	}
	if d.Stops != "nonstop" {
		t.Errorf("stops = %q, want nonstop", d.Stops)
	}
	if d.CabinClass != "business" {
		t.Errorf("cabin = %q, want business", d.CabinClass)
	}
	if d.Type != "error_fare" {
		t.Errorf("type = %q, want error_fare", d.Type)
	}
	if d.DateRange != "April 2026 to January 2027" {
		t.Errorf("dateRange = %q, want 'April 2026 to January 2027'", d.DateRange)
	}
	if d.Price != 399 {
		t.Errorf("price = %.2f, want 399", d.Price)
	}
}
