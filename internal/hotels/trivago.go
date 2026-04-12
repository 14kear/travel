package hotels

// Trivago hotel provider.
//
// Uses the Trivago MCP server at https://mcp.trivago.com/mcp — a public
// JSON-RPC 2.0 endpoint that requires no API key. Responses are delivered
// as Server-Sent Events (SSE) with a single "data:" line containing the
// JSON-RPC result.
//
// Tool sequence:
//  1. trivago-search-suggestions(query) -> location item with item ID
//  2. trivago-accommodation-search(item, dates, guests) -> hotel list

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MikkoParkkola/trvl/internal/models"
	"golang.org/x/time/rate"
)

const trivagoMCPEndpoint = "https://mcp.trivago.com/mcp"

// trivagoEnabled controls whether SearchTrivago makes live HTTP requests.
// Set to false in tests that mock the Google Hotels transport to avoid
// unintended real-network calls to mcp.trivago.com.
var trivagoEnabled = true

// trivagoLimiter enforces a 2 req/s rate limit — conservative to avoid 429s.
var trivagoLimiter = rate.NewLimiter(rate.Every(500*time.Millisecond), 1)

// trivagoHTTPClient is a dedicated HTTP client for Trivago MCP calls.
var trivagoHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// ---- JSON-RPC request/response types ----

type trivagoRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  trivagoRPCParam `json:"params"`
}

type trivagoRPCParam struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type trivagoRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// trivagoToolResult is the outer envelope returned in result.content[0].text.
type trivagoToolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// ---- Suggestions response types ----

type trivagoSuggestionsResult struct {
	Suggestions []trivagoSuggestion `json:"suggestions"`
}

type trivagoSuggestion struct {
	// Trivago returns the item as a nested object with varying fields;
	// we capture what we need for the accommodation search call.
	Item trivagoItem `json:"item"`
	Name string      `json:"name"`
	Type string      `json:"type"`
}

type trivagoItem struct {
	// The item is passed verbatim to the next API call — capture raw JSON.
	raw json.RawMessage
}

func (t *trivagoItem) UnmarshalJSON(b []byte) error {
	t.raw = make(json.RawMessage, len(b))
	copy(t.raw, b)
	return nil
}

func (t trivagoItem) MarshalJSON() ([]byte, error) {
	if len(t.raw) == 0 {
		return []byte("null"), nil
	}
	return t.raw, nil
}

// ---- Accommodation response types ----

type trivagoAccomResult struct {
	Accommodations []trivagoAccommodation `json:"accommodations"`
}

type trivagoAccommodation struct {
	Name         string              `json:"name"`
	Rating       float64             `json:"rating"`
	ReviewCount  int                 `json:"reviewCount"`
	Stars        int                 `json:"stars"`
	Address      string              `json:"address"`
	Lat          float64             `json:"latitude"`
	Lon          float64             `json:"longitude"`
	Price        trivagoPrice        `json:"price"`
	BookingLinks []trivagoBookingLink `json:"bookingLinks"`
}

type trivagoPrice struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type trivagoBookingLink struct {
	URL      string  `json:"url"`
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
	Provider string  `json:"provider"`
}

// ---- MCP caller ----

// trivagoMCPCall sends a single tools/call JSON-RPC request to the Trivago
// MCP endpoint and parses the SSE response. Returns the raw JSON from the
// result content text field.
func trivagoMCPCall(ctx context.Context, toolName string, args map[string]any) (json.RawMessage, error) {
	if err := trivagoLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("trivago: rate limiter: %w", err)
	}

	reqBody := trivagoRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: trivagoRPCParam{
			Name:      toolName,
			Arguments: args,
		},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("trivago: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, trivagoMCPEndpoint, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("trivago: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := trivagoHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("trivago: HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("trivago: rate limited (HTTP 429)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("trivago: HTTP %d", resp.StatusCode)
	}

	// Parse response — may be plain JSON or SSE (text/event-stream).
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB cap
	if err != nil {
		return nil, fmt.Errorf("trivago: read body: %w", err)
	}

	return parseTrivagoResponse(body)
}

// parseTrivagoResponse extracts the JSON-RPC result payload from either a
// plain JSON body or an SSE stream. The Trivago MCP endpoint may return
// either depending on content negotiation.
func parseTrivagoResponse(body []byte) (json.RawMessage, error) {
	// Try plain JSON first.
	var rpcResp trivagoRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err == nil {
		return extractTrivagoContent(rpcResp)
	}

	// Fall back to SSE: scan for "data:" lines containing JSON.
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var rpc trivagoRPCResponse
		if err := json.Unmarshal([]byte(data), &rpc); err != nil {
			continue
		}
		return extractTrivagoContent(rpc)
	}

	return nil, fmt.Errorf("trivago: no usable JSON-RPC response found in body")
}

// extractTrivagoContent unwraps the result content from a JSON-RPC response.
// The Trivago MCP result looks like:
//
//	{"content":[{"type":"text","text":"{...actual JSON...}"}]}
func extractTrivagoContent(rpc trivagoRPCResponse) (json.RawMessage, error) {
	if rpc.Error != nil {
		return nil, fmt.Errorf("trivago: RPC error %d: %s", rpc.Error.Code, rpc.Error.Message)
	}
	if rpc.Result == nil {
		return nil, fmt.Errorf("trivago: empty result")
	}

	// Result may already be the data we want, or it may be wrapped in a
	// tool-result envelope with a content[].text field.
	// 512 KB is ample for any reasonable hotel list; guards against a
	// malicious/compromised mcp.trivago.com sending a multi-MB text payload
	// that the 1 MB HTTP body cap would not catch (the text sits inside the
	// already-limited body, but we want defense in depth).
	const maxContentText = 512 * 1024

	var toolResult trivagoToolResult
	if err := json.Unmarshal(rpc.Result, &toolResult); err == nil && len(toolResult.Content) > 0 {
		// Prefer the first text content block.
		for _, c := range toolResult.Content {
			if c.Type == "text" && c.Text != "" {
				if len(c.Text) > maxContentText {
					return nil, fmt.Errorf("trivago: content text too large (%d bytes)", len(c.Text))
				}
				return json.RawMessage(c.Text), nil
			}
		}
	}

	// Return raw result as-is.
	return rpc.Result, nil
}

// ---- Public API ----

// SearchTrivago searches for hotels using the Trivago MCP API.
//
// It performs two sequential MCP calls:
//  1. trivago-search-suggestions to resolve the location string to an item ID.
//  2. trivago-accommodation-search to get hotel listings for that location.
//
// Each returned HotelResult is tagged with a PriceSource for "trivago".
func SearchTrivago(ctx context.Context, location string, opts HotelSearchOptions) ([]models.HotelResult, error) {
	if !trivagoEnabled {
		return nil, nil
	}
	if opts.CheckIn == "" || opts.CheckOut == "" {
		return nil, fmt.Errorf("trivago: check-in and check-out dates are required")
	}
	if opts.Guests <= 0 {
		opts.Guests = 2
	}
	currency := opts.Currency
	if currency == "" {
		currency = "USD"
	}

	// Step 1: Resolve location to a Trivago item.
	slog.Debug("trivago search suggestions", "location", location)
	suggRaw, err := trivagoMCPCall(ctx, "trivago-search-suggestions", map[string]any{
		"query": location,
	})
	if err != nil {
		return nil, fmt.Errorf("trivago suggestions: %w", err)
	}

	item, err := parseTrivagoSuggestions(suggRaw)
	if err != nil {
		return nil, fmt.Errorf("trivago suggestions parse: %w", err)
	}

	// Step 2: Search accommodations.
	slog.Debug("trivago accommodation search", "location", location, "checkIn", opts.CheckIn, "checkOut", opts.CheckOut, "guests", opts.Guests)
	accomArgs := map[string]any{
		"item":     item,
		"checkIn":  opts.CheckIn,
		"checkOut": opts.CheckOut,
		"adults":   opts.Guests,
	}
	accomRaw, err := trivagoMCPCall(ctx, "trivago-accommodation-search", accomArgs)
	if err != nil {
		return nil, fmt.Errorf("trivago accommodation search: %w", err)
	}

	hotels, err := parseTrivagoAccommodations(accomRaw, currency)
	if err != nil {
		return nil, fmt.Errorf("trivago accommodation parse: %w", err)
	}

	slog.Debug("trivago results", "location", location, "count", len(hotels))
	return hotels, nil
}

// parseTrivagoSuggestions extracts the first location item from a
// trivago-search-suggestions response. The item is returned as raw JSON
// so it can be forwarded verbatim to trivago-accommodation-search.
func parseTrivagoSuggestions(raw json.RawMessage) (json.RawMessage, error) {
	// Try structured suggestions wrapper first.
	var result trivagoSuggestionsResult
	if err := json.Unmarshal(raw, &result); err == nil && len(result.Suggestions) > 0 {
		return result.Suggestions[0].Item.raw, nil
	}

	// Some responses embed suggestions in an outer object with a different key.
	var outer map[string]json.RawMessage
	if err := json.Unmarshal(raw, &outer); err != nil {
		return nil, fmt.Errorf("unexpected suggestions format")
	}

	// Walk possible key names.
	for _, key := range []string{"suggestions", "results", "data", "items"} {
		if v, ok := outer[key]; ok {
			var arr []json.RawMessage
			if err := json.Unmarshal(v, &arr); err == nil && len(arr) > 0 {
				// Return the first element as the item.
				return arr[0], nil
			}
		}
	}

	// Last resort: return the raw payload itself — Trivago might have changed
	// the envelope. The accommodation search will fail informatively if wrong.
	return raw, nil
}

// parseTrivagoAccommodations maps a trivago-accommodation-search response to
// a slice of HotelResult values, each tagged with a "trivago" PriceSource.
func parseTrivagoAccommodations(raw json.RawMessage, currency string) ([]models.HotelResult, error) {
	// Try the typed struct first.
	var result trivagoAccomResult
	if err := json.Unmarshal(raw, &result); err == nil && len(result.Accommodations) > 0 {
		return mapTrivagoAccommodations(result.Accommodations, currency), nil
	}

	// Accommodations might be at a different key.
	var outer map[string]json.RawMessage
	if err := json.Unmarshal(raw, &outer); err != nil {
		return nil, fmt.Errorf("unexpected accommodations format")
	}

	for _, key := range []string{"accommodations", "hotels", "results", "data"} {
		if v, ok := outer[key]; ok {
			var accoms []trivagoAccommodation
			if err := json.Unmarshal(v, &accoms); err == nil {
				return mapTrivagoAccommodations(accoms, currency), nil
			}
		}
	}

	// Return empty list if we genuinely got no hotels (e.g. obscure location).
	return nil, nil
}

// sanitizeBookingURL returns rawURL if it has an http or https scheme, or ""
// if it is empty, has an unexpected scheme (e.g. javascript:, data:), or is
// not a valid URL. This prevents a malicious MCP response from injecting
// non-HTTP URLs into booking links that a client might follow.
func sanitizeBookingURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return ""
	}
	return rawURL
}

// mapTrivagoAccommodations converts trivago API accommodation records to the
// canonical HotelResult model, selecting the cheapest booking link price.
func mapTrivagoAccommodations(accoms []trivagoAccommodation, defaultCurrency string) []models.HotelResult {
	results := make([]models.HotelResult, 0, len(accoms))

	for _, a := range accoms {
		if a.Name == "" {
			continue
		}

		// Pick the best (lowest) price from available booking links; fall back
		// to the top-level price field when no links are present.
		price := a.Price.Amount
		cur := a.Price.Currency
		bookingURL := ""

		for _, link := range a.BookingLinks {
			if link.Price <= 0 {
				continue
			}
			if price == 0 || link.Price < price {
				price = link.Price
				cur = link.Currency
				bookingURL = sanitizeBookingURL(link.URL)
			}
		}

		if cur == "" {
			cur = defaultCurrency
		}

		h := models.HotelResult{
			Name:        a.Name,
			Rating:      a.Rating,
			ReviewCount: a.ReviewCount,
			Stars:       a.Stars,
			Price:       price,
			Currency:    strings.ToUpper(cur),
			Address:     a.Address,
			Lat:         a.Lat,
			Lon:         a.Lon,
			BookingURL:  bookingURL,
			Sources: []models.PriceSource{{
				Provider:   "trivago",
				Price:      price,
				Currency:   strings.ToUpper(cur),
				BookingURL: bookingURL,
			}},
		}

		results = append(results, h)
	}

	return results
}
