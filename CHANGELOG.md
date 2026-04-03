# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-04-02

### Added
- **Explore destinations** — discover cheapest flights from any airport (`trvl explore HEL`)
- **CalendarGraph** — visual price grid across departure and return date ranges (`trvl grid`)
- **Destination intelligence** — weather, safety, holidays, currency, and country info from 6 free APIs (`destination_info` tool)
- **Trip cost calculator** — estimate total cost including flights and hotel (`calculate_trip_cost` tool)
- **Multi-city optimizer** — find cheapest routing order for up to 6 cities (`optimize_multi_city` tool)
- **Weekend getaway finder** — cheapest weekend destinations ranked by total cost (`weekend_getaway` tool)
- **Smart date suggestions** — analyze prices around a target date with savings insights (`suggest_dates` tool)
- **Hotel reviews** — guest review summaries and scores (`hotel_reviews` tool)
- **Nearby places** — points of interest from OpenStreetMap (`nearby_places` tool)
- **Travel guide** — local tips and practical info (`travel_guide` tool)
- **Local events** — upcoming events at destination (`local_events` tool)
- MCP structured content with content annotations (`audience`, `priority`)
- MCP elicitation for interactive parameter collection
- MCP output schemas with full JSON Schema validation for all tools
- MCP prompts: `plan-trip`, `find-cheapest-dates`, `compare-hotels`
- MCP resources: airport codes, flight/hotel usage guides, session summary
- Progressive disclosure with follow-up suggestions in every response
- Travel profile support for personalized recommendations
- 4 Claude Code skills: trvl, travel-hacks, travel-agent, travel-agent-compact
- Booking links to Google Flights and Google Hotels in results
- Docker support (`docker run ghcr.io/mikkoparkkola/trvl`)

### Changed
- Expanded from 4 to 13 MCP tools
- Upgraded MCP protocol to v2025-11-25

## [0.1.0] - 2026-03-15

### Added
- **Flight search** — real-time Google Flights data via batchexecute protocol (`search_flights` tool)
- **Date search** — cheapest flight prices across a date range (`search_dates` tool)
- **Hotel search** — Google Hotels with ratings, prices, and amenities (`search_hotels` tool)
- **Hotel prices** — compare prices across booking providers (`hotel_prices` tool)
- Chrome TLS fingerprint via utls for reliable access
- MCP server with stdio transport (4 tools)
- CLI with table and JSON output formats
- Rate limiting with token bucket and exponential backoff
- Single static binary, zero runtime dependencies
- MIT license

[0.2.0]: https://github.com/MikkoParkkola/trvl/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/MikkoParkkola/trvl/releases/tag/v0.1.0
