package providers

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/all" // register all browser cookie finders
)

// browserCookieLookupTimeout bounds how long we spend reading cookies from
// browser stores. Local SQLite reads are fast but Keychain prompts on macOS
// can block indefinitely; a short deadline lets us fail fast.
const browserCookieLookupTimeout = 5 * time.Second

// browserCookiesForURL reads cookies from the user's browsers matching the
// given URL's domain. Iterates all registered browser cookie stores and
// returns every cookie whose domain matches the URL host (or is a parent
// domain of it). Returns nil if the URL cannot be parsed, no cookies are
// found, or cookie access fails (e.g. user denied Keychain access on macOS).
//
// This is used as a fallback when standard HTTP preflight gets blocked by
// JavaScript bot-detection challenges (HTTP 202/403). The user's actual
// browser has already solved any JS challenges and has valid session
// cookies, which we can read directly from their disk-backed cookie jars.
func browserCookiesForURL(targetURL string) []*http.Cookie {
	u, err := url.Parse(targetURL)
	if err != nil || u.Host == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), browserCookieLookupTimeout)
	defer cancel()

	host := u.Hostname()
	cookies, err := kooky.ReadCookies(ctx, kooky.Valid, kooky.DomainHasSuffix(registrableSuffix(host)))
	if err != nil && len(cookies) == 0 {
		return nil
	}

	result := make([]*http.Cookie, 0, len(cookies))
	seen := make(map[string]struct{}, len(cookies))
	for _, c := range cookies {
		if c == nil {
			continue
		}
		if !cookieDomainMatchesHost(c.Cookie.Domain, host) {
			continue
		}
		key := c.Cookie.Name + "\x00" + c.Cookie.Domain + "\x00" + c.Cookie.Path
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}

		cp := c.Cookie // copy
		result = append(result, &cp)
	}
	return result
}

// registrableSuffix returns a suffix of host suitable for a DomainHasSuffix
// filter. For e.g. "www.booking.com" it returns "booking.com"; for short
// hosts it returns the original host. This is a heuristic — we filter
// precisely afterwards in cookieDomainMatchesHost.
func registrableSuffix(host string) string {
	host = strings.TrimPrefix(host, ".")
	parts := strings.Split(host, ".")
	if len(parts) <= 2 {
		return host
	}
	return strings.Join(parts[len(parts)-2:], ".")
}

// cookieDomainMatchesHost reports whether a cookie's Domain attribute applies
// to the given request host per RFC 6265: the cookie domain must equal the
// host or be a dot-prefixed parent domain.
func cookieDomainMatchesHost(cookieDomain, host string) bool {
	if cookieDomain == "" || host == "" {
		return false
	}
	cd := strings.ToLower(strings.TrimPrefix(cookieDomain, "."))
	h := strings.ToLower(host)
	if cd == h {
		return true
	}
	return strings.HasSuffix(h, "."+cd)
}
