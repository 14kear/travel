Output demo.gif

Set Shell "bash"
Set FontSize 12
Set Width 1300
Set Height 650
Set Theme "Catppuccin Mocha"
Set Padding 12
Set TypingSpeed 30ms

# ─────────────────────────────────────────────────────
# trvl — What no other travel tool does
# ─────────────────────────────────────────────────────

Type "# ✈️ Multi-airport: fly from AMS or EIN to HEL or TLL"
Enter
Sleep 400ms
Type "trvl flights AMS,EIN HEL,TLL 2026-06-15 --stops 1 --currency EUR"
Enter
Sleep 30s

Type ""
Enter
Sleep 800ms

Type "# 🏨 Hotels in Prague — best rated, with amenities, in EUR"
Enter
Sleep 400ms
Type "trvl hotels Prague --checkin 2026-06-15 --checkout 2026-06-18 --sort rating --currency EUR"
Enter
Sleep 20s

Type ""
Enter
Sleep 800ms

Type "# 🚂 Prague → Vienna — buses from EUR 9.90"
Enter
Sleep 400ms
Type "trvl ground Prague Vienna 2026-06-18 --provider regiojet,flixbus --max-price 20"
Enter
Sleep 40s

Type ""
Enter
Sleep 800ms

Type "# 📉 Monitor flight AND hotel prices — get alerts on drops"
Enter
Sleep 400ms
Type "trvl watch add AMS HEL --below 100 --currency EUR"
Enter
Sleep 3s
Type "trvl watch add --type hotel Prague --depart 2026-06-15 --return 2026-06-18 --below 60"
Enter
Sleep 3s

Type ""
Enter
Sleep 400ms
Type "# 16 tools. 6 ground providers. No API keys. Just HTTP."
Enter
Sleep 3s
