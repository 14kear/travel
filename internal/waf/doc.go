// Package waf implements a Tier 3 solver for AWS WAF JavaScript challenges.
//
// Many scraping targets (booking.com, Expedia, etc.) front their pages with
// AWS WAF, which serves an interstitial HTML page containing a <script>
// reference to challenge.js on *.awswaf.com. The real browser executes that
// script, performs a proof-of-work / environment fingerprint, and receives an
// "aws-waf-token" cookie that subsequent requests must carry.
//
// This package replays that flow headlessly in-process:
//
//  1. Parse the interstitial HTML to locate gokuProps and the challenge.js URL.
//  2. Fetch challenge.js through the caller-supplied *http.Client (which is
//     expected to carry the uTLS Chrome fingerprint + any existing session
//     cookies — this package deliberately does not mint its own client).
//  3. Spin up a fresh sobek (ES2022) runtime, install a minimal browser-like
//     DOM / window / navigator / crypto.subtle / timers surface via stubs.js
//     plus a handful of Go-side host bridges (fetch, crypto random, SHA
//     digests), evaluate challenge.js inside that surface, and drive the
//     resulting Promise to completion through a small microtask+timer loop.
//  4. Harvest the resulting token from the scripted document.cookie jar (or
//     from the JS return value as a fallback) and hand it back as an
//     *http.Cookie ready to be installed into the caller's cookie jar.
//
// The solver is intentionally scoped narrowly. It does NOT implement a full
// DOM, it does NOT provide layout or rendering, and it does NOT claim to
// defeat every flavour of WAF challenge — only the AWS WAF JS integration as
// shipped on major travel sites at the time of writing. Anything broader
// (Cloudflare Turnstile, DataDome, PerimeterX) is out of scope.
//
// Integration with the rest of trvl (providers/runtime.go, providers/cookies.go)
// happens in a separate PR; this package is self-contained and has no
// dependency on internal/providers.
package waf
