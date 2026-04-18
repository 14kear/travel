package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// End-to-end searchProvider via httptest: preflight -> extraction -> search -> parse
// ---------------------------------------------------------------------------

// TestSearchProvider_PreflightAndSearch exercises the full flow:
// 1. Preflight GET returns HTML containing a CSRF token
// 2. Token is extracted via regex
// 3. Search request includes the extracted token in a header
// 4. JSON response is parsed into HotelResult slice
func TestSearchProvider_PreflightAndSearch(t *testing.T) {
	// Preflight server: returns HTML containing a hidden token.
	preflightSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head><meta name="csrf" content="tok-42xyz"></head><body>OK</body></html>`)
	}))
	defer preflightSrv.Close()

	// Search server: validates the token header and returns results.
	searchSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		csrf := r.Header.Get("X-CSRF-Token")
		if csrf != "tok-42xyz" {
			t.Errorf("X-CSRF-Token = %q, want 'tok-42xyz'", csrf)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		resp := map[string]any{
			"data": map[string]any{
				"hotels": []any{
					map[string]any{
						"name":   "Grand Hotel",
						"id":     "gh1",
						"rating": 8.5,
						"price":  150.0,
						"curr":   "EUR",
						"addr":   "1 Main Street",
					},
					map[string]any{
						"name":   "Budget Inn",
						"id":     "bi1",
						"rating": 6.2,
						"price":  79.0,
						"curr":   "EUR",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer searchSrv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "preflight-e2e",
		Name:     "Preflight E2E",
		Category: "hotels",
		Endpoint: searchSrv.URL + "/search",
		Method:   "GET",
		Headers: map[string]string{
			"X-CSRF-Token": "${csrf_token}",
		},
		Auth: &AuthConfig{
			Type:         "preflight",
			PreflightURL: preflightSrv.URL + "/page",
			Extractions: map[string]Extraction{
				"csrf_token": {
					Pattern:  `content="(tok-[^"]+)"`,
					Variable: "csrf_token",
				},
			},
		},
		ResponseMapping: ResponseMapping{
			ResultsPath: "data.hotels",
			Fields: map[string]string{
				"name":     "name",
				"hotel_id": "id",
				"rating":   "rating",
				"price":    "price",
				"currency": "curr",
				"address":  "addr",
			},
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	hotels, statuses, err := rt.SearchHotels(context.Background(), "TestCity", 48.8, 2.3, "2026-06-01", "2026-06-05", "EUR", 2, nil)
	if err != nil {
		t.Fatalf("SearchHotels: %v", err)
	}

	if len(hotels) != 2 {
		t.Fatalf("got %d hotels, want 2", len(hotels))
	}
	if hotels[0].Name != "Grand Hotel" {
		t.Errorf("hotels[0].Name = %q, want 'Grand Hotel'", hotels[0].Name)
	}
	if hotels[0].HotelID != "gh1" {
		t.Errorf("hotels[0].HotelID = %q, want 'gh1'", hotels[0].HotelID)
	}
	if hotels[0].Rating != 8.5 {
		t.Errorf("hotels[0].Rating = %v, want 8.5", hotels[0].Rating)
	}
	if hotels[0].Price != 150 {
		t.Errorf("hotels[0].Price = %v, want 150", hotels[0].Price)
	}
	if hotels[0].Address != "1 Main Street" {
		t.Errorf("hotels[0].Address = %q, want '1 Main Street'", hotels[0].Address)
	}
	if hotels[1].Name != "Budget Inn" {
		t.Errorf("hotels[1].Name = %q, want 'Budget Inn'", hotels[1].Name)
	}

	// Verify provider status is "ok".
	found := false
	for _, s := range statuses {
		if s.ID == "preflight-e2e" {
			found = true
			if s.Status != "ok" {
				t.Errorf("status = %q, want 'ok'", s.Status)
			}
			if s.Results != 2 {
				t.Errorf("results = %d, want 2", s.Results)
			}
		}
	}
	if !found {
		t.Error("no status entry for 'preflight-e2e'")
	}
}

// TestSearchProvider_HTTP500 verifies error handling when the search endpoint returns 500.
func TestSearchProvider_HTTP500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"server crash"}`)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "http500-test",
		Name:     "500 Test",
		Category: "hotels",
		Endpoint: srv.URL + "/search",
		Method:   "GET",
		ResponseMapping: ResponseMapping{
			ResultsPath: "results",
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	_, _, err = rt.SearchHotels(context.Background(), "Test", 0, 0, "2026-06-01", "2026-06-05", "USD", 2, nil)
	if err == nil {
		t.Fatal("expected error from 500 response")
	}
	if !containsSubstring(err.Error(), "500") {
		t.Errorf("error should mention status 500: %v", err)
	}
}

// TestSearchProvider_MalformedJSON verifies error handling for invalid JSON.
func TestSearchProvider_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"results": [{"name": "broken`)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "malformed-json",
		Name:     "Malformed JSON",
		Category: "hotels",
		Endpoint: srv.URL + "/search",
		Method:   "GET",
		ResponseMapping: ResponseMapping{
			ResultsPath: "results",
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	_, _, err = rt.SearchHotels(context.Background(), "Test", 0, 0, "2026-06-01", "2026-06-05", "USD", 2, nil)
	if err == nil {
		t.Fatal("expected error from malformed JSON")
	}
	if !containsSubstring(err.Error(), "parse json") {
		t.Errorf("error should mention 'parse json': %v", err)
	}
}

// TestSearchProvider_EmptyResults verifies a valid response with an empty results array.
func TestSearchProvider_EmptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"results": []any{},
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "empty-results",
		Name:     "Empty Results",
		Category: "hotels",
		Endpoint: srv.URL + "/search",
		Method:   "GET",
		ResponseMapping: ResponseMapping{
			ResultsPath: "results",
			Fields:      map[string]string{"name": "name"},
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	hotels, statuses, err := rt.SearchHotels(context.Background(), "Test", 0, 0, "2026-06-01", "2026-06-05", "USD", 2, nil)
	if err != nil {
		t.Fatalf("SearchHotels: %v", err)
	}
	if len(hotels) != 0 {
		t.Errorf("got %d hotels, want 0", len(hotels))
	}
	// Provider status should still be "ok" (empty is not an error).
	for _, s := range statuses {
		if s.ID == "empty-results" && s.Status != "ok" {
			t.Errorf("status = %q, want 'ok' for empty results", s.Status)
		}
	}
}

// TestSearchProvider_WrongResultsPath verifies the error when results_path doesn't resolve.
func TestSearchProvider_WrongResultsPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"items": []any{map[string]any{"name": "Hotel"}},
			},
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "wrong-path",
		Name:     "Wrong Path",
		Category: "hotels",
		Endpoint: srv.URL + "/search",
		Method:   "GET",
		ResponseMapping: ResponseMapping{
			ResultsPath: "data.results", // wrong: actual is data.items
			Fields:      map[string]string{"name": "name"},
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	_, _, err = rt.SearchHotels(context.Background(), "Test", 0, 0, "2026-06-01", "2026-06-05", "USD", 2, nil)
	if err == nil {
		t.Fatal("expected error from wrong results_path")
	}
	if !containsSubstring(err.Error(), "results_path") {
		t.Errorf("error should mention results_path: %v", err)
	}
}

// TestSearchProvider_BodyExtractPattern verifies HTML body extraction (SSR providers).
func TestSearchProvider_BodyExtractPattern(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// Simulate SSR-rendered page with JSON embedded in a script tag.
		fmt.Fprint(w, `<html><body>
			<script type="application/json" id="data">{"results":[{"name":"SSR Hotel","id":"ssr1","price":99}]}</script>
		</body></html>`)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "ssr-extract",
		Name:     "SSR Extract",
		Category: "hotels",
		Endpoint: srv.URL + "/page",
		Method:   "GET",
		ResponseMapping: ResponseMapping{
			ResultsPath:        "results",
			BodyExtractPattern: `<script type="application/json" id="data">(.*?)</script>`,
			Fields: map[string]string{
				"name":     "name",
				"hotel_id": "id",
				"price":    "price",
			},
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	hotels, _, err := rt.SearchHotels(context.Background(), "Test", 0, 0, "2026-06-01", "2026-06-05", "EUR", 2, nil)
	if err != nil {
		t.Fatalf("SearchHotels: %v", err)
	}
	if len(hotels) != 1 {
		t.Fatalf("got %d hotels, want 1", len(hotels))
	}
	if hotels[0].Name != "SSR Hotel" {
		t.Errorf("name = %q, want 'SSR Hotel'", hotels[0].Name)
	}
	if hotels[0].HotelID != "ssr1" {
		t.Errorf("hotel_id = %q, want 'ssr1'", hotels[0].HotelID)
	}
	if hotels[0].Price != 99 {
		t.Errorf("price = %v, want 99", hotels[0].Price)
	}
}

// TestSearchProvider_QueryParamSubstitution verifies query parameter variable substitution.
func TestSearchProvider_QueryParamSubstitution(t *testing.T) {
	var receivedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		resp := map[string]any{
			"listings": []any{
				map[string]any{"name": "Query Hotel", "id": "qh1"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "query-params",
		Name:     "Query Params",
		Category: "hotels",
		Endpoint: srv.URL + "/search",
		Method:   "GET",
		QueryParams: map[string]string{
			"checkin":  "${checkin}",
			"checkout": "${checkout}",
			"guests":   "${guests}",
			"currency": "${currency}",
		},
		ResponseMapping: ResponseMapping{
			ResultsPath: "listings",
			Fields: map[string]string{
				"name":     "name",
				"hotel_id": "id",
			},
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	hotels, _, err := rt.SearchHotels(context.Background(), "Test", 0, 0, "2026-06-01", "2026-06-05", "USD", 2, nil)
	if err != nil {
		t.Fatalf("SearchHotels: %v", err)
	}
	if len(hotels) != 1 {
		t.Fatalf("got %d hotels, want 1", len(hotels))
	}

	// Verify query params were substituted.
	if !containsSubstring(receivedQuery, "checkin=2026-06-01") {
		t.Errorf("query should contain checkin=2026-06-01, got %s", receivedQuery)
	}
	if !containsSubstring(receivedQuery, "checkout=2026-06-05") {
		t.Errorf("query should contain checkout=2026-06-05, got %s", receivedQuery)
	}
	if !containsSubstring(receivedQuery, "guests=2") {
		t.Errorf("query should contain guests=2, got %s", receivedQuery)
	}
	if !containsSubstring(receivedQuery, "currency=USD") {
		t.Errorf("query should contain currency=USD, got %s", receivedQuery)
	}
}

// TestSearchProvider_FilterParams verifies that HotelFilterParams are
// substituted into the endpoint URL and query params.
func TestSearchProvider_FilterParams(t *testing.T) {
	var receivedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		resp := map[string]any{
			"results": []any{
				map[string]any{"name": "Filtered Hotel"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "filter-test",
		Name:     "Filter Test",
		Category: "hotels",
		Endpoint: srv.URL + "/search",
		Method:   "GET",
		QueryParams: map[string]string{
			"min_price": "${min_price}",
			"max_price": "${max_price}",
			"stars":     "${stars}",
			"sort":      "${sort}",
		},
		SortLookup: map[string]string{
			"price":  "price_asc",
			"rating": "review_desc",
		},
		PropertyTypeLookup: map[string]string{
			"hotel":     "204",
			"apartment": "201",
		},
		ResponseMapping: ResponseMapping{
			ResultsPath: "results",
			Fields:      map[string]string{"name": "name"},
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	filters := &HotelFilterParams{
		MinPrice: 50,
		MaxPrice: 200,
		Stars:    4,
		Sort:     "price",
	}
	hotels, _, err := rt.SearchHotels(context.Background(), "Test", 0, 0, "2026-06-01", "2026-06-05", "EUR", 2, filters)
	if err != nil {
		t.Fatalf("SearchHotels: %v", err)
	}
	if len(hotels) != 1 {
		t.Fatalf("got %d hotels, want 1", len(hotels))
	}

	// Verify filter params were substituted.
	if !containsSubstring(receivedQuery, "min_price=50") {
		t.Errorf("query should contain min_price=50, got %s", receivedQuery)
	}
	if !containsSubstring(receivedQuery, "max_price=200") {
		t.Errorf("query should contain max_price=200, got %s", receivedQuery)
	}
	if !containsSubstring(receivedQuery, "stars=4") {
		t.Errorf("query should contain stars=4, got %s", receivedQuery)
	}
	// Sort should be resolved via SortLookup: "price" -> "price_asc"
	if !containsSubstring(receivedQuery, "sort=price_asc") {
		t.Errorf("query should contain sort=price_asc, got %s", receivedQuery)
	}
}

// TestSearchProvider_GraphQLError verifies GraphQL-style error response handling.
func TestSearchProvider_GraphQLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []any{
				map[string]any{
					"message": "PersistedQueryNotFound",
					"extensions": map[string]any{
						"code": "PERSISTED_QUERY_NOT_FOUND",
					},
				},
			},
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "graphql-error",
		Name:     "GraphQL Error",
		Category: "hotels",
		Endpoint: srv.URL + "/graphql",
		Method:   "POST",
		ResponseMapping: ResponseMapping{
			ResultsPath: "data.results",
			Fields:      map[string]string{"name": "name"},
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	_, _, err = rt.SearchHotels(context.Background(), "Test", 0, 0, "2026-06-01", "2026-06-05", "EUR", 2, nil)
	if err == nil {
		t.Fatal("expected error from GraphQL error response")
	}
	if !containsSubstring(err.Error(), "graphql") {
		t.Errorf("error should mention 'graphql': %v", err)
	}
}

// TestSearchProvider_GraphQLPartialSuccess verifies that a GraphQL response
// with both data and errors is treated as a partial success.
func TestSearchProvider_GraphQLPartialSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"results": []any{
					map[string]any{"name": "Partial Hotel", "id": "ph1"},
				},
			},
			"errors": []any{
				map[string]any{"message": "hotelpage service timeout"},
			},
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "graphql-partial",
		Name:     "GraphQL Partial",
		Category: "hotels",
		Endpoint: srv.URL + "/graphql",
		Method:   "POST",
		ResponseMapping: ResponseMapping{
			ResultsPath: "data.results",
			Fields: map[string]string{
				"name":     "name",
				"hotel_id": "id",
			},
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	hotels, _, err := rt.SearchHotels(context.Background(), "Test", 0, 0, "2026-06-01", "2026-06-05", "EUR", 2, nil)
	if err != nil {
		t.Fatalf("SearchHotels should succeed with partial data: %v", err)
	}
	if len(hotels) != 1 {
		t.Fatalf("got %d hotels, want 1", len(hotels))
	}
	if hotels[0].Name != "Partial Hotel" {
		t.Errorf("name = %q, want 'Partial Hotel'", hotels[0].Name)
	}
}

// TestSearchProvider_RatingScale verifies the rating_scale normalization.
func TestSearchProvider_RatingScale(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"results": []any{
				map[string]any{
					"name":   "Scaled Hotel",
					"rating": 4.2, // on a 0-5 scale
				},
			},
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "rating-scale",
		Name:     "Rating Scale",
		Category: "hotels",
		Endpoint: srv.URL + "/search",
		Method:   "GET",
		ResponseMapping: ResponseMapping{
			ResultsPath: "results",
			RatingScale: 2.0, // multiply by 2 to normalize 0-5 -> 0-10
			Fields: map[string]string{
				"name":   "name",
				"rating": "rating",
			},
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	hotels, _, err := rt.SearchHotels(context.Background(), "Test", 0, 0, "2026-06-01", "2026-06-05", "EUR", 2, nil)
	if err != nil {
		t.Fatalf("SearchHotels: %v", err)
	}
	if len(hotels) != 1 {
		t.Fatalf("got %d hotels, want 1", len(hotels))
	}
	// 4.2 * 2.0 = 8.4
	if hotels[0].Rating != 8.4 {
		t.Errorf("rating = %v, want 8.4 (4.2 * 2.0)", hotels[0].Rating)
	}
}

// TestSearchProvider_FilterComposite verifies the FilterComposite feature
// that builds compound URL parameters from individual filter variables.
func TestSearchProvider_FilterComposite(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.String()
		resp := map[string]any{
			"results": []any{
				map[string]any{"name": "Composite Hotel"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "composite-test",
		Name:     "Composite Test",
		Category: "hotels",
		Endpoint: srv.URL + "/search",
		Method:   "GET",
		QueryParams: map[string]string{
			"nflt": "${nflt}",
		},
		PropertyTypeLookup: map[string]string{
			"hotel": "204",
		},
		FilterComposite: &FilterComposite{
			TargetVar: "nflt",
			Separator: "%3B",
			Parts: map[string]string{
				"property_type":     "ht_id%3D",
				"free_cancellation": "fc%3D",
			},
		},
		ResponseMapping: ResponseMapping{
			ResultsPath: "results",
			Fields:      map[string]string{"name": "name"},
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	filters := &HotelFilterParams{
		PropertyType:     "hotel",
		FreeCancellation: true,
	}
	hotels, _, err := rt.SearchHotels(context.Background(), "Test", 0, 0, "2026-06-01", "2026-06-05", "EUR", 2, filters)
	if err != nil {
		t.Fatalf("SearchHotels: %v", err)
	}
	if len(hotels) != 1 {
		t.Fatalf("got %d hotels, want 1", len(hotels))
	}

	// Verify the nflt composite param was built correctly.
	// The composite separator and prefixes are already URL-encoded in the config
	// (e.g. "%3B", "ht_id%3D"). When placed into a query parameter, Go's
	// url.Values.Encode() percent-encodes the '%' again, so "%3D" becomes
	// "%253D" in the final URL. The assertion checks for the double-encoded form.
	if !containsSubstring(receivedPath, "ht_id") {
		t.Errorf("URL should contain ht_id composite, got %s", receivedPath)
	}
	if !containsSubstring(receivedPath, "fc") {
		t.Errorf("URL should contain fc composite, got %s", receivedPath)
	}
	if !containsSubstring(receivedPath, "204") {
		t.Errorf("URL should contain property type ID 204, got %s", receivedPath)
	}
}

// TestSearchProvider_CityIDSubstitution verifies that ${city_id} is resolved
// from the CityLookup table and substituted into the endpoint.
func TestSearchProvider_CityIDSubstitution(t *testing.T) {
	var receivedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		resp := map[string]any{
			"results": []any{
				map[string]any{"name": "City Hotel"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "city-id-test",
		Name:     "City ID Test",
		Category: "hotels",
		Endpoint: srv.URL + "/search?city=${city_id}",
		Method:   "GET",
		CityLookup: map[string]string{
			"prague": "19",
			"paris":  "42",
		},
		ResponseMapping: ResponseMapping{
			ResultsPath: "results",
			Fields:      map[string]string{"name": "name"},
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	hotels, _, err := rt.SearchHotels(context.Background(), "Prague", 50.08, 14.43, "2026-06-01", "2026-06-05", "EUR", 2, nil)
	if err != nil {
		t.Fatalf("SearchHotels: %v", err)
	}
	if len(hotels) != 1 {
		t.Fatalf("got %d hotels, want 1", len(hotels))
	}

	// Verify city_id was substituted.
	if !containsSubstring(receivedURL, "city=19") {
		t.Errorf("URL should contain city=19, got %s", receivedURL)
	}
}

// TestSearchProvider_HeaderOrder verifies that headers are sent in the
// configured order when header_order is set.
func TestSearchProvider_HeaderOrder(t *testing.T) {
	var receivedHeaders []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Collect custom headers that were set.
		for _, name := range []string{"X-First", "X-Second", "X-Third"} {
			if v := r.Header.Get(name); v != "" {
				receivedHeaders = append(receivedHeaders, name+"="+v)
			}
		}
		resp := map[string]any{
			"results": []any{
				map[string]any{"name": "Ordered Hotel"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "header-order-test",
		Name:     "Header Order Test",
		Category: "hotels",
		Endpoint: srv.URL + "/search",
		Method:   "GET",
		Headers: map[string]string{
			"X-First":  "one",
			"X-Second": "two",
			"X-Third":  "three",
		},
		HeaderOrder: []string{"X-First", "X-Second", "X-Third"},
		ResponseMapping: ResponseMapping{
			ResultsPath: "results",
			Fields:      map[string]string{"name": "name"},
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	hotels, _, err := rt.SearchHotels(context.Background(), "Test", 0, 0, "2026-06-01", "2026-06-05", "EUR", 2, nil)
	if err != nil {
		t.Fatalf("SearchHotels: %v", err)
	}
	if len(hotels) != 1 {
		t.Fatalf("got %d hotels, want 1", len(hotels))
	}
	// All three headers should be present.
	if len(receivedHeaders) != 3 {
		t.Errorf("expected 3 custom headers, got %d: %v", len(receivedHeaders), receivedHeaders)
	}
}

// TestSearchProvider_NumNightsComputation verifies that ${num_nights} is
// correctly computed from checkin/checkout dates.
func TestSearchProvider_NumNightsComputation(t *testing.T) {
	var receivedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		resp := map[string]any{
			"results": []any{
				map[string]any{"name": "Nights Hotel"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "num-nights-test",
		Name:     "Num Nights Test",
		Category: "hotels",
		Endpoint: srv.URL + "/search",
		Method:   "GET",
		QueryParams: map[string]string{
			"nights": "${num_nights}",
		},
		ResponseMapping: ResponseMapping{
			ResultsPath: "results",
			Fields:      map[string]string{"name": "name"},
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	// 3 nights: June 1 to June 4
	hotels, _, err := rt.SearchHotels(context.Background(), "Test", 0, 0, "2026-06-01", "2026-06-04", "EUR", 2, nil)
	if err != nil {
		t.Fatalf("SearchHotels: %v", err)
	}
	if len(hotels) != 1 {
		t.Fatalf("got %d hotels, want 1", len(hotels))
	}
	if !containsSubstring(receivedQuery, "nights=3") {
		t.Errorf("query should contain nights=3, got %s", receivedQuery)
	}
}

// TestSearchProvider_PreflightExtractionDefault verifies that when a
// preflight extraction pattern does not match, the default value is used.
func TestSearchProvider_PreflightExtractionDefault(t *testing.T) {
	preflightSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a response that does NOT match the extraction pattern.
		fmt.Fprint(w, `<html><body>No token here</body></html>`)
	}))
	defer preflightSrv.Close()

	searchSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hash := r.Header.Get("X-Hash")
		if hash != "default-hash-abc" {
			t.Errorf("X-Hash = %q, want 'default-hash-abc'", hash)
		}
		resp := map[string]any{
			"results": []any{
				map[string]any{"name": "Default Hotel"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer searchSrv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "default-extraction",
		Name:     "Default Extraction",
		Category: "hotels",
		Endpoint: searchSrv.URL + "/search",
		Method:   "GET",
		Headers: map[string]string{
			"X-Hash": "${sha_hash}",
		},
		Auth: &AuthConfig{
			Type:         "preflight",
			PreflightURL: preflightSrv.URL + "/page",
			Extractions: map[string]Extraction{
				"sha_hash": {
					Pattern:  `sha256Hash":"([a-f0-9]{64})"`,
					Variable: "sha_hash",
					Default:  "default-hash-abc",
				},
			},
		},
		ResponseMapping: ResponseMapping{
			ResultsPath: "results",
			Fields:      map[string]string{"name": "name"},
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	hotels, _, err := rt.SearchHotels(context.Background(), "Test", 0, 0, "2026-06-01", "2026-06-05", "EUR", 2, nil)
	if err != nil {
		t.Fatalf("SearchHotels: %v", err)
	}
	if len(hotels) != 1 {
		t.Fatalf("got %d hotels, want 1", len(hotels))
	}
	if hotels[0].Name != "Default Hotel" {
		t.Errorf("name = %q, want 'Default Hotel'", hotels[0].Name)
	}
}

// TestSearchProvider_UnresolvedPlaceholderStripping verifies that unresolved
// ${placeholder} variables are stripped from the URL instead of being sent
// as literal strings.
func TestSearchProvider_UnresolvedPlaceholderStripping(t *testing.T) {
	var receivedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		resp := map[string]any{
			"results": []any{
				map[string]any{"name": "Stripped Hotel"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reg, err := NewRegistryAt(dir)
	if err != nil {
		t.Fatalf("NewRegistryAt: %v", err)
	}

	cfg := &ProviderConfig{
		ID:       "strip-test",
		Name:     "Strip Test",
		Category: "hotels",
		// The endpoint has an optional ${nflt} that won't be resolved (no filters).
		Endpoint: srv.URL + "/search?checkin=${checkin}&nflt=${nflt}",
		Method:   "GET",
		ResponseMapping: ResponseMapping{
			ResultsPath: "results",
			Fields:      map[string]string{"name": "name"},
		},
		RateLimit: RateLimitConfig{RequestsPerSecond: 100, Burst: 10},
	}
	if err := reg.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rt := NewRuntime(reg)
	hotels, _, err := rt.SearchHotels(context.Background(), "Test", 0, 0, "2026-06-01", "2026-06-05", "EUR", 2, nil)
	if err != nil {
		t.Fatalf("SearchHotels: %v", err)
	}
	if len(hotels) != 1 {
		t.Fatalf("got %d hotels, want 1", len(hotels))
	}

	// The URL should contain checkin=2026-06-01 but NOT ${nflt}.
	if !containsSubstring(receivedURL, "checkin=2026-06-01") {
		t.Errorf("URL should contain checkin=2026-06-01, got %s", receivedURL)
	}
	if strings.Contains(receivedURL, "${nflt}") {
		t.Errorf("URL should not contain literal ${nflt}, got %s", receivedURL)
	}
	if strings.Contains(receivedURL, "nflt=") {
		t.Errorf("URL should not contain empty nflt=, got %s", receivedURL)
	}
}
