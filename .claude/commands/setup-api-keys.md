---
description: Set up free API keys for enhanced travel data (events, restaurants, attractions)
argument-hint: "[all] or [ticketmaster|foursquare|geoapify|opentripmap]"
---

Help the user get free API keys for trvl's optional data sources. Walk them through ONE AT A TIME.

## Which keys to set up

Without keys, trvl uses Wikivoyage + OpenStreetMap (free, no key needed).
With keys, you get richer data:

| Service | What it adds | Signup URL | Time |
|---------|-------------|-----------|------|
| Ticketmaster | Events during your trip (concerts, sports, festivals) | https://developer.ticketmaster.com/ | 2 min |
| Foursquare | Restaurant ratings, tips, price levels | https://developer.foursquare.com/ | 2 min |
| Geoapify | "10 min walk from hotel" POI search | https://myprojects.geoapify.com/ | 2 min |
| OpenTripMap | Tourist attractions with Wikipedia descriptions | https://opentripmap.io/product | 2 min |

All are FREE, no credit card required, instant key generation.

## For each service, guide the user:

### Ticketmaster
1. Go to https://developer.ticketmaster.com/
2. Click "Get your API key" / Sign up
3. Create an account (email + password)
4. Go to "My Apps" → your key is shown
5. Run: `echo 'export TICKETMASTER_API_KEY="your-key-here"' >> ~/.zshrc && source ~/.zshrc`

### Foursquare
1. Go to https://developer.foursquare.com/
2. Sign up / Log in
3. Create a new project
4. Copy the API key from the project dashboard
5. Run: `echo 'export FOURSQUARE_API_KEY="your-key-here"' >> ~/.zshrc && source ~/.zshrc`

### Geoapify
1. Go to https://myprojects.geoapify.com/
2. Sign up (email or GitHub)
3. Create a project → API key is generated
4. Run: `echo 'export GEOAPIFY_API_KEY="your-key-here"' >> ~/.zshrc && source ~/.zshrc`

### OpenTripMap
1. Go to https://opentripmap.io/product
2. Click "Get API Key"
3. Register with email
4. Key is emailed to you
5. Run: `echo 'export OPENTRIPMAP_API_KEY="your-key-here"' >> ~/.zshrc && source ~/.zshrc`

## After setup, verify:
```bash
trvl events "Barcelona" --from 2026-07-01 --to 2026-07-08
# Should show events (not "set TICKETMASTER_API_KEY")

trvl nearby 41.38 2.17 --category restaurant
# Should show ratings if Foursquare key is set
```

## One-liner for all (after getting keys):
```bash
cat >> ~/.zshrc << 'EOF'
export TICKETMASTER_API_KEY="xxx"
export FOURSQUARE_API_KEY="xxx"
export GEOAPIFY_API_KEY="xxx"
export OPENTRIPMAP_API_KEY="xxx"
EOF
source ~/.zshrc
```

Walk the user through each one. Open the signup URL for them. Wait for them to paste their key. Confirm it works.
