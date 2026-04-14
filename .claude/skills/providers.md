---
name: providers
description: Configure external data providers for trvl
globs:
  - "internal/providers/**"
  - "templates/**"
---

## External Provider System

trvl has built-in providers (Google Hotels, Google Flights, FlixBus, etc.) and
supports user-configured external providers via the `configure_provider` MCP tool.

### Critical: Read Before You Generate

When adding a provider, you MUST fetch and read the reference project's
source code BEFORE generating a ProviderConfig. Do not rely on your
training data for API endpoints — they change frequently.

Use `nab fetch`, web browsing, or any available tool to:
1. Read the reference project's README
2. Read the specific source file mentioned in the catalog hint (auth_hint field)
3. Extract the actual endpoint URL, auth pattern, and response schema
4. Only then generate the config

If you cannot access the reference project, ask the user to paste the
relevant source code rather than guessing. Hallucinated endpoints are the
number one cause of provider setup failure.

### PROACTIVE BEHAVIOR

When a user searches for hotels and gets results only from built-in providers,
**proactively suggest** additional providers they could enable:

> "I found 47 hotels from Google Hotels. I can also search Booking.com, Airbnb,
> Hostelworld, and VRBO for more options. Want me to set any of these up?"

When a user first uses trvl or asks about setup/preferences, suggest the
provider configuration:

> "Would you like to enable additional hotel providers? Available options
> include Booking.com, Airbnb, VRBO, Hostelworld, and others."

### How it works (Multi-Step, Verified)

1. **Discover**: Call `suggest_providers` to get the catalog. Each entry
   includes an `auth_hint` with specific file paths in the reference project.
2. **Research**: Fetch and read the reference project source files listed in
   `auth_hint`. Extract the real endpoint, auth method, and response schema.
   Do NOT skip this step.
3. **Generate**: Using verified information from step 2, generate a config.
   - body_template MUST be a JSON string, never a nested object
   - rate_limit defaults to 0.5 req/s if not specified
4. **Configure**: Call `configure_provider` — trvl asks the user directly
   for consent.
5. **Test**: Call `test_provider` to verify the config works.
6. **Iterate**: If the test fails, read the error hint carefully:
   - "no match" on extraction: your regex is wrong, check the actual page
   - "HTTP 403/202": set tls_fingerprint="chrome" and browser_escape_hatch=true
   - "0 results": results_path is wrong, check the actual response JSON
   - "HTTP 401": auth extraction failed, re-read the reference project
   Retry up to 3 times, adjusting based on the specific error.
7. Config saved to ~/.trvl/providers/ — included in future searches.

The runtime supports: HTTP GET/POST, preflight auth with POST support,
regex extraction, JSONPath field mapping, modern TLS compatibility,
per-provider rate limiting, and cookie jar.

### IMPORTANT: Consent

- ALWAYS use `configure_provider` (it triggers direct user consent via elicitation)
- NEVER bypass the consent flow
- ALWAYS inform the user about ToS restrictions before configuring

### Templates

Three generic templates in templates/ (all use example.com):
- `graphql_accommodation.yaml` — GraphQL persisted query pattern
- `rest_api.yaml` — REST JSON API pattern
- `oauth2_accommodation.yaml` — OAuth2 token exchange pattern

---

## PROVIDER CATALOG

Available providers and reference projects. The `auth_hint` field in the
`suggest_providers` output has specific file paths — read those files.

### Hotels & Accommodation

**Booking.com** — Hotels, apartments worldwide
- Reference: github.com/opentabs-dev/opentabs (MIT)
- See src/api.ts for endpoint, src/types.ts for response schema
- WAF-protected: enable browser_escape_hatch

**Airbnb** — Vacation rentals, apartments
- Reference: github.com/johnbalvin/gobnb (MIT)
- See api/search.go for endpoint, api/types.go for response schema

**VRBO** — Vacation rentals (Expedia Group)
- Reference: search GitHub for "vrbo graphql"

**Hostelworld** — Hostels, budget accommodation
- Reference: search GitHub for "hostelworld api"
- Uses numeric city IDs, REST API with results in .properties[]

### Reviews & Ratings

**TripAdvisor** — Hotel and restaurant reviews
- Reference: search GitHub for "tripadvisor graphql"

### Ground Transport

**BlaBlaCar** — Ridesharing
- Reference: search GitHub for "blablacar api"

### Restaurants

**OpenTable** — Restaurant availability
- Reference: search GitHub for "opentable api"

---

## Config Generation Guidelines

When generating a `configure_provider` config:

1. **Consult the reference project** for the target service to get current
   endpoints, auth patterns, and response paths.
   Do NOT guess or hallucinate these values — they change frequently.

2. **Use the appropriate template pattern** from the templates/ directory
   as a structural guide.

3. **Always set conservative rate limits** (0.5-2 req/s).

4. **body_template must be a JSON string**, not a nested object. If your
   LLM wants to send it as an object, stringify it first.

5. **Use `test_provider`** after configuration to verify it works.
   Read the hint in the error response carefully — it tells you what to fix.
   Iterate automatically up to 3 times if the test fails.

---

## Self-Healing

If a provider returns errors after working previously:
- **400:** API structure likely changed. Check the reference project for updates.
- **403:** TLS compatibility issue. Try enabling Chrome TLS fingerprint.
- **429:** Rate limited. The runtime handles backoff automatically.
- **Empty results:** Response structure may have changed. Check the reference
  project for updated field paths.
