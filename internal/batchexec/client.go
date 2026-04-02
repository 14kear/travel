// Package batchexec provides an HTTP client for Google's internal batchexecute API.
//
// Google's travel frontends (Flights, Hotels) communicate via a protocol that
// POSTs form-encoded "f.req" payloads and returns JSON with an anti-XSSI prefix.
// This package handles TLS fingerprint impersonation (Chrome via utls), request
// encoding, and response decoding.
package batchexec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	utls "github.com/refraction-networking/utls"
)

// Endpoint constants for Google Travel APIs.
const (
	FlightsURL = "https://www.google.com/_/FlightsFrontendUi/data/travel.frontend.flights.FlightsFrontendService/GetShoppingResults"
	HotelsURL  = "https://www.google.com/_/TravelFrontendUi/data/batchexecute"
)

// chromeUA is a recent Chrome User-Agent string.
const chromeUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// Client wraps an http.Client with Chrome TLS fingerprint impersonation via utls.
type Client struct {
	http *http.Client
}

// NewClient creates a Client that impersonates Chrome's TLS fingerprint.
//
// Chrome's ClientHello is used for TLS fingerprinting, but we force HTTP/1.1
// via ALPN to avoid the complexity of HTTP/2 framing with custom TLS connections.
// Google's servers support HTTP/1.1 and this is sufficient for API access.
func NewClient() *Client {
	transport := &http.Transport{
		DialTLSContext:      dialTLSChromeHTTP1,
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		// Force HTTP/1.1 — we handle TLS ourselves and net/http can't do HTTP/2
		// on externally-provided TLS connections without extra wiring.
		ForceAttemptHTTP2: false,
	}
	return &Client{
		http: &http.Client{
			Transport: transport,
			Timeout:   20 * time.Second,
		},
	}
}

// dialTLSChromeHTTP1 dials a TCP connection and wraps it with a utls client
// that impersonates Chrome's TLS ClientHello but forces HTTP/1.1 via ALPN.
//
// We start from HelloChrome_Auto's spec (Chrome-like cipher suites, extensions,
// curves, etc.) but override the ALPN extension to only advertise "http/1.1".
// The UClient is created with HelloCustom so ApplyPreset installs our modified
// spec rather than ignoring it in favour of a built-in profile.
func dialTLSChromeHTTP1(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	rawConn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, fmt.Errorf("dial tcp: %w", err)
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		rawConn.Close()
		return nil, fmt.Errorf("split host: %w", err)
	}

	// Build a Chrome-like spec but with ALPN forced to HTTP/1.1.
	spec, err := utls.UTLSIdToSpec(utls.HelloChrome_Auto)
	if err != nil {
		rawConn.Close()
		return nil, fmt.Errorf("utls spec: %w", err)
	}
	for _, ext := range spec.Extensions {
		if alpn, ok := ext.(*utls.ALPNExtension); ok {
			alpn.AlpnProtocols = []string{"http/1.1"}
			break
		}
	}

	// HelloCustom tells utls to use our spec verbatim instead of a preset.
	uConn := utls.UClient(rawConn, &utls.Config{
		ServerName: host,
	}, utls.HelloCustom)

	if err := uConn.ApplyPreset(&spec); err != nil {
		uConn.Close()
		return nil, fmt.Errorf("apply preset: %w", err)
	}

	if err := uConn.HandshakeContext(ctx); err != nil {
		uConn.Close()
		return nil, fmt.Errorf("utls handshake: %w", err)
	}

	return uConn, nil
}

// Get performs a GET request with Chrome headers.
func (c *Client) Get(ctx context.Context, url string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("User-Agent", chromeUA)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	return resp.StatusCode, body, err
}

// PostForm sends a POST with form-encoded body to the given URL. It sets the
// Content-Type to application/x-www-form-urlencoded and uses a Chrome User-Agent.
func (c *Client) PostForm(ctx context.Context, url, formBody string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(formBody))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("User-Agent", chromeUA)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Origin", "https://www.google.com")
	req.Header.Set("Referer", "https://www.google.com/travel/flights")

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	return resp.StatusCode, body, err
}

// SearchFlights posts an encoded flight search payload to the Flights endpoint
// and returns the raw response body.
func (c *Client) SearchFlights(ctx context.Context, encodedFilters string) (int, []byte, error) {
	return c.PostForm(ctx, FlightsURL, "f.req="+encodedFilters)
}

// BatchExecute posts an encoded batchexecute payload to the Hotels/Travel endpoint
// and returns the raw response body.
func (c *Client) BatchExecute(ctx context.Context, encodedPayload string) (int, []byte, error) {
	return c.PostForm(ctx, HotelsURL, "f.req="+encodedPayload)
}

// ErrBlocked is returned when Google responds with 403 Forbidden.
var ErrBlocked = errors.New("request blocked (403)")
