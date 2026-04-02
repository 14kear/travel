---
description: Find the cheapest travel deals from your location
argument-hint: "[origin airport] or leave blank for HEL"
allowed-tools:
  - Bash
  - mcp__gateway__gateway_invoke
  - mcp__gateway__gateway_search_tools
---

Find the best travel deals using trvl tools.

Origin: $ARGUMENTS (default: HEL)

1. Run explore_destinations from the origin to find cheapest destinations
2. For the top 5 cheapest: get destination info (weather, safety)
3. Check nearby airports for even cheaper options
4. Present as a ranked deal list with: destination, price, weather, what it's known for
5. Offer: "Want me to plan a full trip to any of these?"
