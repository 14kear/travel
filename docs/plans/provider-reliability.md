# Provider Reliability Improvements

All original items completed 2026-04-15. Follow-up items from the /swarm session:

## Completed in session 1

### 1. Booking.com: Elicitation-based WAF flow ✅
MCP `ElicitConfirmFunc` prompts user before opening browser. 30s deadline after confirm.

### 2. Auto-healing: provider_status in search responses ✅
`HotelSearchResult.ProviderStatuses` with `FixHint` per failure pattern.

### 3. Merge pipeline: external results not visible ✅
`HasExternalProviderSource()` bypasses quality filters for external results.

## Completed in session 2 (swarm /all)

### 4. Airbnb v2 REST API ✅
Switched from v3 GraphQL (stripped data) to v2 REST (`/api/v2/explore_tabs`).
All fields populated: name, lat/lng, ratings, reviews, address. `listing.id_str`
used to avoid float64 precision loss on int64 IDs.

### 5. Dynamic city IDs via CityLookup ✅
`ProviderConfig.CityLookup` maps city names to provider-specific IDs.
`resolveCityID()` supports exact + partial matching. 17 European cities
mapped for Hostelworld (Prague=19, Amsterdam=15, Brussels=35, etc.).

### 6. jsonPath mid-path array traversal ✅
Previous implementation returned early when hitting arrays mid-path.
Now traverses array elements, prefers non-empty values, continues with
remaining path segments. Enables Airbnb's `explore_tabs.sections.listings`
where section[0] is always empty "inserts" and section[1] has real data.

### 7. Booking.com CSRF regex fix ✅
Pattern updated from `"b_csrf_token":"..."` to `b_csrf_token:\s*'([^']+)'`
matching Booking.com's current JS object-literal format.

## Known limitations

### ~~Airbnb v2 initial test shows 0 results~~ RESOLVED (session 5)
SSR extraction via Niobe cache unwrapper now operational.

### ~~Hostelworld CityLookup needs server reload~~ RESOLVED (session 5)
Hot-reload (item 11) picks up changes. 82 cities mapped.

### Booking.com WAF workaround established (was: still blocked)
**What was fixed this session**:
- destId hardcoding removed from body_template — now uses `${city_id}` variable
- Booking CityLookup populated with 9 verified destIds: Amsterdam=-2140479,
  Barcelona=-372490, Berlin=-1746443, London=-2601889, Lisbon=-2167973,
  Madrid=-390625, Paris=-1456928, Prague=-553173, Rome=-126693
- CSRF regex updated: `b_csrf_token:\s*'([^']+)'`
- Elicitation-based WAF recovery wired end-to-end

**Session 5 update — now working**: Accept-Encoding fix, response
decompression (gzip/br/zstd), deterministic header ordering (HeaderOrder),
X-Personal-Use header skip for browser providers, CLI provider wiring.
Booking searches return results via SSR Apollo cache extraction (item 13).
Requires a Brave tab open on Booking.com for fresh aws-waf-token cookie.

**Sobek JS solver limitation**: The sobek WAF solver can execute
challenge.js but cannot produce a valid fingerprint — AWS performs canvas,
WebGL, and AudioContext fingerprinting that requires a real browser.
Browser tab workaround is the practical solution.

**Affiliate API remains recommended** for eliminating WAF/ToS risk:
Register at `developers.booking.com`. Free, authorized, Genius pricing
preserved. Same provider runtime framework applies.

## Legal review of elicitation flow (summary)

**Verdict**: Not theater — provides genuine legal risk reduction.

| Dimension | Risk | Why |
|-----------|------|-----|
| Criminal (FI Rikoslaki 38:8) | LOW | Human-solved CAPTCHA = security as designed. Explicit consent defeats mens rea. |
| ToS breach (civil) | HIGH tech / LOW practical | All 3 platforms prohibit automation. Enforcement targets commercial scrapers. |
| Cookie reading (kooky) | VERY LOW | Reading own browser cookies on own machine. |
| 24h cookie persistence | LOW | Aligns with typical session TTLs. |
| GDPR | N/A | Household exemption for personal tool, own data. |

## ToS risk reduction recommendations

**Tier 1 (eliminate)**: Register for Booking.com Affiliate API — free,
authorized access with Genius pricing. Single highest-value move.

**Tier 2 (minimize enforcement)**:
- Rate limit at 0.5 req/s (already in place)
- Don't cache/redistribute results (already the case)
- Consider `robots.txt` check on first request per provider
- Keep UA spoofing or switch to honest UA (trade-off with WAF blocks)

**Tier 3 (mitigate if caught)**:
- PolyForm Noncommercial license documented (already in place)
- Don't publish provider configs publicly (`~/.trvl/providers/*.json` are
  per-user, not in git — keep it that way)
- Elicitation flow preserved (computer-access-law defense)

Full legal analysis: see `/tmp` session transcript archived to
`~/.claude/data/legal-review-2026-04-15.md` (if preserved).

## Completed in session 3 (API-first Booking, no browser per search)

### 8. Two-stage preflight extractions ✅
`Extraction.URL` field added to `config.go`. `applyURLExtractions()` fetches
that URL with the jar's cookies and substitutes `${var}` placeholders from
prior extractions, enabling "HTML → bundle-URL extracted → JS fetched →
sha256Hash extracted" chains. Variables extracted in stage 2 are visible to
subsequent URL substitutions in the same pass (N-stage chains).

### 9. Extraction default-value fallback ✅
`Extraction.Default` field added. When a pattern does not match, the variable
is populated with the default value (used for last-known persisted-query
hashes). Prevents the search body from being transmitted with a literal
`${sha256_hash}` placeholder when an upstream HTML layout changes.

### 10. GraphQL error surfacing ✅
Both `searchProvider` and `TestProvider` now detect a top-level `errors[]`
array in the JSON response and return the first error message (with
extensions.code) instead of the generic "results_path did not resolve to an
array". Turns 28+ mystery failures into one-line diagnostics.

### 11. Registry hot-reload ✅
`Registry.Reload(id)` re-parses a single provider JSON on demand.
`Registry.ReloadIfChanged(id)` uses file mtime so each production search
picks up on-disk edits without restarting the MCP server, while the common
no-change path is a single os.Stat. `test_provider` MCP tool reloads
unconditionally so config iterations are visible in the very next call.
`searchProvider` also drops the cached `providerClient` when the config
changes so the rate limiter, TLS fingerprint and auth cache are rebuilt.

### 12. Booking config migrated to dynamic sha256Hash ✅
`body_template` uses `${sha256_hash}` instead of the hardcoded value.
New extraction `sha256_hash_inline` tries to pick a fresh hash out of the
preflight HTML (regex `FullSearch...sha256Hash...[a-f0-9]{64}`); if the
regex misses, `Default` provides the known-working hash as a fallback so
the request never ships with an unresolved placeholder.

## Completed in sessions 3–4 (SSR extraction, filters, dedup)

### 13. Booking.com SSR Apollo cache extraction ✅
Abandoned GraphQL persisted-query approach. New strategy: GET the
`/searchresults.html` page, regex-extract the Apollo JSON blob from the
embedded `<script>` tag, denormalize `__ref` pointers across the cache,
and map the flattened results to `HotelResult`. No API keys, no CSRF,
no persisted-query hashes — just server-rendered HTML.

### 14. Airbnb v2 query params with array expansion ✅
Amenity filter IDs are encoded as repeated `amenities[]=4&amenities[]=7`
query parameters. Array expansion logic added to the provider runtime
so `amenities[]` in the config expands correctly instead of producing
a single `amenities[]=[4,7]` literal.

### 15. Hostelworld expanded to 20+ European cities ✅
CityLookup grown from the original 17 to 20+ entries covering major
European destinations. City IDs verified against Hostelworld's search
endpoint.

### 16. Nine filter types wired ✅
`property_type`, `min_price`, `max_price`, `amenities`, `sort`, `stars`,
`min_rating`, `free_cancellation`, `bedrooms`, `bathrooms` — all mapped
from the generic `HotelSearchParams` filter schema to provider-specific
query parameters or body fields.

### 17. FilterComposite for Booking's nflt parameter ✅
Booking.com encodes multiple filters into a single `nflt` compound query
parameter (semicolon-delimited key=value pairs). `FilterComposite` system
assembles individual filter selections into the correct `nflt` string at
search time.

### 18. Geo-proximity dedup + brand suffix stripping ✅
Cross-provider merge uses 150m haversine distance to detect duplicate
properties across Google, Booking, Airbnb, and Hostelworld. Brand suffix
stripping normalizes 14 hotel name patterns (e.g. "Hotel X by Hilton" →
"Hotel X") before fuzzy matching to improve dedup recall.

### 19. Currency normalization ✅
Hardcoded FX rates applied during result merging so all prices display
in the user's preferred currency. Rates are static (not live) — adequate
for comparison ranking, not for booking decisions.

### 20. Cookie jar preservation across config reloads ✅
`Registry.Reload()` and `Registry.ReloadIfChanged()` now preserve the
existing HTTP cookie jar when rebuilding the `providerClient`. Previously
a config reload discarded all cookies, forcing re-authentication with
WAF-protected providers.

### 21. stripUnresolvedPlaceholders ✅
URLs and request bodies are post-processed to remove any `${var_name}`
placeholders that were not resolved during extraction. Prevents literal
placeholder strings from reaching upstream APIs and triggering 400 errors.

### 22. Airbnb array param expansion fix ✅
Separate from item 14: the URL builder now correctly expands
`amenities[]=4&amenities[]=7` rather than encoding the array as a
single JSON value. This fixed 0-result searches when amenity filters
were active.

## Known limitations (updated sessions 3–4)

### Booking ratings blocked by Akamai bot classification
Booking.com's review/rating endpoints classify automated requests via
Akamai's `b_bot` signal derived from HTTP/2 SETTINGS frame fingerprinting.
Ratings are currently sourced from Google's hotel card as a dedup
workaround. A proper fix requires either the Booking Affiliate API or
HTTP/2 SETTINGS frame mimicry.
**Session 5 update**: Confirmed ratings are still 0 in Booking SSR;
dedup merge pipeline sources ratings from Google hotel cards instead.

### Booking per-city WAF cookies
Switching search cities on Booking may require a fresh WAF preflight.
The cookie jar preserves cookies across config reloads (item 20), but
city switches within a single session can still trigger a new challenge
if Booking's WAF associates the cookie with a specific `dest_id`.
**Session 5 update**: aws-waf-token now persists in Brave cookie DB and
auto-refreshes with an open Booking tab. Sobek JS solver evaluated but
cannot fake canvas/WebGL/AudioContext fingerprints.

### Hostelworld Helsinki city ID may be incorrect
Helsinki searches return 0 results. The mapped city ID may not match
Hostelworld's current taxonomy. Needs verification against their
autocomplete endpoint.

### ~~Airbnb v2 initial test shows 0 results~~ RESOLVED (session 5)
Airbnb now uses SSR extraction via Niobe cache unwrapper and
deferred-state-0 script tag. Gzip decompression fallback handles
Content-Encoding mismatches. All fields populated.

### ~~Hostelworld CityLookup needs server reload~~ RESOLVED (session 5)
Hostelworld expanded to 82 cities (25 new + 12 aliases). Hot-reload
(item 11) picks up changes without server restart.

## Completed in session 5 (Streamable HTTP, decompression, SSR, dedup)

### 23. Trivago: Streamable HTTP MCP protocol migration ✅
Trivago provider was returning 404. Migrated from legacy SSE transport to
Streamable HTTP MCP protocol. Provider now fully operational.

### 24. Booking: Accept-Encoding + response decompression ✅
Booking responses were failing silently due to compressed payloads.
Added Accept-Encoding header support, response decompression for gzip,
Brotli (br), and zstd. Deterministic header ordering via `HeaderOrder`
config field. `X-Personal-Use` header skip for browser-based providers.
CLI provider wiring completed.

### 25. Airbnb SSR extraction ✅
Implemented Niobe cache unwrapper for Airbnb's server-rendered pages.
Extracts data from `deferred-state-0` script tag. Updated field mappings
to match current Airbnb SSR payload structure.

### 26. Airbnb gzip decompression fallback ✅
Graceful decompression fallback when server Content-Encoding header
mismatches actual encoding. Prevents failures on Airbnb responses that
claim gzip but send uncompressed (or vice versa).

### 27. Source deduplication in MergeHotelResults ✅
Cross-provider merge now deduplicates by source, preventing the same
hotel from appearing multiple times when multiple providers return it.

### 28. test_provider search path alignment ✅
Five fixes to align `test_provider` MCP tool with the actual provider
search path, ensuring test results match production behavior.

### 29. Hostelworld expanded to 82 cities ✅
Added 25 new city IDs and 12 city name aliases. Total coverage now 82
European cities, up from the previous ~20.

### 30. AWS WAF JS solver investigation ✅
Evaluated sobek-based JS solver for AWS WAF challenge.js execution.
The solver can run the JS but cannot produce a valid fingerprint —
AWS challenge.js performs canvas, WebGL, and AudioContext fingerprinting
that requires a real browser environment. Browser tab workaround is the
viable path.

### 31. Cookie freshness: aws-waf-token persistence ✅
Verified that `aws-waf-token` cookie persists in Brave's cookie DB and
auto-refreshes as long as a Booking tab remains open. Cookie jar
integration reads fresh tokens automatically.

## Provider status (as of session 5)

| Provider    | Status  | Notes |
|-------------|---------|-------|
| Google      | Working | Direct scraping, no auth needed |
| Booking     | Working | Requires Brave tab open for WAF cookie refresh |
| Airbnb      | Working | SSR extraction via Niobe/deferred-state-0 |
| Hostelworld | Working | 82 cities mapped |
| Trivago     | Working | Streamable HTTP MCP protocol |

## Known limitations (updated session 5)

### Booking requires Brave tab open for fresh aws-waf-token
The aws-waf-token cookie auto-refreshes when a Booking.com tab is open
in Brave. Without an open tab, the token expires and searches fail with
a WAF challenge response. This is the practical workaround since the
sobek JS solver cannot fake browser fingerprints.

### WAF JS solver (sobek) cannot fake browser fingerprint
AWS challenge.js performs canvas, WebGL, and AudioContext fingerprinting.
The sobek JS runtime lacks these browser APIs, so it cannot produce a
valid aws-waf-token. A headless browser (Playwright/Puppeteer) could
work but adds significant complexity and dependency weight.

### Booking ratings still 0 in SSR
Booking's server-rendered search results do not include ratings.
The dedup merge pipeline sources ratings from Google's hotel card data
instead. This is adequate for comparison but means Booking-only results
(not matched to Google) show rating 0.

### Hostelworld Helsinki city ID may be incorrect
Helsinki searches return 0 results. The mapped city ID may not match
Hostelworld's current taxonomy. Needs verification against their
autocomplete endpoint.

## Next actions

1. Verify Hostelworld Helsinki city ID against their autocomplete API
2. Consider live FX rates (Open Exchange Rates free tier) to replace
   hardcoded currency conversion
3. Booking.com Affiliate API registration remains the highest-ROI single
   action for eliminating WAF/ToS risk entirely
4. Evaluate Playwright-based WAF solver as alternative to sobek for
   environments where a Brave tab workaround is impractical
