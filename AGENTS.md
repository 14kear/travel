# trvl — Complete Setup for AI Assistants

> **Human**: Give this URL to your AI assistant and say "set up trvl":
> `https://raw.githubusercontent.com/MikkoParkkola/trvl/main/AGENTS.md`

## What To Do

You are setting up `trvl` — a travel MCP server that gives you real-time Google Flights and Google Hotels data. No API keys needed. Run these steps in order:

### Step 1: Install

```bash
brew install MikkoParkkola/tap/trvl
```

Pre-built binary, no toolchain needed, 1 second. If Homebrew is not available:

```bash
# Manual: detect OS/arch, download, extract
curl -fsSL https://github.com/MikkoParkkola/trvl/releases/latest/download/trvl_$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz | tar xz -C /usr/local/bin trvl
```

### Step 2: Register as MCP server

```bash
trvl mcp install                       # Claude Desktop (default)
trvl mcp install --client cursor       # Cursor / Windsurf
trvl mcp install --client claude-code  # Claude Code
```

Then restart your MCP client. No JSON editing needed.

For Claude Code, you can also run:
```bash
claude mcp add trvl --transport stdio -- trvl mcp
```

### Step 3: Install the bundled skill (makes you smarter about travel searches)

```bash
mkdir -p ~/.claude/skills
for s in trvl; do
  curl -fsSL "https://raw.githubusercontent.com/MikkoParkkola/trvl/main/.claude/skills/$s.md" -o "$HOME/.claude/skills/$s.md"
done
```

This bundled skill file teaches you how to use trvl optimally whenever the user mentions flights, hotels, travel, or trips.

### Step 4: Verify

```bash
trvl version
# Expected: trvl 0.3.0 (or later)

trvl flights HEL LHR 2026-07-01 --format json | head -5
# Expected: JSON with flight results
```

Tell the user: "trvl is installed with 32 MCP tools and 1 bundled Claude skill. I can search flights, hotels, destinations, plan trips, find weekend getaways, find optimal travel windows, optimize multi-city routes, find nearby restaurants, check local events, search ground transport, detect travel hacks, check weather forecasts, and look up airline baggage rules. Just ask me anything about travel."

### Step 5: (Optional) Set up free API keys for enhanced data

trvl works out of the box with Wikivoyage + OpenStreetMap (no keys needed). For richer data (events, restaurant ratings, attractions), the user can get free API keys:

| Service | What it adds | Signup |
|---------|-------------|--------|
| Ticketmaster | Events (concerts, sports, festivals) | https://developer.ticketmaster.com/ |
| Foursquare | Restaurant ratings, tips, price levels | https://developer.foursquare.com/ |
| Geoapify | Walking-distance POI search | https://myprojects.geoapify.com/ |
| OpenTripMap | Tourist attractions + Wikipedia | https://opentripmap.io/product |

All free, no credit card, 2 min signup each. Walk the user through each signup:
1. Open the URL for them
2. Tell them what to click (Sign up → Create project → Copy key)
3. Have them paste the key
4. Set it: `echo 'export TICKETMASTER_API_KEY="their-key"' >> ~/.zshrc && source ~/.zshrc`
5. Verify: `trvl events "Barcelona" --from 2026-07-01 --to 2026-07-08`

Use `/setup-api-keys` command for the guided wizard.

### Step 6: Build the traveller profile

This is the most important step. A good profile means every search is
personalized from the first query. The profile lives at
`~/.trvl/preferences.json` and drives real filtering: hotels get filtered
by stars, ratings, and neighborhood; hostels and airport hotels get excluded;
flights get sorted by loyalty airlines.

**How to interview: open-ended, not a form.**

Start with ONE question:

> "Tell me about yourself as a traveller — where do you live, what kind of
> trips do you usually take, and what matters most when you're booking?"

Then listen. Their answer tells you which follow-ups matter. Examples:

- They mention **work travel** → ask about loyalty programs, cabin class, expenses
- They mention **digital nomad** → ask about wifi, co-working, long-stay discounts
- They mention **family** → ask about family members, checked bags, accessibility
- They mention **budget** → skip luxury hotel questions, ask about hostels
- They mention **specific cities** → ask about preferred neighborhoods
- They mention **quality** → ask about minimum stars, review thresholds

Ask follow-ups **2-3 at a time**, never more. Each round should feel like
a natural conversation, not a questionnaire. Adapt based on what they said.

After 3-4 rounds you should have enough to fill most of these fields:

| Field | What it controls |
|-------|-----------------|
| `home_airports` | Default origin for searches (e.g. `["HEL", "AMS"]`) |
| `home_cities` | Cities to exclude from destination suggestions |
| `carry_on_only` | Enables hidden-city and throwaway-ticketing hacks |
| `prefer_direct` | Prioritizes nonstop flights |
| `no_dormitories` | Removes hostels, capsule hotels, shared rooms |
| `ensuite_only` | Requires private bathroom |
| `fast_wifi_needed` | Flags for co-working / remote work properties |
| `min_hotel_stars` | 0=any, 3=no motels, 4=business-grade |
| `min_hotel_rating` | e.g. 4.0 — also activates 20-review minimum |
| `preferred_districts` | Per-city neighborhoods (e.g. `{"Prague": ["Prague 1"]}`) |
| `display_currency` | Price display (EUR, USD, GBP, etc.) |
| `locale` | Language/region for formatting |
| `loyalty_airlines` | IATA codes (e.g. `["AY", "KL"]`) |
| `loyalty_hotels` | e.g. `["Marriott Bonvoy"]` |
| `family_members` | People the user books for, with notes |

Save to `~/.trvl/preferences.json` using the `update_preferences` tool,
or write the file directly. Show the user what you saved and ask if
anything needs adjusting.

**Keeping the profile current — iterative learning:**

The profile is never "done". Update it as you learn from conversations:

1. **Observe patterns**: User searches 4-star hotels 3 times → ask:
   "You keep picking 4-star properties — want me to set that as your
   minimum so I filter automatically?"

2. **Catch life changes**: "You've been searching from AMS a lot lately.
   Should I add Amsterdam to your home airports?"

3. **Explicit corrections**: User says "I got SkyTeam Elite Plus" → update
   loyalty immediately, confirm what changed.

4. **New cities**: User explores a city for the first time, then picks
   hotels in a specific area → ask: "Want me to remember [neighborhood]
   as your preferred area in [city] for next time?"

5. **Family updates**: "I see you're booking for someone new — want me
   to add them to your profile?"

Always confirm before updating. Use the `update_preferences` MCP tool
to write changes — it merges fields, so you only send what changed.

Or use the interactive CLI wizard: `trvl prefs init`

---

## How To Use (after setup)

You now have 32 MCP tools available. Use them when the user asks about travel:

### search_flights — Find flights between airports
```json
{"origin": "HEL", "destination": "NRT", "departure_date": "2026-06-15"}
```
Optional parameters:
- `return_date`: "2026-06-22" (makes it round-trip)
- `cabin_class`: "economy" | "premium_economy" | "business" | "first"
- `max_stops`: "any" | "nonstop" | "one_stop" | "two_plus"
- `sort_by`: "cheapest" | "duration" | "departure" | "arrival"

### search_dates — Find the cheapest day to fly
```json
{"origin": "HEL", "destination": "NRT", "start_date": "2026-06-01", "end_date": "2026-06-30"}
```
Optional: `trip_duration` (days), `is_round_trip` (true/false)

### search_hotels — Find hotels in any city
```json
{"location": "Tokyo", "check_in": "2026-06-15", "check_out": "2026-06-18"}
```
Optional: `guests` (number), `stars` (1-5 minimum), `sort` ("price" | "rating"), `currency` ("EUR" | "USD" etc.)

### hotel_prices — Compare prices across booking sites
```json
{"hotel_id": "<from search_hotels>", "check_in": "2026-06-15", "check_out": "2026-06-18"}
```

### destination_info — Travel intelligence for any city
```json
{"location": "Tokyo"}
```
Optional: `travel_dates` ("2026-06-15,2026-06-18" — comma-separated check-in,check-out)

Returns: weather forecast, country info (capital, languages, currencies), public holidays during travel dates, safety advisory (1-5 scale), currency exchange rates vs EUR, timezone.

### calculate_trip_cost — Estimate total trip cost
```json
{"origin": "HEL", "destination": "BCN", "depart_date": "2026-07-01", "return_date": "2026-07-08"}
```
Optional: `guests` (number, default 1), `currency` ("EUR" | "USD" etc.)

Returns: cheapest outbound flight + return flight + cheapest hotel per night, total cost, per-person cost, per-day cost.

### weekend_getaway — Find cheap weekend destinations
```json
{"origin": "HEL", "month": "july-2026"}
```
Optional: `max_budget` (number in EUR, 0 = no limit), `nights` (default: 2)

Returns: top 10 cheapest weekend destinations ranked by total estimated cost (round-trip flight + estimated hotel).

### suggest_dates — Smart date suggestions around a target date
```json
{"origin": "HEL", "destination": "BCN", "target_date": "2026-07-15"}
```
Optional: `flex_days` (default: 7), `round_trip` (boolean), `duration` (days for round-trip, default: 7)

Returns: 3 cheapest dates, weekday vs weekend analysis, savings insights, average price comparison.

### optimize_multi_city — Find cheapest routing for multi-city trips
```json
{"home_airport": "HEL", "cities": "BCN,ROM,PAR", "depart_date": "2026-07-01"}
```
Optional: `return_date` ("2026-07-21")

Returns: optimal visit order, per-segment prices, total cost, savings vs worst order. Tries all permutations (up to 6 cities).

### MCP Prompts (for complex workflows)
- `plan-trip` — Full trip planning: flights + hotels + budget analysis
- `find-cheapest-dates` — Month-wide price calendar for a route
- `compare-hotels` — Side-by-side hotel comparison by user priorities

## Response Tips

- Results include `booking_url` — share these with the user for direct Google links
- Results include `suggestions` — use these to offer follow-up searches
- Prices reflect the user's IP geolocation currency
- For trip planning: search flights first, then hotels at the destination
- For budget trips: use `weekend_getaway` or `suggest_dates` to find the cheapest options
- For multi-city: use `optimize_multi_city` to find the cheapest routing order
- For full cost estimates: use `calculate_trip_cost` for flights + hotel totals
- For destination research: use `destination_info` for weather, safety, holidays
- Common IATA codes: HEL (Helsinki), JFK (New York), LHR (London), NRT (Tokyo), CDG (Paris), BCN (Barcelona), BKK (Bangkok), SIN (Singapore), DXB (Dubai), LAX (Los Angeles), FRA (Frankfurt), AMS (Amsterdam), ICN (Seoul)

## Troubleshooting

- **"command not found"**: `which trvl` — if empty, the binary isn't in PATH. Re-run Step 1.
- **No results**: Google may rate-limit. Wait 60 seconds and retry.
- **Wrong currency**: Normal — currency follows IP geolocation.
- **MCP tools not showing**: Restart Claude Code / Claude Desktop after Step 2.

## Source

- GitHub: https://github.com/MikkoParkkola/trvl
- License: PolyForm Noncommercial 1.0.0
- Inspired by [fli](https://github.com/punitarani/fli) by Punit Arani
