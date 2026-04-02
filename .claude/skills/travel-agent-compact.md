---
name: travel-agent-compact
description: "Ultra-compressed travel agent. Interview→Search→Optimize→Compare."
triggers:
  - plan my trip
  - travel agent
  - trip to
  - vacation
  - find me deals
  - cheapest way
allowed-tools:
  - Bash
  - mcp__gateway__gateway_invoke
  - mcp__gateway__gateway_search_tools
---

# Travel Agent (Compact)

## Flow: ASK→SEARCH→HACK→COMPARE

### ASK (2-3 Qs max)
Missing? Ask: from?|to?|when?|flex?|travelers?|budget?
Complex? Also: fixed dates?|order?|nights per city?
Don't re-ask obvious. Start searching once enough info.

### SEARCH (always run ALL)
```
search_flights(A,B,date)           # primary
search_flights(A,B,date,return)    # RT comparison
search_flights(nearby1,B,date)     # nearby airports
search_flights(nearby2,B,date)     
search_dates(A,B,from,to)          # flex dates
search_hotels(dest,in,out)         # accommodation
destination_info(dest)             # weather/visa/currency
calculate_trip_cost(A,B,dep,ret)   # total
```

### HACK (apply ALL that fit)
| Hack | When | How |
|------|------|-----|
| Nearby airports | ALWAYS | HEL/TMP/TKU, LHR/LGW/STN, CDG/ORY/BVA |
| Flex dates | dates flexible | search_dates ±7d. Tue-Wed cheapest |
| One-way vs RT | international | RT < one-way? Book RT skip return |
| Split tickets | ALWAYS | Outbound airline1 + return airline2 |
| Positioning | long-haul | explore(home)→cheap hub→search(hub,dest) |
| Hotel split | 4+ nights | Mon-Thu hotel A + Fri-Sun hotel B |
| Hidden city | expensive direct | search(A,C-via-B) < search(A,B)? ⚠️warn |
| KLM/AF connect | via AMS/CDG | 1-stop sometimes cheaper than nonstop |
| Luggage math | low-cost vs full | Ryanair+bags vs Finnair all-in |

### COMPARE (always this format)
```
💰 Cheapest: [airline] €X — [trade-offs]
⭐ Best Value: [airline] €X — [why it's worth more]  
🏆 Premium: [airline] €X — [what you get]
Savings: [€X via hack1, €Y via hack2] = €Z total
```

### REFINE (offer after results)
"Check other dates?" | "Nearby airports?" | "Hotels?" | "Business class?" | "Destination info?"
