#!/usr/bin/env python3
"""CDP-based Trainline scraper — uses user's browser with debug port."""
import asyncio, json, sys, urllib.request, base64
try:
    import websockets
except ImportError:
    print(json.dumps({"error":"websockets not installed"})); sys.exit(0)

CDP_PORT = 9222

async def search_trainline(from_id, to_id, date):
    try:
        tabs = json.loads(urllib.request.urlopen(f"http://localhost:{CDP_PORT}/json/list").read())
    except:
        return {"error": "no browser with CDP on port 9222"}
    
    # Find a page tab
    ws_url = None
    for t in tabs:
        if t.get("type")=="page" and not t.get("url","").startswith("chrome"):
            ws_url = t["webSocketDebuggerUrl"]
            break
    if not ws_url:
        return {"error": "no usable browser tab"}
    
    async with websockets.connect(ws_url, max_size=50*1024*1024) as ws:
        await ws.send(json.dumps({"id":1,"method":"Network.enable"}))
        await ws.recv()
        
        # Navigate to results page — let the SPA make its own API calls
        url = f"https://www.thetrainline.com/book/results?journeySearchType=single&origin=urn%3Atrainline%3Ageneric%3Aloc%3A{from_id}&destination=urn%3Atrainline%3Ageneric%3Aloc%3A{to_id}&outwardDate={date}T08%3A00%3A00&outwardDateType=departAfter&passengers%5B%5D=1996-01-01&lang=en&transportModes%5B%5D=mixed"
        await ws.send(json.dumps({"id":2,"method":"Page.navigate","params":{"url":url}}))
        
        # Wait for DOM to render with prices
        await asyncio.sleep(12)
        try:
            while True: await asyncio.wait_for(ws.recv(), timeout=1)
        except: pass
        
        # Extract from rendered DOM
        js = r"""JSON.stringify({
            prices: (document.body.innerText.match(/[£€]\s*\d+[\.,]?\d*/g)||[]).slice(0,15),
            times: (document.body.innerText.match(/\d{2}:\d{2}/g)||[]).slice(0,20),
            noTickets: document.body.innerText.includes('No tickets'),
            error: document.body.innerText.includes('Something went wrong')
        })"""
        await ws.send(json.dumps({"id":10,"method":"Runtime.evaluate","params":{"expression":js}}))
        r = json.loads(await asyncio.wait_for(ws.recv(), timeout=5))
        return json.loads(r.get("result",{}).get("result",{}).get("value","{}"))

result = asyncio.run(search_trainline("4916", "4718", "2026-04-06"))
print(json.dumps(result, indent=2))
