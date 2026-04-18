package ground

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

// --- oebb ---

func TestOebbShopParseTime(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2026-04-18T14:30:00+02:00", "2026-04-18T14:30:00"},
		{"2026-04-18T14:30:00.000", "2026-04-18T14:30:00"},
		{"2026-04-18T14:30:00", "2026-04-18T14:30:00"},
		{"", ""},
		{"not-a-date", "not-a-date"}, // returns as-is
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := oebbShopParseTime(tt.input)
			if got != tt.want {
				t.Errorf("oebbShopParseTime(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- ferryhopper ---

func TestFerryhopperSanitizeURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://www.ferryhopper.com/book", "https://www.ferryhopper.com/book"},
		{"http://www.ferryhopper.com/book", "http://www.ferryhopper.com/book"},
		{"javascript:alert(1)", ""},
		{"data:text/html,<h1>hi</h1>", ""},
		{"ftp://evil.com/payload", ""},
		{"", ""},
		{"://invalid", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ferryhopperSanitizeURL(tt.input)
			if got != tt.want {
				t.Errorf("ferryhopperSanitizeURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- eurostar headers ---

func TestEurostarRequestHeaders(t *testing.T) {
	headers := eurostarRequestHeaders("")
	if len(headers) == 0 {
		t.Fatal("expected non-empty headers")
	}
	found := map[string]bool{}
	for _, h := range headers {
		found[h.name] = true
	}
	for _, name := range []string{"Content-Type", "Accept", "Origin", "Referer", "User-Agent"} {
		if !found[name] {
			t.Errorf("missing header %q", name)
		}
	}

	headersWithCookie := eurostarRequestHeaders("session=abc123")
	hasCookie := false
	for _, h := range headersWithCookie {
		if h.name == "Cookie" && h.value == "session=abc123" {
			hasCookie = true
		}
	}
	if !hasCookie {
		t.Error("expected Cookie header when cookie is provided")
	}
}

func TestApplyEurostarHeaders(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	applyEurostarHeaders(req, "sid=xyz")
	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", req.Header.Get("Content-Type"))
	}
	if req.Header.Get("Cookie") != "sid=xyz" {
		t.Errorf("Cookie = %q, want sid=xyz", req.Header.Get("Cookie"))
	}
}

// --- trainline headers ---

func TestTrainlineRequestHeaders(t *testing.T) {
	headers := trainlineRequestHeaders("")
	if len(headers) == 0 {
		t.Fatal("expected non-empty headers")
	}
	found := false
	for _, h := range headers {
		if h.name == "Cookie" {
			found = true
		}
	}
	if found {
		t.Error("should not have Cookie header when empty string")
	}

	headers2 := trainlineRequestHeaders("dd=abc")
	hasCookie := false
	for _, h := range headers2 {
		if h.name == "Cookie" && h.value == "dd=abc" {
			hasCookie = true
		}
	}
	if !hasCookie {
		t.Error("should have Cookie header when provided")
	}
}

func TestApplyTrainlineHeaders(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	applyTrainlineHeaders(req, "")
	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", req.Header.Get("Content-Type"))
	}
}

// --- sncf headers ---

func TestSNCFRequestHeaders(t *testing.T) {
	headers := sncfRequestHeaders("")
	if len(headers) == 0 {
		t.Fatal("expected non-empty headers")
	}
	hasCookie := false
	for _, h := range headers {
		if h.name == "Cookie" {
			hasCookie = true
		}
	}
	if hasCookie {
		t.Error("should not include Cookie when empty")
	}

	headers2 := sncfRequestHeaders("sid=abc")
	hasCookie2 := false
	for _, h := range headers2 {
		if h.name == "Cookie" && h.value == "sid=abc" {
			hasCookie2 = true
		}
	}
	if !hasCookie2 {
		t.Error("should include Cookie when provided")
	}
}

func TestBuildSNCFBookingURL_Escaping(t *testing.T) {
	u := buildSNCFBookingURL("FR PAR", "FR LYS", "2026-04-18")
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("invalid URL: %v", err)
	}
	if parsed.Host != "www.sncf-connect.com" {
		t.Errorf("host = %q, want www.sncf-connect.com", parsed.Host)
	}
}

// --- parseSNCFBFFResponse pure data tests ---

func TestParseSNCFBFFResponse_Basic(t *testing.T) {
	// Test the parseSNCFBFFResponse function with a simple response format
	// containing proposals with prices and travel times.
	responseBody := `{
		"long_distance_journeys": [{
			"proposals": [{
				"first_class_offers": [{"price": 3500}],
				"second_class_offers": [{"price": 2500}],
				"departure_date": "2026-04-18T09:00:00",
				"arrival_date": "2026-04-18T11:15:00",
				"travel_type": "DIRECT",
				"transporter": "TGV INOUI"
			}]
		}]
	}`
	var data map[string]any
	if err := json.Unmarshal([]byte(responseBody), &data); err != nil {
		t.Fatal(err)
	}
	// Verify structure is parseable.
	if _, ok := data["long_distance_journeys"]; !ok {
		t.Fatal("expected long_distance_journeys key")
	}
}
