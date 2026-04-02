package batchexec

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestEncodeBatchExecute(t *testing.T) {
	result := EncodeBatchExecute("AtySUc", `["Helsinki"]`)

	// The result should be URL-encoded.
	decoded, err := url.QueryUnescape(result)
	if err != nil {
		t.Fatalf("unescape: %v", err)
	}

	// The decoded result should be valid JSON.
	var outer []any
	if err := json.Unmarshal([]byte(decoded), &outer); err != nil {
		t.Fatalf("unmarshal outer: %v", err)
	}

	// Structure: [[[rpcid, args_json, null, "generic"]]]
	if len(outer) != 1 {
		t.Fatalf("expected 1 outer element, got %d", len(outer))
	}
	mid, ok := outer[0].([]any)
	if !ok || len(mid) != 1 {
		t.Fatalf("expected 1 mid element, got %v", outer[0])
	}
	inner, ok := mid[0].([]any)
	if !ok || len(inner) != 4 {
		t.Fatalf("expected 4 inner elements, got %v", mid[0])
	}

	if inner[0] != "AtySUc" {
		t.Errorf("rpcid = %v, want AtySUc", inner[0])
	}
	if inner[1] != `["Helsinki"]` {
		t.Errorf("args = %v, want [\"Helsinki\"]", inner[1])
	}
	if inner[2] != nil {
		t.Errorf("inner[2] = %v, want nil", inner[2])
	}
	if inner[3] != "generic" {
		t.Errorf("inner[3] = %v, want generic", inner[3])
	}
}

func TestEncodeFlightFilters(t *testing.T) {
	filters := BuildFlightFilters("HEL", "NRT", "2026-06-15", 1)

	encoded, err := EncodeFlightFilters(filters)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	// Should be non-empty URL-encoded string.
	if len(encoded) == 0 {
		t.Fatal("encoded is empty")
	}

	// Should be valid URL encoding.
	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		t.Fatalf("unescape: %v", err)
	}

	// The decoded string should be valid JSON: [null, "<json-string>"]
	var wrapper []any
	if err := json.Unmarshal([]byte(decoded), &wrapper); err != nil {
		t.Fatalf("unmarshal wrapper: %v", err)
	}
	if len(wrapper) != 2 {
		t.Fatalf("expected 2 wrapper elements, got %d", len(wrapper))
	}
	if wrapper[0] != nil {
		t.Errorf("wrapper[0] = %v, want nil", wrapper[0])
	}

	// wrapper[1] is a JSON string containing the filters.
	filtersJSON, ok := wrapper[1].(string)
	if !ok {
		t.Fatalf("wrapper[1] not string, got %T", wrapper[1])
	}

	var filtersArr []any
	if err := json.Unmarshal([]byte(filtersJSON), &filtersArr); err != nil {
		t.Fatalf("unmarshal filters: %v", err)
	}

	// Should have 6 top-level elements.
	if len(filtersArr) != 6 {
		t.Fatalf("expected 6 filter elements, got %d", len(filtersArr))
	}
}

func TestBuildFlightFilters(t *testing.T) {
	filters := BuildFlightFilters("HEL", "NRT", "2026-06-15", 1)

	data, err := json.Marshal(filters)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var arr []any
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(arr) != 6 {
		t.Fatalf("expected 6 top elements, got %d", len(arr))
	}

	// arr[1] = settings
	settings, ok := arr[1].([]any)
	if !ok {
		t.Fatalf("settings not array")
	}

	// Trip type = 2 (one-way)
	if v, ok := settings[2].(float64); !ok || int(v) != 2 {
		t.Errorf("trip type = %v, want 2", settings[2])
	}

	// Cabin = 1 (economy)
	if v, ok := settings[5].(float64); !ok || int(v) != 1 {
		t.Errorf("cabin = %v, want 1", settings[5])
	}

	// Passengers = [1, 0, 0, 0]
	pax, ok := settings[6].([]any)
	if !ok || len(pax) != 4 {
		t.Fatalf("passengers not [4]array")
	}
	if v, ok := pax[0].(float64); !ok || int(v) != 1 {
		t.Errorf("adults = %v, want 1", pax[0])
	}

	// Segments at settings[13]
	segments, ok := settings[13].([]any)
	if !ok || len(segments) != 1 {
		t.Fatalf("segments: expected 1, got %v", settings[13])
	}

	segment, ok := segments[0].([]any)
	if !ok || len(segment) < 7 {
		t.Fatalf("segment too short")
	}

	// Date at segment[6]
	if segment[6] != "2026-06-15" {
		t.Errorf("segment date = %v, want 2026-06-15", segment[6])
	}
}

func TestBuildHotelSearchPayload(t *testing.T) {
	result := BuildHotelSearchPayload("Helsinki", [3]int{2026, 6, 15}, [3]int{2026, 6, 18}, 2)

	// Should be non-empty URL-encoded string.
	if len(result) == 0 {
		t.Fatal("result is empty")
	}

	decoded, err := url.QueryUnescape(result)
	if err != nil {
		t.Fatalf("unescape: %v", err)
	}

	// Should be valid JSON.
	var outer []any
	if err := json.Unmarshal([]byte(decoded), &outer); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Structure: [[[rpcid, args, null, "generic"]]]
	if len(outer) != 1 {
		t.Fatalf("expected 1 outer, got %d", len(outer))
	}
}

func TestBuildHotelPricePayload(t *testing.T) {
	result := BuildHotelPricePayload("/g/11test", [3]int{2026, 6, 15}, [3]int{2026, 6, 18}, "USD")

	if len(result) == 0 {
		t.Fatal("result is empty")
	}

	decoded, err := url.QueryUnescape(result)
	if err != nil {
		t.Fatalf("unescape: %v", err)
	}

	var outer []any
	if err := json.Unmarshal([]byte(decoded), &outer); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Should contain the rpcid "yY52ce"
	mid := outer[0].([]any)
	inner := mid[0].([]any)
	if inner[0] != "yY52ce" {
		t.Errorf("rpcid = %v, want yY52ce", inner[0])
	}
}
