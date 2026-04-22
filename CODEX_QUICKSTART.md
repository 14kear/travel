# Codex Quickstart

Use this when someone clones the repository and wants Codex to set up trvl for them.

## 1. Clone and open the repo

```bash
git clone https://github.com/14kear/travel.git
cd travel
codex
```

## 2. Ask Codex to set everything up

Paste this into Codex from the repository root:

```text
Read AGENTS.md and README.md, then set up this trvl repository end to end.

Please:
- install any required local dependencies;
- build the CLI;
- run the relevant tests;
- verify ./bin/trvl version;
- register trvl as an MCP server for Codex if it is not already registered;
- show me one working flight search and one travel-hack search.

After setup, explain the exact commands I can use for flights, hidden-city checks,
stopovers, split tickets, and flexible-date searches.
```

## 3. Example checks after setup

```bash
./bin/trvl version
./bin/trvl flights TBS TAS 2026-05-09 --adults 1 --currency USD
./bin/trvl hacks TBS TAS 2026-05-09 --currency USD
```

For business or domestic first searches in the United States:

```bash
./bin/trvl flights JFK PIT 2026-05-13 --adults 1 --cabin business --currency USD
```

## Notes for travel searches

- Hidden-city should still be searched even when the traveler has checked baggage.
- If checked baggage is involved, ask about short-checking the bag to the intermediate city.
- For flexible dates, search every possible departure date separately unless the tool has a dedicated date optimizer for the route.
- For "NYC", check JFK, LGA, and EWR.
- For "domestic business" in the United States, results are usually domestic first or premium cabin rather than international lie-flat business.
