Output demo.gif

Set Shell "bash"
Set FontSize 13
Set Width 1100
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 12
Set TypingSpeed 35ms

# ─────────────────────────────────────────────────
# trvl — Real travel planning from your terminal
# ─────────────────────────────────────────────────

Type "# ✈️  Complex trip: Amsterdam → Helsinki → cheap city → back"
Enter
Sleep 800ms

Type "# Find cheapest day to fly AMS → HEL this week"
Enter
Sleep 400ms

Type "./bin/trvl dates AMS HEL --from 2026-04-04 --to 2026-04-09"
Enter
Sleep 15s

Type ""
Enter
Sleep 800ms

Type "# Where's cheap and fun from Helsinki? 🌍"
Enter
Sleep 400ms

Type "./bin/trvl explore HEL"
Enter
Sleep 25s

Type ""
Enter
Sleep 800ms

Type "# Prague! What's the weather + vibe? 🍺"
Enter
Sleep 400ms

Type "./bin/trvl destination Prague"
Enter
Sleep 8s

Type ""
Enter
Sleep 800ms

Type "# Hotels under €80, best rated"
Enter
Sleep 400ms

Type "./bin/trvl hotels Prague --checkin 2026-04-12 --checkout 2026-04-15 --max-price 80 --sort rating"
Enter
Sleep 15s

Type ""
Enter
Sleep 800ms

Type "# Prague → Amsterdam by train? 🚂"
Enter
Sleep 400ms

Type "./bin/trvl ground Prague Amsterdam 2026-04-15"
Enter
Sleep 25s

Type ""
Enter
Sleep 800ms

Type "# Track the price — alert me if it drops below €100"
Enter
Sleep 400ms

Type "./bin/trvl watch add HEL PRG --depart 2026-04-10 --below 100"
Enter
Sleep 5s

Type ""
Enter
Sleep 500ms

Type "# ✅ Full trip planned. No API keys. No monthly fees. Just HTTP."
Enter

Sleep 3s
