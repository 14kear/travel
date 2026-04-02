# trvl — Google Flights + Hotels from your terminal

> **Free. No API keys. Real-time prices. One binary.**

```bash
$ trvl flights HEL NRT 2026-06-15

Found 86 flights (one_way)

| Price    | Duration | Stops   | Route                    | Airline               | Departs          |
+----------+----------+---------+--------------------------+-----------------------+------------------+
| EUR 603  | 24h 20m  | 2 stops | HEL -> CPH -> AUH -> NRT | Scandinavian Airlines | 2026-06-15T06:10 |
| EUR 656  | 24h 10m  | 2 stops | HEL -> CPH -> AUH -> NRT | Finnair               | 2026-06-15T06:20 |
| EUR 875  | 31h 20m  | 1 stop  | HEL -> IST -> NRT        | Turkish Airlines      | 2026-06-15T19:35 |
```

```bash
$ trvl hotels "Tokyo" --checkin 2026-06-15 --checkout 2026-06-18

Found 20 hotels:

| Name                              | Stars | Rating | Reviews | Price   |
+-----------------------------------+-------+--------+---------+---------+
| HOTEL MYSTAYS PREMIER Omori       | 4     | 4.1    | 2059    | 150 EUR |
| Hotel JAL City Tokyo Toyosu       | 4     | 4.2    | 1080    | 150 EUR |
| Koko Hotel Tsukiji Ginza          | 4     | 3.9    | 650     | 89 EUR  |
```

## Why trvl?

Google Flights and Hotels have no public API. Your options are $50+/mo SERP proxies or fragile Selenium scrapers.

`trvl` talks directly to Google's internal protocol — the same one the website uses. No scraping, no headless browsers, no API keys, no monthly fees. Just real-time travel data in a 15MB binary.

**Inspired by [fli](https://github.com/punitarani/fli)** by Punit Arani, which pioneered this for flights in Python. `trvl` extends it to hotels and ships as a single Go binary.

## Install

### One-liner (recommended)

```bash
# macOS / Linux — download latest binary
curl -fsSL https://github.com/MikkoParkkola/trvl/releases/latest/download/trvl_$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz | tar xz -C /usr/local/bin trvl
```

### Homebrew (macOS / Linux)

```bash
brew install MikkoParkkola/tap/trvl
```

### Go install

```bash
go install github.com/MikkoParkkola/trvl/cmd/trvl@latest
```

### Download binary

Grab a prebuilt binary from [**Releases**](https://github.com/MikkoParkkola/trvl/releases) — available for Linux and macOS (amd64 + arm64).

### Build from source

```bash
git clone https://github.com/MikkoParkkola/trvl.git
cd trvl
make build      # binary at ./bin/trvl
```

### Docker

```bash
docker run --rm ghcr.io/mikkoparkkola/trvl flights HEL NRT 2026-06-15
```

### As MCP server (Claude Code / Claude Desktop)

No separate install needed — just point to the binary:

```json
{
  "mcpServers": {
    "trvl": {
      "command": "trvl",
      "args": ["mcp"]
    }
  }
}
```

### Verify installation

```bash
trvl version
# trvl v0.1.0
```

## Flights

```bash
trvl flights JFK LHR 2026-07-01                              # One-way
trvl flights HEL BCN 2026-07-01 --return 2026-07-08          # Round-trip
trvl flights JFK LHR 2026-07-01 --cabin business --stops nonstop  # Filters
trvl flights HEL NRT 2026-06-15 --format json                # JSON output
trvl flights HEL NRT 2026-06-15 --sort duration              # Sort by duration
trvl flights HEL NRT 2026-06-15 --airline AY,SK              # Filter airlines
```

**Filters**: `--cabin` (economy/premium_economy/business/first), `--stops` (any/nonstop/one_stop/two_plus), `--sort` (cheapest/duration/departure/arrival), `--airline` (comma-separated IATA codes)

## Cheapest Dates

```bash
trvl dates HEL NRT --from 2026-06-01 --to 2026-06-30                     # One-way
trvl dates HEL BCN --from 2026-07-01 --to 2026-08-31 --duration 7 --round-trip  # Round-trip
```

Searches each date in parallel (3 concurrent) and returns the cheapest price per day. Great for flexible travel planning.

## Hotels

```bash
trvl hotels "Helsinki" --checkin 2026-06-15 --checkout 2026-06-18
trvl hotels "Tokyo" --checkin 2026-06-15 --checkout 2026-06-18 --stars 4 --sort rating
trvl hotels "Paris" --checkin 2026-07-01 --checkout 2026-07-05 --format json
```

**Filters**: `--stars` (minimum 1-5), `--guests` (default 2), `--sort` (price/rating), `--currency` (USD/EUR/etc.)

## Hotel Price Comparison

```bash
trvl prices "<hotel_id>" --checkin 2026-06-15 --checkout 2026-06-18
```

Compares prices across Booking.com, Hotels.com, Expedia, and other providers. `hotel_id` comes from hotel search results.

## MCP Server — AI Agent Integration

`trvl` ships a built-in [Model Context Protocol](https://modelcontextprotocol.io/) server (v2025-11-25) for seamless AI assistant integration.

```bash
trvl mcp              # stdio (Claude Code, Cursor, Windsurf, etc.)
trvl mcp --http       # HTTP transport (gateway, remote access)
```

### Claude Code / Claude Desktop

```json
{
  "mcpServers": {
    "trvl": {
      "command": "trvl",
      "args": ["mcp"]
    }
  }
}
```

### MCP Features

| Feature | Details |
|---------|---------|
| **Tools** | `search_flights`, `search_dates`, `search_hotels`, `hotel_prices` |
| **Prompts** | `plan-trip`, `find-cheapest-dates`, `compare-hotels` |
| **Resources** | Airport codes (50 major), flight/hotel help guides |
| **Elicitation** | Interactive forms for search refinement (dates, cabin, stars) |
| **Structured content** | Typed JSON (`structuredContent`) + human summary with audience annotations |
| **Progressive disclosure** | Suggestions for follow-up searches (round-trip, nonstop, flexible dates) |
| **Output schemas** | Full JSON Schema validation for all tool responses |

### mcp-gateway

```yaml
backends:
  trvl:
    transport: stdio
    command: trvl mcp
```

## How It Works

Google's travel frontend uses an internal gRPC-over-HTTP protocol called **batchexecute**. `trvl` speaks this protocol natively:

1. **Chrome TLS fingerprint** — [utls](https://github.com/refraction-networking/utls) impersonates Chrome's exact TLS ClientHello to pass bot detection
2. **Flights** — `FlightsFrontendService/GetShoppingResults` with encoded filter arrays
3. **Hotels** — `TravelFrontendUi` embedded JSON parsing from `AF_initDataCallback` blocks
4. **Hotel prices** — `TravelFrontendUi/data/batchexecute` with rpcid `yY52ce`
5. **Rate limiting** — 10 req/s token bucket with exponential backoff (1s/2s/4s) on 429/5xx

No Selenium. No Puppeteer. No browser. Just HTTP.

## Features at a Glance

| | |
|---|---|
| **Binary** | Single static 15MB binary. Zero runtime dependencies. |
| **Data** | Real-time from Google Flights + Google Hotels |
| **Auth** | None required. No API keys, no accounts, no tokens. |
| **Output** | Pretty tables (default) or JSON (`--format json`) |
| **MCP** | Full v2025-11-25 with elicitation, structured content, prompts |
| **Platforms** | Linux, macOS (amd64, arm64) |
| **Language** | Go 1.24+, pure Go, no CGO |
| **Tests** | 325 test functions, race-detector clean |
| **License** | MIT |

## Attribution

This project stands on the shoulders of:

- **[fli](https://github.com/punitarani/fli)** by Punit Arani — the original Google Flights reverse-engineering library. `trvl`'s flight search is a direct Go reimplementation of fli's approach.
- **[utls](https://github.com/refraction-networking/utls)** by Refraction Networking — Chrome TLS fingerprint impersonation
- **[icecreamsoft](https://icecreamsoft.hashnode.dev/building-a-web-app-for-travel-search)** — Google Hotels batchexecute documentation
- **[SerpAPI](https://serpapi.com/google-hotels-api)** — Hotel parameter reference documentation

## Legal

`trvl` accesses Google's public-facing internal APIs. It does not bypass authentication, access protected content, or circumvent rate limits. This is the same approach used by [fli](https://github.com/punitarani/fli) (1K+ GitHub stars, actively maintained, MIT licensed).

Use responsibly. Respect rate limits.

## For AI Assistants

Point your AI assistant to one of these URLs for automatic setup:

- **Quick reference**: `https://raw.githubusercontent.com/MikkoParkkola/trvl/main/llms.txt`
- **Full setup guide**: `https://raw.githubusercontent.com/MikkoParkkola/trvl/main/AGENTS.md`

These files contain everything an LLM needs to install trvl, configure the MCP server, and start searching flights and hotels.

## License

[MIT](LICENSE)
