package providers

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCookieDomainMatchesHost(t *testing.T) {
	cases := []struct {
		name         string
		cookieDomain string
		host         string
		want         bool
	}{
		{"exact", "booking.com", "booking.com", true},
		{"dot prefix parent", ".booking.com", "www.booking.com", true},
		{"parent no dot", "booking.com", "www.booking.com", true},
		{"unrelated", "example.com", "booking.com", false},
		{"suffix only collision", "oking.com", "booking.com", false},
		{"empty cookie", "", "booking.com", false},
		{"empty host", "booking.com", "", false},
		{"subdomain mismatch", "api.booking.com", "booking.com", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := cookieDomainMatchesHost(tc.cookieDomain, tc.host); got != tc.want {
				t.Errorf("cookieDomainMatchesHost(%q, %q) = %v, want %v", tc.cookieDomain, tc.host, got, tc.want)
			}
		})
	}
}

func TestRegistrableSuffix(t *testing.T) {
	cases := map[string]string{
		"booking.com":          "booking.com",
		"www.booking.com":      "booking.com",
		"a.b.c.booking.com":    "booking.com",
		"localhost":            "localhost",
		".leading.booking.com": "booking.com",
	}
	for in, want := range cases {
		if got := registrableSuffix(in); got != want {
			t.Errorf("registrableSuffix(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNeedsBrowserCookieFallback(t *testing.T) {
	extractions := map[string]Extraction{"csrf": {Pattern: `x`, Variable: "csrf"}}
	cases := []struct {
		name       string
		status     int
		extracted  int
		extractors map[string]Extraction
		want       bool
	}{
		{"200 all matched", 200, 1, extractions, false},
		{"200 none matched", 200, 0, extractions, true},
		{"202 challenge", 202, 0, extractions, true},
		{"403 forbidden", 403, 0, extractions, true},
		{"202 but matched", 202, 1, extractions, true},
		{"200 no extractions", 200, 0, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := needsBrowserCookieFallback(tc.status, tc.extracted, tc.extractors)
			if got != tc.want {
				t.Errorf("needsBrowserCookieFallback(%d, %d, %v) = %v, want %v",
					tc.status, tc.extracted, tc.extractors, got, tc.want)
			}
		})
	}
}

// TestApplyBrowserCookies_NilJar ensures the helper fails safely when no
// cookie jar is configured.
func TestApplyBrowserCookies_NilJar(t *testing.T) {
	client := &http.Client{}
	if applyBrowserCookies(client, "https://example.com") {
		t.Error("expected false when client has no jar")
	}
}

// TestApplyBrowserCookies_BadURL ensures the helper fails safely on bad URLs.
func TestApplyBrowserCookies_BadURL(t *testing.T) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	if applyBrowserCookies(client, "::not a url::") {
		t.Error("expected false for bad URL")
	}
}

// TestBrowserCookiesForURL_BadURL ensures safe handling of malformed URLs.
func TestBrowserCookiesForURL_BadURL(t *testing.T) {
	if got := browserCookiesForURL("::not a url::"); got != nil {
		t.Errorf("expected nil for bad URL, got %d cookies", len(got))
	}
	if got := browserCookiesForURL(""); got != nil {
		t.Errorf("expected nil for empty URL, got %d cookies", len(got))
	}
}

// TestBrowserCookiesForURL_UnknownDomain ensures we don't crash when no
// browser store has cookies for a random domain.
func TestBrowserCookiesForURL_UnknownDomain(t *testing.T) {
	// Serve a random .invalid domain that no browser will have cookies for.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	// Just ensure the call returns without panicking. Whether it returns
	// cookies depends on the test environment.
	_ = browserCookiesForURL(u.String())
}
