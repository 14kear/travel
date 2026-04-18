package testutil

import (
	"net/http"
	"net/http/httptest"
	"strings"
)

// MockResponse defines a canned HTTP response for a specific URL path.
type MockResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// NewMockServer creates an httptest.Server that returns canned responses
// based on URL path. Paths are matched by prefix: the request path must
// start with the registered key. If no match is found, HTTP 404 is returned.
//
// Usage:
//
//	srv := testutil.NewMockServer(map[string]testutil.MockResponse{
//	    "/search": {StatusCode: 200, Body: `{"results":[]}`},
//	    "/auth":   {StatusCode: 200, Body: `token=abc`},
//	})
//	defer srv.Close()
func NewMockServer(responses map[string]MockResponse) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try exact path match first, then prefix match for longest match.
		if resp, ok := responses[r.URL.Path]; ok {
			writeResponse(w, resp)
			return
		}
		// Prefix match: find the longest matching prefix.
		bestPath := ""
		for path := range responses {
			if strings.HasPrefix(r.URL.Path, path) && len(path) > len(bestPath) {
				bestPath = path
			}
		}
		if bestPath != "" {
			writeResponse(w, responses[bestPath])
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func writeResponse(w http.ResponseWriter, resp MockResponse) {
	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}
	code := resp.StatusCode
	if code == 0 {
		code = http.StatusOK
	}
	w.WriteHeader(code)
	_, _ = w.Write([]byte(resp.Body))
}
