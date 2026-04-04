# Eurostar API — Captured via Playwright 2026-04-04

## Endpoint
POST https://site-api.eurostar.com/gateway

## Headers (required)
- Content-Type: application/json
- Accept: */*
- x-platform: web
- x-market-code: uk
- Referer: https://www.eurostar.com/
- Accept-Language: en-GB

## cheapestFaresSearch Query

### Variables format
```json
{
  "fares": [{
    "origin": "7015400",
    "destination": "8727100", 
    "startDate": "2026-04-01",
    "endDate": "2026-05-01",
    "journeyType": "RETURN",
    "direction": "OUTBOUND"
  }],
  "currency": "GBP",
  "numberOfPassenger": 1
}
```

### For Snap fares (cheaper)
Add to variables:
```json
"productFamiliesSearch": ["PUB_STANDARD", "RED_PUB_STANDARD"]
```

### Response
```json
{"data": {"cheapestFaresSearch": [{"cheapestFares": [
  {"date": "2026-04-10", "price": 130, "__typename": "CheapestFare"},
  {"date": "2026-08-01", "price": 39, "__typename": "CheapestFare"}
]}]}}
```

## timetableServices Query (schedules)
```json
{
  "operationName": "timetableServices",
  "variables": {
    "date": "2026-04-04T13:52:19.740Z",
    "originUic": "7015400",
    "destinationUic": "8727100"
  }
}
```

## KEY FINDING
Playwright headless gets 200 OK. Our Go utls Chrome TLS gets 403.
The Playwright Chromium binary uses different TLS than utls.
Solution: Either fix utls config, or proxy through Playwright.

## Station UICs (confirmed)
- London St Pancras: 7015400
- Paris Gare du Nord: 8727100
- Brussels Midi: 8814001
- Amsterdam Centraal: 8400058
- Rotterdam Centraal: 8400530
- Lille Europe: 8722326
- Cologne Hbf: 8015458
