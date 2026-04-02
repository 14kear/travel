//go:build proof

package batchexec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestTLSHandshake validates that our utls Chrome impersonation gets through
// Google's TLS fingerprint checks without being blocked.
//
// KILL: FF-1 if status != 200.
func TestTLSHandshake(t *testing.T) {
	c := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	status, body, err := c.Get(ctx, "https://www.google.com")
	if err != nil {
		t.Fatalf("FF-1: TLS handshake/GET failed: %v", err)
	}

	t.Logf("Status: %d, Body length: %d bytes", status, len(body))

	if status == 403 {
		t.Fatalf("FF-1 KILL: Google returned 403 Forbidden — TLS fingerprint blocked")
	}
	if status != 200 {
		t.Fatalf("FF-1: unexpected status %d (expected 200)", status)
	}

	t.Log("PASS: TLS handshake OK — Chrome impersonation accepted")
}

// TestFlightSearch attempts a real flight search: HEL -> NRT on 2026-06-15.
//
// This mirrors what the Python fli library does: POST encoded filters to the
// FlightsFrontendUi endpoint with f.req= body.
//
// KILL: FF-1 if 403, FF-3 if unparseable.
func TestFlightSearch(t *testing.T) {
	c := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Build filters using fli-compatible format
	filters := BuildFlightFilters("HEL", "NRT", "2026-06-15", 1)

	encoded, err := EncodeFlightFilters(filters)
	if err != nil {
		t.Fatalf("encode filters: %v", err)
	}

	t.Logf("Encoded payload length: %d chars", len(encoded))
	t.Logf("Encoded payload (first 300): %.300s", encoded)

	status, body, err := c.SearchFlights(ctx, encoded)
	if err != nil {
		t.Fatalf("flight search request failed: %v", err)
	}

	t.Logf("Status: %d", status)
	t.Logf("Raw response length: %d bytes", len(body))

	if status == 403 {
		t.Fatalf("FF-1 KILL: Google returned 403 — TLS fingerprint or request blocked")
	}
	if status == 400 {
		t.Logf("Raw response (first 2000): %s", truncate(body, 2000))
		t.Fatalf("400 Bad Request — payload format likely wrong, needs debugging")
	}
	if status != 200 {
		t.Logf("Raw response (first 1000): %s", truncate(body, 1000))
		t.Fatalf("unexpected status %d", status)
	}

	// Print raw response for analysis
	t.Logf("=== RAW FLIGHT RESPONSE (first 3000 chars) ===")
	t.Logf("%s", truncate(body, 3000))
	t.Logf("=== END RAW FLIGHT RESPONSE ===")

	// Try to decode
	inner, err := DecodeFlightResponse(body)
	if err != nil {
		t.Logf("FF-3 WARNING: could not decode flight response: %v", err)
		t.Logf("Full response (first 5000): %s", truncate(body, 5000))
		t.Fatalf("FF-3 KILL: response format unparseable")
	}

	// Pretty-print the decoded structure
	pretty, _ := json.MarshalIndent(inner, "", "  ")
	t.Logf("=== DECODED FLIGHT DATA (first 3000 chars) ===")
	t.Logf("%s", truncate(pretty, 3000))
	t.Logf("=== END DECODED ===")

	// Try to extract flight entries
	flights, err := ExtractFlightData(inner)
	if err != nil {
		t.Logf("WARNING: could not extract flight entries: %v", err)
		t.Log("This may mean the response structure differs from expected — check decoded output above")
	} else {
		t.Logf("SUCCESS: found %d flight entries", len(flights))
		// Print first flight entry for analysis
		if len(flights) > 0 {
			first, _ := json.MarshalIndent(flights[0], "", "  ")
			t.Logf("First flight entry (first 2000): %s", truncate(first, 2000))
		}
	}
}

// TestHotelSearch attempts a hotel search for Helsinki using batchexecute.
//
// rpcid "AtySUc" is the hotel search endpoint. The exact payload format is
// reverse-engineered and may need adjustment.
//
// KILL: FF-2 if empty/error.
func TestHotelSearch(t *testing.T) {
	c := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Try hotel search with AtySUc rpcid
	checkIn := [3]int{2026, 6, 15}
	checkOut := [3]int{2026, 6, 18}
	encoded := BuildHotelSearchPayload("Helsinki", checkIn, checkOut, 2)

	t.Logf("Hotel search encoded payload length: %d", len(encoded))

	status, body, err := c.BatchExecute(ctx, encoded)
	if err != nil {
		t.Fatalf("hotel search request failed: %v", err)
	}

	t.Logf("Status: %d", status)
	t.Logf("Raw response length: %d bytes", len(body))

	if status == 403 {
		t.Fatalf("FF-1 KILL: Google returned 403 — blocked")
	}

	t.Logf("=== RAW HOTEL SEARCH RESPONSE (first 3000 chars) ===")
	t.Logf("%s", truncate(body, 3000))
	t.Logf("=== END RAW HOTEL SEARCH RESPONSE ===")

	if status == 400 {
		t.Log("400 Bad Request — payload format needs adjustment")
		// Try alternative payload formats
		t.Log("Trying alternative hotel payload formats...")
		tryAlternativeHotelPayloads(t, c, ctx)
		return
	}

	if status != 200 {
		t.Logf("unexpected status %d", status)
		return
	}

	// Try to decode
	entries, err := DecodeBatchResponse(body)
	if err != nil {
		t.Logf("FF-3 WARNING: could not decode hotel response: %v", err)
	} else {
		t.Logf("Decoded %d entries from batch response", len(entries))
		for i, entry := range entries {
			pretty, _ := json.MarshalIndent(entry, "", "  ")
			t.Logf("=== HOTEL ENTRY %d (first 2000 chars) ===", i)
			t.Logf("%s", truncate(pretty, 2000))
		}
	}
}

// TestHotelPriceLookup attempts a hotel price lookup using yY52ce rpcid.
//
// This requires a known hotel ID. We try a few known Helsinki hotel IDs.
func TestHotelPriceLookup(t *testing.T) {
	c := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Known Google hotel ID format: /g/... or ChIJ...
	// Try a few common Helsinki hotel place IDs
	hotelIDs := []string{
		"/g/11b6d4_v_4",                     // Hotel Kamp Helsinki (well-known)
		"ChIJy7MSZP0LkkYRZw2dDekQP78",       // Another Helsinki hotel
		"/m/0dr7_h",                          // Hotel Kamp alt ID
	}

	checkIn := [3]int{2026, 6, 15}
	checkOut := [3]int{2026, 6, 18}

	for _, hotelID := range hotelIDs {
		t.Logf("--- Trying hotel ID: %s ---", hotelID)
		encoded := BuildHotelPricePayload(hotelID, checkIn, checkOut, "USD")

		status, body, err := c.BatchExecute(ctx, encoded)
		if err != nil {
			t.Logf("request failed for %s: %v", hotelID, err)
			continue
		}

		t.Logf("Status: %d, Body length: %d", status, len(body))
		t.Logf("Response (first 2000): %s", truncate(body, 2000))

		if status == 200 && len(body) > 100 {
			entries, err := DecodeBatchResponse(body)
			if err == nil {
				t.Logf("SUCCESS: decoded %d entries for hotel %s", len(entries), hotelID)
				for i, entry := range entries {
					pretty, _ := json.MarshalIndent(entry, "", "  ")
					t.Logf("Entry %d (first 1500): %s", i, truncate(pretty, 1500))
				}
				return
			}
		}
	}

	t.Log("WARNING: no hotel price lookups succeeded — hotel IDs may need updating")
}

// tryAlternativeHotelPayloads tries different payload structures for hotel search
// to discover the correct format.
func tryAlternativeHotelPayloads(t *testing.T, c *Client, ctx context.Context) {
	t.Helper()

	alternatives := []struct {
		name string
		args string
	}{
		{
			name: "minimal location",
			args: `["Helsinki"]`,
		},
		{
			name: "location with dates",
			args: fmt.Sprintf(`[null,"Helsinki",null,[%d,%d,%d],[%d,%d,%d],null,2]`,
				2026, 6, 15, 2026, 6, 18),
		},
		{
			name: "location as nested",
			args: `[null,null,"Helsinki",null,null,null,null,null,null,2,null,null,null,[2026,6,15],[2026,6,18]]`,
		},
		{
			name: "H1oSAb rpcid (hotel list)",
			args: `["Helsinki, Finland",[2026,6,15],[2026,6,18],2,null,null,null,null,null,null,"USD"]`,
		},
	}

	rpcids := []string{"AtySUc", "H1oSAb", "K2N1Nc"}

	for _, rpcid := range rpcids {
		for _, alt := range alternatives {
			encoded := EncodeBatchExecute(rpcid, alt.args)
			status, body, err := c.BatchExecute(ctx, encoded)
			if err != nil {
				t.Logf("[%s/%s] request error: %v", rpcid, alt.name, err)
				continue
			}
			bodyStr := string(body)
			hasData := len(body) > 200 && !strings.Contains(bodyStr, "error")

			t.Logf("[%s/%s] status=%d len=%d hasData=%v",
				rpcid, alt.name, status, len(body), hasData)

			if status == 200 && hasData {
				t.Logf("=== PROMISING: %s/%s (first 2000) ===", rpcid, alt.name)
				t.Logf("%s", truncate(body, 2000))
				return
			}
		}
	}

	t.Log("FF-2: No hotel search payload variant returned useful data")
}

func truncate(b []byte, maxLen int) string {
	if len(b) <= maxLen {
		return string(b)
	}
	return string(b[:maxLen]) + fmt.Sprintf("... [truncated, %d total]", len(b))
}
