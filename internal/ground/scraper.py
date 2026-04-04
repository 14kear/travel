#!/usr/bin/env python3
"""
Browser-based train price scraper using Playwright.

Reads JSON from stdin:
  {"provider":"trainline","from":"London","to":"Paris","date":"2026-04-10","currency":"EUR"}

Writes JSON to stdout:
  {"routes":[{"price":39.00,"currency":"GBP","departure":"06:31","arrival":"09:47",
              "duration":196,"type":"train","provider":"eurostar","transfers":0}]}

On error:
  {"routes":[],"error":"reason"}
"""

import json
import sys
import re

try:
    from playwright_stealth import Stealth as _Stealth
    _STEALTH_AVAILABLE = True
except ImportError:
    _Stealth = None
    _STEALTH_AVAILABLE = False


def _apply_stealth(page):
    """Apply playwright-stealth patches to a page if available."""
    if _STEALTH_AVAILABLE and _Stealth is not None:
        try:
            _Stealth().apply_stealth_sync(page)
        except Exception:
            pass


def main():
    try:
        from playwright.sync_api import sync_playwright, TimeoutError as PWTimeoutError
    except ImportError:
        out([], "playwright not installed: pip install playwright && playwright install chromium")
        return

    raw = sys.stdin.read().strip()
    if not raw:
        out([], "no input on stdin")
        return

    try:
        inp = json.loads(raw)
    except json.JSONDecodeError as e:
        out([], f"invalid JSON input: {e}")
        return

    provider = inp.get("provider", "").lower()
    from_city = inp.get("from", "")
    to_city = inp.get("to", "")
    date = inp.get("date", "")
    currency = inp.get("currency", "EUR").upper()

    if not all([provider, from_city, to_city, date]):
        out([], "missing required fields: provider, from, to, date")
        return

    scrapers = {
        "trainline": scrape_trainline,
        "oebb": scrape_oebb,
        "sncf": scrape_sncf,
    }

    fn = scrapers.get(provider)
    if fn is None:
        out([], f"unsupported provider: {provider}")
        return

    try:
        with sync_playwright() as pw:
            browser = pw.chromium.launch(
                headless=True,
                args=[
                    "--no-sandbox",
                    "--disable-blink-features=AutomationControlled",
                    "--disable-dev-shm-usage",
                ],
            )
            context = browser.new_context(
                user_agent=(
                    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
                    "AppleWebKit/537.36 (KHTML, like Gecko) "
                    "Chrome/133.0.0.0 Safari/537.36"
                ),
                locale="en-GB",
                viewport={"width": 1280, "height": 800},
            )
            # Mask webdriver flag to reduce bot detection.
            context.add_init_script(
                "Object.defineProperty(navigator, 'webdriver', {get: () => undefined})"
            )
            page = context.new_page()
            result = fn(page, from_city, to_city, date, currency)
            browser.close()
            # Scrapers may return either a list of routes or a dict {routes, error}.
            if isinstance(result, dict):
                out(result.get("routes", []), result.get("error"))
            else:
                out(result)
    except Exception as e:
        out([], f"{provider} scraper error: {e}")


# ---------------------------------------------------------------------------
# Trainline
# ---------------------------------------------------------------------------

# Station ID map matching trainline.go
TRAINLINE_STATIONS = {
    "london": "8267",
    "paris": "4916",
    "amsterdam": "8657",
    "brussels": "5893",
    "berlin": "7527",
    "munich": "7480",
    "frankfurt": "7604",
    "hamburg": "7626",
    "cologne": "21178",
    "vienna": "22644",
    "zurich": "6401",
    "milan": "8490",
    "rome": "8544",
    "barcelona": "6617",
    "madrid": "6663",
    "prague": "17587",
    "warsaw": "10491",
    "budapest": "18819",
    "copenhagen": "17515",
    "stockholm": "38711",
    "rotterdam": "23616",
    "lille": "4652",
    "lyon": "4718",
    "marseille": "4790",
    "nice": "4836",
    "strasbourg": "153",
    "toulouse": "5306",
    "venice": "8574",
    "florence": "8434",
    "salzburg": "6994",
    "innsbruck": "10461",
    "geneva": "5335",
    "basel": "5877",
    "antwerp": "5929",
}


def scrape_trainline(page, from_city, to_city, date, currency):
    from_id = TRAINLINE_STATIONS.get(from_city.lower())
    to_id = TRAINLINE_STATIONS.get(to_city.lower())
    if not from_id or not to_id:
        raise ValueError(f"no Trainline station ID for {from_city!r} or {to_city!r}")

    _apply_stealth(page)

    # Navigate to homepage first — establishes Datadome cookies / session.
    page.goto("https://www.thetrainline.com", wait_until="networkidle", timeout=25000)
    _dismiss_cookies(page)

    # Call the journey-search API from the authenticated page context.
    result = page.evaluate(f"""
    async () => {{
        const r = await fetch('/api/journey-search/', {{
            method: 'POST',
            headers: {{
                'Content-Type': 'application/json',
                'Accept': 'application/json',
                'x-version': '4.46.32109'
            }},
            body: JSON.stringify({{
                passengers: [{{dateOfBirth: '1996-01-01', cardIds: []}}],
                isEurope: true,
                cards: [],
                transitDefinitions: [{{
                    direction: 'outward',
                    origin: 'urn:trainline:generic:loc:{from_id}',
                    destination: 'urn:trainline:generic:loc:{to_id}',
                    journeyDate: {{type: 'departAfter', time: '{date}T06:00:00'}}
                }}],
                type: 'single',
                maximumJourneys: 5,
                includeRealtime: true,
                transportModes: ['mixed'],
                directSearch: false
            }})
        }});

        if (r.status !== 200) {{
            const t = await r.text();
            return JSON.stringify({{error: 'HTTP ' + r.status + ': ' + t.substring(0, 200)}});
        }}

        const d = await r.json();
        const journeys = (d.data && d.data.journeySearch && d.data.journeySearch.journeys)
            ? Object.values(d.data.journeySearch.journeys)
            : (d.journeys || []);
        const legs = (d.data && d.data.journeySearch && d.data.journeySearch.legs) || {{}};
        const fares = (d.data && d.data.journeySearch && d.data.journeySearch.journeyFares) || {{}};

        return JSON.stringify({{
            count: journeys.length,
            journeys: journeys.slice(0, 8).map(j => {{
                const legCount = (j.legs || []).length;
                const fareEntry = fares[j.id];
                let price = null;
                let priceCur = '{currency}';
                if (fareEntry && fareEntry.fares && fareEntry.fares.length > 0) {{
                    const cheapest = fareEntry.fares.reduce((a, b) =>
                        (a.price && b.price && a.price.amount < b.price.amount) ? a : b);
                    if (cheapest.price) {{
                        price = cheapest.price.amount / 100;
                        priceCur = cheapest.price.currencyCode || priceCur;
                    }}
                }}
                return {{
                    dep: j.departureTime,
                    arr: j.arrivalTime,
                    dur: j.duration,
                    legs: legCount,
                    price: price,
                    priceCur: priceCur
                }};
            }})
        }});
    }}
    """)

    data = json.loads(result)
    if data.get("error"):
        raise RuntimeError(f"trainline api: {data['error']}")

    booking_url = (
        f"https://www.thetrainline.com/book/trains/"
        f"{from_city.lower().replace(' ', '-')}/"
        f"{to_city.lower().replace(' ', '-')}/"
        f"{date}"
    )

    routes = []
    for j in data.get("journeys", []):
        price = j.get("price") or 0.0
        if price <= 0:
            continue
        dep = j.get("dep") or ""
        arr = j.get("arr") or ""
        dur_str = j.get("dur") or ""
        legs = j.get("legs") or 1
        price_cur = j.get("priceCur") or currency

        # Parse ISO8601 duration string (PT2H15M) into minutes.
        duration = _parse_iso_duration(dur_str)

        routes.append({
            "price": float(price),
            "currency": price_cur,
            "departure": dep,
            "arrival": arr,
            "duration": duration,
            "type": "train",
            "provider": "trainline",
            "transfers": max(0, legs - 1),
            "booking_url": booking_url,
        })

    return routes


def _parse_trainline_card(card, from_city, to_city, date, currency):
    """Legacy DOM card parser — superseded by the JS API path in scrape_trainline.

    Kept for reference. scrape_trainline now calls /api/journey-search/ directly
    from the page context after navigating to the homepage, which avoids Datadome
    detection and returns structured JSON instead of requiring DOM scraping.

    The new approach:
      1. page.goto("https://www.thetrainline.com") — establishes session cookies.
      2. page.evaluate(fetch('/api/journey-search/', ...)) — calls the internal API.
      3. Parse the JSON response directly — no DOM selectors needed.

    Retained fields for backwards-compat documentation:
      price      -> d.data.journeySearch.journeyFares[id].fares[0].price.amount / 100
      currency   -> fares[0].price.currencyCode
      departure  -> journey.departureTime (ISO-8601)
      arrival    -> journey.arrivalTime   (ISO-8601)
      duration   -> _parse_iso_duration(journey.duration)
      transfers  -> len(journey.legs) - 1
      provider   -> "trainline"
      booking_url -> https://www.thetrainline.com/book/trains/{from}/{to}/{date}
    """
    # Not called — scrape_trainline uses page.evaluate() / JS API directly.
    _ = (card, from_city, to_city, date, currency)
    return None


# ---------------------------------------------------------------------------
# ÖBB
# ---------------------------------------------------------------------------

# ÖBB shop station ExtIDs (UIC/EVA) for browser URL construction.
OEBB_SHOP_STATIONS = {
    # Austria
    "vienna": "1190100",
    "wien": "1190100",
    "salzburg": "8100002",
    "innsbruck": "8100108",
    "graz": "8100173",
    "linz": "8100013",
    # Germany
    "munich": "8000261",
    "münchen": "8000261",
    "berlin": "8011160",
    "frankfurt": "8000105",
    "hamburg": "8002549",
    # Switzerland
    "zurich": "8503000",
    "zürich": "8503000",
    "geneva": "8501008",
    "basel": "8500010",
    # Italy
    "venice": "8300137",
    "milan": "8300046",
    "rome": "8300003",
    # Hungary
    "budapest": "5500017",
    # Czech Republic
    "prague": "5400014",
    "praha": "5400014",
    # Slovakia
    "bratislava": "5600002",
    # Slovenia
    "ljubljana": "7900001",
    # Croatia
    "zagreb": "7800001",
    # Poland
    "warsaw": "5100028",
    "krakow": "5100066",
}


def scrape_oebb(page, from_city, to_city, date, currency):
    from_id = OEBB_SHOP_STATIONS.get(from_city.lower())
    to_id = OEBB_SHOP_STATIONS.get(to_city.lower())
    if not from_id or not to_id:
        raise ValueError(f"no ÖBB station ID for {from_city!r} or {to_city!r}")

    # Look up the HAFAS internal numeric IDs (subset used by the timetable API).
    # The OEBB_SHOP_STATIONS map stores EVA/UIC string IDs used for booking URLs.
    # The timetable API uses integer station numbers derived from those EVA codes.
    OEBB_HAFAS_NUMBERS = {
        "vienna": (1290401, "Wien Hbf"),
        "wien": (1290401, "Wien Hbf"),
        "salzburg": (1290301, "Salzburg Hbf"),
        "innsbruck": (1290201, "Innsbruck Hbf"),
        "graz": (1290601, "Graz Hbf"),
        "linz": (1290501, "Linz Hbf"),
        "munich": (1280401, "München Hbf"),
        "münchen": (1280401, "München Hbf"),
        "berlin": (8011160, "Berlin Hbf"),
        "frankfurt": (8000105, "Frankfurt(Main)Hbf"),
        "hamburg": (8002549, "Hamburg Hbf"),
        "zurich": (8503000, "Zürich HB"),
        "zürich": (8503000, "Zürich HB"),
        "geneva": (8501008, "Genève"),
        "basel": (8500010, "Basel SBB"),
        "venice": (8300137, "Venezia Santa Lucia"),
        "milan": (8300046, "Milano Centrale"),
        "rome": (8300003, "Roma Termini"),
        "budapest": (5500017, "Budapest-Keleti"),
        "prague": (5400014, "Praha hl.n."),
        "praha": (5400014, "Praha hl.n."),
        "bratislava": (5600002, "Bratislava hl.st."),
        "ljubljana": (7900001, "Ljubljana"),
        "zagreb": (7800001, "Zagreb Gl.kol."),
        "warsaw": (5100028, "Warszawa Centralna"),
        "krakow": (5100066, "Kraków Główny"),
    }

    from_hafas = OEBB_HAFAS_NUMBERS.get(from_city.lower())
    to_hafas = OEBB_HAFAS_NUMBERS.get(to_city.lower())
    if not from_hafas or not to_hafas:
        raise ValueError(f"no ÖBB HAFAS number for {from_city!r} or {to_city!r}")

    from_num, from_name = from_hafas
    to_num, to_name = to_hafas

    _apply_stealth(page)

    # Capture the anonymousToken the SPA fetches on page load.
    # Re-requesting it would create a new session and invalidate the existing one (HTTP 440).
    token_holder = {}
    def _capture_token(resp):
        if "/api/domain/v1/anonymousToken" in resp.url:
            try:
                d = resp.json()
                token_holder["token"] = d.get("access_token", "")
            except Exception:
                pass
    page.on("response", _capture_token)

    # Navigate to the ticket page — SPA fetches anonymousToken and sets session cookies.
    page.goto("https://shop.oebbtickets.at/en/ticket", wait_until="networkidle", timeout=25000)
    _dismiss_cookies(page)

    accesstoken = token_holder.get("token", "")
    if not accesstoken:
        raise RuntimeError("oebb: could not capture anonymousToken from page load")

    # Three-step flow using the already-established session:
    #  1. POST /api/offer/v2/travelActions  -> travelActionId
    #  2. POST /api/hafas/v4/timetable      -> connections + IDs + durations (ms)
    #  3. GET  /api/offer/v1/prices         -> price per connectionId
    result = page.evaluate(f"""
    async () => {{
        const accesstoken = {json.dumps(accesstoken)};
        const hdrs = {{
            'Content-Type': 'application/json',
            'Accept': 'application/json, text/plain, */*',
            'accesstoken': accesstoken,
            'channel': 'inet',
            'clientid': '1',
            'clientversion': '2.4.11709-TSPNEU-153089-2',
            'isoffernew': 'true',
            'lang': 'en'
        }};

        // Step 1: get travelActionId for timetable search
        const ta = await fetch('/api/offer/v2/travelActions', {{
            method: 'POST',
            headers: hdrs,
            body: JSON.stringify({{
                departureTime: true,
                from: {{number: {from_num}, name: '{from_name}'}},
                to: {{number: {to_num}, name: '{to_name}'}},
                datetime: '{date}T08:00:00.000',
                customerVias: [],
                travelActionTypes: ['timetable'],
                filter: {{productTypes: [], history: false, maxEntries: 1, channel: 'inet'}}
            }})
        }});
        if (ta.status !== 200) {{
            const t = await ta.text();
            return JSON.stringify({{error: 'travelActions HTTP ' + ta.status + ': ' + t.substring(0, 200)}});
        }}
        const taData = await ta.json();
        const travelActionId = taData.travelActions && taData.travelActions[0] && taData.travelActions[0].id;
        if (!travelActionId) {{
            return JSON.stringify({{error: 'no travelActionId: ' + JSON.stringify(taData).substring(0,200)}});
        }}

        // Step 2: fetch timetable connections
        const tt = await fetch('/api/hafas/v4/timetable', {{
            method: 'POST',
            headers: hdrs,
            body: JSON.stringify({{
                travelActionId: travelActionId,
                datetimeDeparture: '{date}T08:00:00.000',
                filter: {{regionaltrains: false, direct: false, wheelchair: false, bikes: false, trains: false, motorail: false, connections: []}},
                passengers: [{{me: false, remembered: false, markedForDeath: false, type: 'ADULT', id: 1, cards: [], relations: [], isSelected: true}}],
                count: 6,
                from: {{number: {from_num}, name: '{from_name}'}},
                to: {{number: {to_num}, name: '{to_name}'}},
                timeout: {{}}
            }})
        }});
        if (tt.status !== 200) {{
            const t = await tt.text();
            return JSON.stringify({{error: 'timetable HTTP ' + tt.status + ': ' + t.substring(0, 300)}});
        }}
        const ttData = await tt.json();
        const conns = (ttData.connections || []).slice(0, 6);
        if (conns.length === 0) {{
            return JSON.stringify({{error: 'no connections', raw: JSON.stringify(ttData).substring(0, 300)}});
        }}

        // Step 3: fetch prices for all connection IDs
        const ids = conns.map(c => c.id).filter(Boolean);
        const priceUrl = '/api/offer/v1/prices?' + ids.map(id => 'connectionIds[]=' + encodeURIComponent(id)).join('&') + '&sortType=DEPARTURE&bestPriceId=undefined';
        const pr = await fetch(priceUrl, {{headers: {{'Accept': 'application/json', 'accesstoken': accesstoken, 'channel': 'inet', 'clientid': '1', 'isoffernew': 'true'}}}});
        let priceMap = {{}};
        if (pr.status === 200) {{
            const prData = await pr.json();
            (prData.offers || []).forEach(o => {{ priceMap[o.connectionId] = o.price; }});
        }}

        return JSON.stringify({{
            count: conns.length,
            connections: conns.map(c => ({{
                id: c.id,
                dep: c.from && c.from.departure,
                arr: c.to && c.to.arrival,
                durMs: c.duration,
                sections: c.sections ? c.sections.length : 1,
                price: priceMap[c.id] || null
            }}))
        }});
    }}
    """)

    data = json.loads(result)
    if data.get("error"):
        raise RuntimeError(f"oebb api: {data['error']}")

    booking_url = (
        f"https://tickets.oebb.at/en/ticket"
        f"?stationOrigExtId={from_id}"
        f"&stationDestExtId={to_id}"
        f"&outwardDate={date}"
    )

    routes = []
    for c in data.get("connections", []):
        price = c.get("price") or 0.0
        if price <= 0:
            continue
        dep = c.get("dep") or ""
        arr = c.get("arr") or ""
        dur_ms = c.get("durMs") or 0
        sections = c.get("sections") or 1

        # Duration comes as milliseconds from the HAFAS API.
        duration = int(dur_ms) // 60000

        routes.append({
            "price": float(price),
            "currency": "EUR",
            "departure": dep,
            "arrival": arr,
            "duration": duration,
            "type": "train",
            "provider": "oebb",
            "transfers": max(0, sections - 1),
            "booking_url": booking_url,
        })

    return routes


# ---------------------------------------------------------------------------
# SNCF
# ---------------------------------------------------------------------------

SNCF_STATION_CODES = {
    "paris": "FRPAR",
    "paris gare de lyon": "FRPLY",
    "paris nord": "FRPNO",
    "paris montparnasse": "FRPMO",
    "paris est": "FRPST",
    "lyon": "FRLYS",
    "marseille": "FRMRS",
    "bordeaux": "FRBOJ",
    "toulouse": "FRTLS",
    "nice": "FRNIC",
    "strasbourg": "FRSBG",
    "lille": "FRLIL",
    "nantes": "FRNTE",
    "montpellier": "FRMPL",
    "rennes": "FRRNS",
    "avignon": "FRAVT",
    "dijon": "FRDIJ",
    "brussels": "BEBMI",
    "geneva": "CHGVA",
    "zurich": "CHZRH",
    "barcelona": "ESBCN",
    "milan": "ITMIL",
    "frankfurt": "DEFRA",
    "london": "GBSPX",
}


def scrape_sncf(page, from_city, to_city, date, currency):
    from_code = SNCF_STATION_CODES.get(from_city.lower())
    to_code = SNCF_STATION_CODES.get(to_city.lower())
    if not from_code or not to_code:
        raise ValueError(f"no SNCF station code for {from_city!r} or {to_city!r}")

    url = f"https://www.sncf-connect.com/en-en/results/train/{from_code}/{to_code}/{date}"

    _apply_stealth(page)
    page.goto(url, timeout=30000, wait_until="domcontentloaded")
    _dismiss_cookies(page)

    result_selectors = [
        "[class*='journey']",
        "[class*='Journey']",
        "[class*='result']",
        "[class*='Result']",
        "[data-testid*='journey']",
        "[data-testid*='result']",
    ]
    loaded_sel = None
    for sel in result_selectors:
        try:
            page.wait_for_selector(sel, timeout=20000)
            loaded_sel = sel
            break
        except Exception:
            continue

    if loaded_sel is None:
        raise RuntimeError(f"no result cards found on SNCF page (title: {page.title()!r})")

    routes = []
    cards = page.query_selector_all(loaded_sel)

    for card in cards[:10]:
        try:
            route = _parse_sncf_card(card, from_city, to_city, date, currency, from_code, to_code)
            if route:
                routes.append(route)
        except Exception:
            continue

    return routes


def _parse_sncf_card(card, from_city, to_city, date, currency, from_code, to_code):
    text = card.inner_text()
    if not text:
        return None

    # SNCF prices in EUR with French locale (e.g. "29,00 €" or "€29.00").
    price = 0.0
    price_m = re.search(
        r"(\d+)[,.](\d{2})\s*€|€\s*(\d+)[,.](\d{2})", text
    )
    if price_m:
        if price_m.group(1):
            price = float(f"{price_m.group(1)}.{price_m.group(2)}")
        else:
            price = float(f"{price_m.group(3)}.{price_m.group(4)}")

    if price <= 0:
        return None

    times = re.findall(r"\b(\d{1,2}[hH]\d{2}|\d{1,2}:\d{2})\b", text)
    # Normalise "14h30" -> "14:30"
    times = [t.replace("h", ":").replace("H", ":") for t in times]
    departure = times[0] if len(times) >= 1 else ""
    arrival = times[1] if len(times) >= 2 else ""

    dep_iso = f"{date}T{departure}:00" if departure else date
    arr_iso = f"{date}T{arrival}:00" if arrival else date

    duration = 0
    dur_m = re.search(r"(\d+)\s*h(?:rs?)?\s*(?:(\d+)\s*m(?:in)?s?)?", text, re.IGNORECASE)
    if dur_m:
        duration = int(dur_m.group(1)) * 60 + int(dur_m.group(2) or 0)

    transfers = 0
    if re.search(r"\bdirect\b|\bsans changement\b", text, re.IGNORECASE):
        transfers = 0
    else:
        chg_m = re.search(r"(\d+)\s+(?:change|correspondance)", text, re.IGNORECASE)
        if chg_m:
            transfers = int(chg_m.group(1))

    return {
        "price": price,
        "currency": "EUR",
        "departure": dep_iso,
        "arrival": arr_iso,
        "duration": duration,
        "type": "train",
        "provider": "sncf",
        "transfers": transfers,
        "booking_url": (
            f"https://www.sncf-connect.com/en-en/result/train"
            f"/{from_code}/{to_code}/{date}"
        ),
    }


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _parse_iso_duration(s):
    """Parse ISO 8601 duration string (PT2H15M, PT45M, P0DT3H) into minutes."""
    if not s:
        return 0
    m = re.search(r"(?:(\d+)H)?(?:(\d+)M)?", s)
    if m:
        hours = int(m.group(1) or 0)
        mins = int(m.group(2) or 0)
        return hours * 60 + mins
    return 0


def _dismiss_cookies(page):
    """Accept cookie banners — try common button patterns."""
    selectors = [
        "button[id*='accept']",
        "button[id*='cookie']",
        "button[class*='accept']",
        "button[class*='cookie']",
        "[data-testid*='cookie'] button",
        "#onetrust-accept-btn-handler",
        ".cookie-accept",
        "button:has-text('Accept all')",
        "button:has-text('Accept cookies')",
        "button:has-text('I agree')",
        "button:has-text('Agree')",
        "button:has-text('OK')",
    ]
    for sel in selectors:
        try:
            btn = page.query_selector(sel)
            if btn and btn.is_visible():
                btn.click()
                page.wait_for_timeout(500)
                return
        except Exception:
            continue


def out(routes, error=None):
    payload = {"routes": routes}
    if error:
        payload["error"] = error
    print(json.dumps(payload), flush=True)


if __name__ == "__main__":
    main()
