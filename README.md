# trvl

Search flights and hotels from your terminal. Free, no API keys, real-time data from Google.

```
trvl flights HEL NRT 2026-06-15
trvl hotels "Tokyo" --checkin 2026-06-15 --checkout 2026-06-18
```

## What is this?

`trvl` is a single Go binary that provides programmatic access to Google Flights and Google Hotels data through reverse engineering of Google's internal APIs. No scraping, no browser automation, no API keys.

**Inspired by [fli](https://github.com/punitarani/fli)** by Punit Arani, which pioneered this approach for Google Flights in Python. `trvl` extends it to hotels and reimplements both in Go for single-binary distribution.

## Install

```bash
# From source
go install github.com/MikkoParkkola/trvl/cmd/trvl@latest

# Or build locally
git clone https://github.com/MikkoParkkola/trvl.git
cd trvl
make build
# Binary at ./bin/trvl
```

## Usage

### Flight Search

```bash
# One-way
trvl flights HEL NRT 2026-06-15

# Round-trip
trvl flights HEL BCN 2026-07-01 --return 2026-07-08

# Business class, nonstop only
trvl flights JFK LHR 2026-07-01 --cabin business --stops nonstop

# JSON output (for scripts/pipelines)
trvl flights HEL NRT 2026-06-15 --format json
```

### Find Cheapest Dates

```bash
# Cheapest day to fly in June
trvl dates HEL NRT --from 2026-06-01 --to 2026-06-30

# Round-trip, 7-day trips
trvl dates HEL BCN --from 2026-07-01 --to 2026-08-31 --duration 7 --round-trip
```

### Hotel Search

```bash
# Search hotels
trvl hotels "Helsinki" --checkin 2026-06-15 --checkout 2026-06-18

# Filter by stars, sort by rating
trvl hotels "Tokyo" --checkin 2026-06-15 --checkout 2026-06-18 --stars 4 --sort rating

# JSON output
trvl hotels "Paris" --checkin 2026-07-01 --checkout 2026-07-05 --format json
```

### Hotel Price Comparison

```bash
# Get prices from multiple booking providers
trvl prices "<hotel_id>" --checkin 2026-06-15 --checkout 2026-06-18
```

The `hotel_id` comes from `trvl hotels` search results.

### MCP Server (AI Agent Integration)

```bash
# stdio mode (for Claude Code, Cursor, etc.)
trvl mcp

# HTTP mode (for gateway/remote access)
trvl mcp --http --port 8000
```

Implements [MCP 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25/) with:
- 4 tools: `search_flights`, `search_dates`, `search_hotels`, `hotel_prices`
- 3 prompts: `plan-trip`, `find-cheapest-dates`, `compare-hotels`
- 3 resources: airport codes, flight help, hotel help
- Elicitation support for interactive search refinement
- Structured content with audience annotations
- Progressive disclosure suggestions

#### Claude Code Integration

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

#### mcp-gateway Integration

```yaml
backends:
  trvl:
    transport: stdio
    command: trvl mcp
    description: "Travel search - flights and hotels via Google"
```

## How It Works

Google's travel search uses an internal gRPC-over-HTTP protocol called "batchexecute". `trvl` speaks this protocol directly:

1. **Chrome TLS fingerprint** via [utls](https://github.com/refraction-networking/utls) to pass Google's bot detection
2. **Flight search** via `FlightsFrontendService/GetShoppingResults` endpoint
3. **Hotel search** via `TravelFrontendUi` page parsing with embedded `AF_initDataCallback` data
4. **Hotel prices** via `TravelFrontendUi/data/batchexecute` with rpcid `yY52ce`
5. **Rate limiting** at 10 req/s with exponential backoff retry on 429/5xx

## Features

- **Zero dependencies** at runtime. Single static binary, no API keys, no config files
- **Real-time data** from Google Flights and Google Hotels
- **Flights**: one-way, round-trip, cabin class, stop filter, airline filter, sort options
- **Hotels**: search by city/location, star rating filter, price sorting, geocoding via Nominatim
- **Cheapest dates**: find the best day to fly across a date range (concurrent search, 3x faster)
- **MCP server**: full MCP 2025-11-25 protocol for AI agent integration
- **JSON output**: `--format json` on all commands for scripting
- **Table output**: aligned columns for terminal display (default)
- **14MB binary**: cross-compiles for Linux/macOS, AMD64/ARM64

## Attribution

This project builds on the work of:
- [fli](https://github.com/punitarani/fli) by Punit Arani — Google Flights Python library, primary inspiration
- [utls](https://github.com/refraction-networking/utls) — Go TLS fingerprint impersonation
- [icecreamsoft](https://icecreamsoft.hashnode.dev/building-a-web-app-for-travel-search) — Google Hotels batchexecute documentation
- [SerpAPI docs](https://serpapi.com/google-hotels-api) — Hotel parameter documentation

## Legal

This tool accesses Google's public-facing internal APIs for personal use. It does not bypass authentication, scrape protected content, or violate rate limits. Same approach as [fli](https://github.com/punitarani/fli) (1K+ GitHub stars, actively maintained).

Use responsibly. Respect Google's rate limits.

## License

MIT
