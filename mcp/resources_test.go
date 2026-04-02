package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResourcesList(t *testing.T) {
	s := NewServer()
	resp := sendRequest(t, s, "resources/list", 1, nil)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	resultJSON, _ := json.Marshal(resp.Result)
	var result ResourcesListResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if len(result.Resources) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(result.Resources))
	}

	expected := map[string]bool{
		"trvl://airports/popular": false,
		"trvl://help/flights":    false,
		"trvl://help/hotels":     false,
	}
	for _, r := range result.Resources {
		if _, ok := expected[r.URI]; !ok {
			t.Errorf("unexpected resource: %s", r.URI)
		}
		expected[r.URI] = true
		if r.Name == "" {
			t.Errorf("resource %s has empty name", r.URI)
		}
	}
	for uri, found := range expected {
		if !found {
			t.Errorf("missing resource: %s", uri)
		}
	}
}

func TestResourcesRead_PopularAirports(t *testing.T) {
	s := NewServer()
	params := ResourcesReadParams{URI: "trvl://airports/popular"}
	resp := sendRequest(t, s, "resources/read", 1, params)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}

	resultJSON, _ := json.Marshal(resp.Result)
	var result ResourcesReadResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(result.Contents) == 0 {
		t.Fatal("expected contents")
	}
	text := result.Contents[0].Text
	if !strings.Contains(text, "HEL") {
		t.Error("airports should contain HEL")
	}
	if !strings.Contains(text, "JFK") {
		t.Error("airports should contain JFK")
	}
	// Should have 50 airports (50 lines).
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) != 50 {
		t.Errorf("expected 50 airports, got %d", len(lines))
	}
}

func TestResourcesRead_FlightGuide(t *testing.T) {
	s := NewServer()
	params := ResourcesReadParams{URI: "trvl://help/flights"}
	resp := sendRequest(t, s, "resources/read", 1, params)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}

	resultJSON, _ := json.Marshal(resp.Result)
	var result ResourcesReadResult
	json.Unmarshal(resultJSON, &result)

	if len(result.Contents) == 0 {
		t.Fatal("expected contents")
	}
	if !strings.Contains(result.Contents[0].Text, "search_flights") {
		t.Error("flight guide should mention search_flights")
	}
	if result.Contents[0].MimeType != "text/markdown" {
		t.Errorf("mime type = %q, want text/markdown", result.Contents[0].MimeType)
	}
}

func TestResourcesRead_HotelGuide(t *testing.T) {
	s := NewServer()
	params := ResourcesReadParams{URI: "trvl://help/hotels"}
	resp := sendRequest(t, s, "resources/read", 1, params)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}

	resultJSON, _ := json.Marshal(resp.Result)
	var result ResourcesReadResult
	json.Unmarshal(resultJSON, &result)

	if len(result.Contents) == 0 {
		t.Fatal("expected contents")
	}
	if !strings.Contains(result.Contents[0].Text, "search_hotels") {
		t.Error("hotel guide should mention search_hotels")
	}
}

func TestResourcesRead_NotFound(t *testing.T) {
	s := NewServer()
	params := ResourcesReadParams{URI: "trvl://nonexistent"}
	resp := sendRequest(t, s, "resources/read", 1, params)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error == nil {
		t.Fatal("expected error for unknown resource")
	}
}
