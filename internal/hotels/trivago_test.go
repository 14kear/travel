package hotels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---- parseTrivagoResponse tests ----

func TestParseTrivagoResponsePlainJSON(t *testing.T) {
	// Plain JSON-RPC response (not SSE).
	payload := `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"hotels\":[]}"}]}}`
	got, err := parseTrivagoResponse([]byte(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `{"hotels":[]}` {
		t.Errorf("got %s, want {\"hotels\":[]}", string(got))
	}
}

func TestParseTrivagoResponseSSE(t *testing.T) {
	// SSE stream with a data: line.
	inner := `{"hotels":[]}`
	wrapped := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":%s}]}}`,
		mustMarshalString(inner))
	sse := "event: message\ndata: " + wrapped + "\n\n"

	got, err := parseTrivagoResponse([]byte(sse))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != inner {
		t.Errorf("got %s, want %s", string(got), inner)
	}
}

func TestParseTrivagoResponseRPCError(t *testing.T) {
	payload := `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"invalid request"}}`
	_, err := parseTrivagoResponse([]byte(payload))
	if err == nil {
		t.Fatal("expected error for RPC error response, got nil")
	}
	if !strings.Contains(err.Error(), "RPC error") {
		t.Errorf("error message should mention RPC error, got: %v", err)
	}
}

func TestParseTrivagoResponseEmpty(t *testing.T) {
	_, err := parseTrivagoResponse([]byte("not json at all"))
	if err == nil {
		t.Fatal("expected error for garbage body, got nil")
	}
}

// ---- parseTrivagoSuggestions tests ----

func TestParseTrivagoSuggestionsStructured(t *testing.T) {
	raw := json.RawMessage(`{
		"suggestions": [
			{"item": {"id": 42, "type": "city"}, "name": "Paris", "type": "city"},
			{"item": {"id": 99, "type": "city"}, "name": "Paris CDG", "type": "airport"}
		]
	}`)

	got, err := parseTrivagoSuggestions(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(got), "42") {
		t.Errorf("expected first suggestion item (id 42), got: %s", string(got))
	}
}

func TestParseTrivagoSuggestionsAlternateKey(t *testing.T) {
	// "results" key instead of "suggestions".
	raw := json.RawMessage(`{"results": [{"id": 7, "name": "Rome"}]}`)
	got, err := parseTrivagoSuggestions(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(got), "7") {
		t.Errorf("expected item with id 7, got: %s", string(got))
	}
}

func TestParseTrivagoSuggestionsEmpty(t *testing.T) {
	// Empty suggestions list — falls back to raw payload.
	raw := json.RawMessage(`{"suggestions": []}`)
	got, err := parseTrivagoSuggestions(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return the raw payload as fallback.
	if got == nil {
		t.Error("expected non-nil fallback, got nil")
	}
}

// ---- parseTrivagoAccommodations tests ----

func TestParseTrivagoAccommodationsTyped(t *testing.T) {
	raw := json.RawMessage(`{
		"accommodations": [
			{
				"name": "Hotel Roma",
				"rating": 4.3,
				"reviewCount": 512,
				"stars": 4,
				"address": "Via del Corso 1, Rome",
				"latitude": 41.9028,
				"longitude": 12.4964,
				"price": {"amount": 149.0, "currency": "EUR"},
				"bookingLinks": [
					{"url": "https://trivago.com/book/1", "price": 149.0, "currency": "EUR", "provider": "booking.com"},
					{"url": "https://trivago.com/book/2", "price": 139.0, "currency": "EUR", "provider": "hotels.com"}
				]
			}
		]
	}`)

	results, err := parseTrivagoAccommodations(raw, "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	h := results[0]
	if h.Name != "Hotel Roma" {
		t.Errorf("name: got %q, want %q", h.Name, "Hotel Roma")
	}
	if h.Price != 139.0 {
		t.Errorf("price: got %.1f, want 139.0 (cheapest booking link)", h.Price)
	}
	if h.Currency != "EUR" {
		t.Errorf("currency: got %q, want EUR", h.Currency)
	}
	if h.Rating != 4.3 {
		t.Errorf("rating: got %.1f, want 4.3", h.Rating)
	}
	if h.Stars != 4 {
		t.Errorf("stars: got %d, want 4", h.Stars)
	}
	if h.Lat != 41.9028 {
		t.Errorf("lat: got %v, want 41.9028", h.Lat)
	}
}

func TestParseTrivagoAccommodationsPriceSourceTagging(t *testing.T) {
	raw := json.RawMessage(`{
		"accommodations": [
			{
				"name": "Budget Inn",
				"price": {"amount": 59.0, "currency": "USD"},
				"bookingLinks": []
			}
		]
	}`)

	results, err := parseTrivagoAccommodations(raw, "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	h := results[0]
	if len(h.Sources) == 0 {
		t.Fatal("expected at least one PriceSource")
	}
	src := h.Sources[0]
	if src.Provider != "trivago" {
		t.Errorf("source provider: got %q, want trivago", src.Provider)
	}
	if src.Price != 59.0 {
		t.Errorf("source price: got %.1f, want 59.0", src.Price)
	}
}

func TestParseTrivagoAccommodationsAlternateKey(t *testing.T) {
	// "hotels" key instead of "accommodations".
	raw := json.RawMessage(`{
		"hotels": [
			{"name": "Hotel Alt", "price": {"amount": 80.0, "currency": "GBP"}, "bookingLinks": []}
		]
	}`)

	results, err := parseTrivagoAccommodations(raw, "GBP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestParseTrivagoAccommodationsEmpty(t *testing.T) {
	raw := json.RawMessage(`{"accommodations": []}`)
	results, err := parseTrivagoAccommodations(raw, "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty list, got %d", len(results))
	}
}

func TestParseTrivagoAccommodationsSkipsEmptyName(t *testing.T) {
	raw := json.RawMessage(`{
		"accommodations": [
			{"name": "", "price": {"amount": 50.0, "currency": "USD"}, "bookingLinks": []},
			{"name": "Real Hotel", "price": {"amount": 75.0, "currency": "USD"}, "bookingLinks": []}
		]
	}`)

	results, err := parseTrivagoAccommodations(raw, "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (skipping empty name), got %d", len(results))
	}
	if results[0].Name != "Real Hotel" {
		t.Errorf("expected Real Hotel, got %q", results[0].Name)
	}
}

// ---- trivagoMCPCall integration with mock server ----

func TestTrivagoMCPCallMockServer(t *testing.T) {
	// Mock server that returns a plain-JSON JSON-RPC response.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Verify content-type header.
		ct := r.Header.Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			t.Errorf("expected application/json content-type, got %q", ct)
		}

		// Parse request body and validate structure.
		var req trivagoRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Method != "tools/call" {
			t.Errorf("expected method tools/call, got %q", req.Method)
		}

		// Return a valid accommodation response.
		payload := `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"accommodations\":[{\"name\":\"Mock Hotel\",\"rating\":4.5,\"reviewCount\":200,\"stars\":4,\"price\":{\"amount\":120.0,\"currency\":\"EUR\"},\"bookingLinks\":[{\"url\":\"https://example.com\",\"price\":110.0,\"currency\":\"EUR\",\"provider\":\"booking.com\"}]}]}"}]}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, payload)
	}))
	defer srv.Close()

	// Temporarily replace the HTTP client and endpoint for this test.
	origClient := trivagoHTTPClient
	origEndpoint := trivagoMCPEndpoint
	trivagoHTTPClient = &http.Client{Timeout: 5 * trivagoHTTPClient.Timeout}
	_ = origEndpoint // can't reassign const; use the server URL via a custom transport
	trivagoHTTPClient = &http.Client{
		Transport: &rewriteTransport{target: srv.URL},
	}
	defer func() { trivagoHTTPClient = origClient }()

	raw, err := trivagoMCPCall(context.Background(), "trivago-accommodation-search", map[string]any{
		"item":     map[string]any{"id": 1},
		"checkIn":  "2026-07-01",
		"checkOut": "2026-07-05",
		"adults":   2,
	})
	if err != nil {
		t.Fatalf("trivagoMCPCall failed: %v", err)
	}

	var accom trivagoAccomResult
	if err := json.Unmarshal(raw, &accom); err != nil {
		t.Fatalf("unmarshal accommodation result: %v", err)
	}
	if len(accom.Accommodations) != 1 {
		t.Fatalf("expected 1 accommodation, got %d", len(accom.Accommodations))
	}
	if accom.Accommodations[0].Name != "Mock Hotel" {
		t.Errorf("expected Mock Hotel, got %q", accom.Accommodations[0].Name)
	}
}

func TestTrivagoMCPCallHTTP429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	origClient := trivagoHTTPClient
	trivagoHTTPClient = &http.Client{
		Transport: &rewriteTransport{target: srv.URL},
	}
	defer func() { trivagoHTTPClient = origClient }()

	_, err := trivagoMCPCall(context.Background(), "trivago-search-suggestions", map[string]any{"query": "Paris"})
	if err == nil {
		t.Fatal("expected error for 429 response, got nil")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error should mention 429, got: %v", err)
	}
}

func TestTrivagoMCPCallSSEResponse(t *testing.T) {
	// Mock server returning SSE format.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerText := `{"suggestions":[{"item":{"id":5,"type":"city"},"name":"Tokyo","type":"city"}]}`
		wrapped := `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":` +
			mustMarshalString(innerText) + `}]}}`
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", wrapped)
	}))
	defer srv.Close()

	origClient := trivagoHTTPClient
	trivagoHTTPClient = &http.Client{
		Transport: &rewriteTransport{target: srv.URL},
	}
	defer func() { trivagoHTTPClient = origClient }()

	raw, err := trivagoMCPCall(context.Background(), "trivago-search-suggestions", map[string]any{"query": "Tokyo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item, err := parseTrivagoSuggestions(raw)
	if err != nil {
		t.Fatalf("parse suggestions: %v", err)
	}
	if !strings.Contains(string(item), "5") {
		t.Errorf("expected item id 5, got: %s", string(item))
	}
}

// ---- SearchTrivago error handling ----

func TestSearchTrivagoMissingDates(t *testing.T) {
	// Re-enable Trivago for this test so input validation is exercised.
	origEnabled := trivagoEnabled
	trivagoEnabled = true
	defer func() { trivagoEnabled = origEnabled }()

	_, err := SearchTrivago(context.Background(), "Paris", HotelSearchOptions{})
	if err == nil {
		t.Fatal("expected error for missing dates, got nil")
	}
}

func TestSearchTrivagoCancelledContext(t *testing.T) {
	// Re-enable Trivago for this test so the rate limiter respects ctx.Done().
	origEnabled := trivagoEnabled
	trivagoEnabled = true
	defer func() { trivagoEnabled = origEnabled }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := SearchTrivago(ctx, "Paris", HotelSearchOptions{
		CheckIn:  "2026-07-01",
		CheckOut: "2026-07-05",
	})
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

// ---- helper types ----

// rewriteTransport rewrites all outbound requests to target for testing.
type rewriteTransport struct {
	target string
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(rt.target, "http://")
	return http.DefaultTransport.RoundTrip(req)
}

// mustMarshalString returns the JSON-encoded form of a string value.
func mustMarshalString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
