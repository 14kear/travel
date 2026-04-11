Output demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 30ms
Set CursorBlink false

# ── ACT 1: Natural language budget search ──

Type "# I have EUR 500 and a free week in June. Surprise me."
Enter
Sleep 600ms

Type "trvl discover --from 2026-06-01 --until 2026-06-30 --budget 500"
Enter
Sleep 800ms

Set TypingSpeed 0
Type "Top 5 value trips from HEL within 500 EUR (2026-06-01 to 2026-06-30)"
Enter
Enter
Type "  #  Destination        Dates                     Nights  Flight   Hotel                           Total    Slack    Notes"
Enter
Type "+----+-------------------+--------------------------+--------+---------+--------------------------------+---------+--------+----------+"
Enter
Type "  1  Vienna (VIE)       2026-06-09 -> 2026-06-12   3       EUR 127  EUR 138 (Hotel Capri)           EUR 265  EUR 235  nonstop"
Enter
Type "  2  Krakow (KRK)       2026-06-16 -> 2026-06-19   3       EUR 89   EUR 102 (Hotel Stary)           EUR 191  EUR 309  nonstop"
Enter
Type "  3  Dubrovnik (DBV)    2026-06-23 -> 2026-06-26   3       EUR 167  EUR 84 (Old Town Studios)       EUR 251  EUR 249  1 stop"
Enter
Type "  4  Lisbon (LIS)       2026-06-10 -> 2026-06-14   4       EUR 189  EUR 196 (Alma Lusa)             EUR 385  EUR 115  1 stop"
Enter
Type "  5  Milan (MXP)        2026-06-15 -> 2026-06-18   3       EUR 109  EUR 135 (Biocity Hotel)         EUR 244  EUR 256  nonstop"
Enter
Set TypingSpeed 30ms

Sleep 4s

# ── ACT 2: Multi-city optimizer ──

Type "# Prague, Krakow, Madrid — what's the cheapest order?"
Enter
Sleep 600ms

Type "trvl multi-city HEL --visit PRG,KRK,MAD --dates 2026-07-01,2026-07-21"
Enter
Sleep 800ms

Set TypingSpeed 0
Type "Multi-city from HEL: optimal order found (6 permutations checked)"
Enter
Enter
Type "Route: HEL -> KRK -> PRG -> MAD -> HEL"
Enter
Enter
Type "  From                To                   Price"
Enter
Type "+---------------------+---------------------+---------+"
Enter
Type "  Helsinki (HEL)      Krakow (KRK)         EUR 89"
Enter
Type "  Krakow (KRK)        Prague (PRG)          EUR 34"
Enter
Type "  Prague (PRG)         Madrid (MAD)          EUR 112"
Enter
Type "  Madrid (MAD)         Helsinki (HEL)        EUR 148"
Enter
Type "+---------------------+---------------------+---------+"
Enter
Type "  Total                                      EUR 383"
Enter
Type "  Savings vs worst                           EUR 174"
Enter
Set TypingSpeed 30ms

Sleep 4s

# ── ACT 3: Calendar-aware planning ──

Type "# Barcelona in summer? Skip the 11-13th, I'm busy."
Enter
Sleep 600ms

Type "trvl when --to BCN --from 2026-06-01 --until 2026-08-31 --busy 2026-07-11:2026-07-13"
Enter
Sleep 800ms

Set TypingSpeed 0
Type "Travel windows from HEL to BCN (avoiding 2026-07-11 to 2026-07-13)"
Enter
Enter
Type "  #  Departure   Return      Nights  Flight   Hotel    Total    Hotel name"
Enter
Type "+----+-----------+-----------+--------+---------+--------+---------+--------------------+"
Enter
Type "  1  2026-06-23  2026-06-26  3       EUR 127  EUR 141  EUR 268  Casa Bonay"
Enter
Type "  2  2026-07-07  2026-07-10  3       EUR 149  EUR 135  EUR 284  Yurbban Passage"
Enter
Type "  3  2026-06-16  2026-06-20  4       EUR 139  EUR 184  EUR 323  Hotel 1898"
Enter
Type "  4  2026-08-18  2026-08-22  4       EUR 144  EUR 176  EUR 320  Sir Victor Hotel"
Enter
Type "  5  2026-08-04  2026-08-08  4       EUR 158  EUR 192  EUR 350  Hotel Brummell"
Enter
Set TypingSpeed 30ms

Sleep 4s

# ── ACT 4: Preference memory ──

Type "# trvl remembers your style."
Enter
Sleep 400ms

Type "trvl prefs"
Enter
Sleep 400ms

Set TypingSpeed 0
Type '{"home_airports":["HEL"],"preferred_districts":{"Prague":["Prague 1","Prague 5"]},'
Enter
Type ' "exclude_property_types":["hostel","guest house"],"display_currency":"EUR"}'
Enter
Set TypingSpeed 30ms

Sleep 3s

# ── CLOSING ──

Sleep 400ms
Type "# 34 MCP tools · 33 CLI commands · 18 providers · No API keys"
Enter
Sleep 400ms
Type "# brew install MikkoParkkola/tap/trvl"
Enter
Sleep 3s
