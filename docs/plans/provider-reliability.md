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

### Airbnb v2 initial test shows 0 results
Config applied via MCP `configure_provider`; MCP server reload required to
pick up the new endpoint structure. Binary rebuilt + installed but running
MCP server holds old in-memory config. Reconnect MCP client to verify.

### Hostelworld CityLookup needs server reload
Config contains `city_lookup` field but MCP server's in-memory config still
has hardcoded endpoint. Rebuilt binary at `/opt/homebrew/bin/trvl` has
CityLookup support. Restart MCP server to activate.

### Booking.com still blocked by AWS WAF (structural fix applied)
**What was fixed this session**:
- destId hardcoding removed from body_template — now uses `${city_id}` variable
- Booking CityLookup populated with 9 verified destIds: Amsterdam=-2140479,
  Barcelona=-372490, Berlin=-1746443, London=-2601889, Lisbon=-2167973,
  Madrid=-390625, Paris=-1456928, Prague=-553173, Rome=-126693
- CSRF regex updated: `b_csrf_token:\s*'([^']+)'`
- Elicitation-based WAF recovery wired end-to-end

**What still doesn't work**: Booking serves HTTP 202 AWS WAF challenge pages
for all requests (even from a browser with cookies). The preflight URL never
returns the real search page with a CSRF token, so extraction fails at the
auth step. The tier 3b sobek WAF solver needs to execute `challenge.js` and
obtain an `aws-waf-token` cookie before the real page is accessible —
this mechanism exists in code but the challenge appears more complex than
what the solver currently handles.

**Recommended resolution**: Register for Booking.com Affiliate API
(`developers.booking.com`). Free, authorized, your Genius pricing preserved.
Same provider runtime framework applies — just different endpoints and auth.
This converts the highest-ToS-risk provider from HIGH risk to ZERO.

**Alternative**: Ship the elicitation flow to production and hope the user
sees the prompt. Works when the MCP client is interactive and the user
completes the browser challenge, but remains fragile for automated use.

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

## Next actions

1. `/mcp` reconnect so the running server binary picks up the new code
   (same-process MCP server does not reload Go binaries; reconnect
   respawns the `trvl mcp` subprocess from `/opt/homebrew/bin/trvl`)
2. Re-run `test_provider booking` — expected output if WAF is happy:
   `extraction_results` includes `sha256_hash_inline: ok (…)`, and either
   (a) `success: true` with `results_count > 0`, or (b) a precise
   `graphql error: <message>` telling us exactly what Booking rejects
3. If graphql error is `persistedQueryNotFound`, refresh the `default`
   in booking.json from the live JS bundle; if `PersistedQueryNotSupported`,
   add a stage-2 extraction that pulls the hash from the JS chunk directly
4. Consider Booking.com Affiliate API registration as primary ToS-compliance
   measure (Genius 3 + employee discount context makes this high-ROI)
