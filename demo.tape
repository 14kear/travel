Output demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 30ms
Set CursorBlink false

# ── ACT 1: Budget discovery — the magic moment ──

Type "# Surprise me. EUR 500, free week in June."
Enter
Sleep 300ms

Type "trvl discover --budget 500 --from 2026-06-01 --until 2026-06-30"
Enter
Sleep 300ms

Set TypingSpeed 0
Type "Top 5 value trips from HEL within EUR 500"
Enter
Enter
Type "  #  Destination       Dates                    Flight   Hotel                     Total    Slack"
Enter
Type "+----+-----------------+--------------------------+---------+---------------------------+---------+---------+"
Enter
Type "  1  Vienna (VIE)     06-09 -> 06-12  3 nights   EUR 127  Hotel Capri               EUR 265  EUR 235"
Enter
Type "  2  Krakow (KRK)     06-16 -> 06-19  3 nights   EUR 89   Hotel Stary               EUR 191  EUR 309"
Enter
Type "  3  Dubrovnik (DBV)  06-23 -> 06-26  3 nights   EUR 167  Old Town Studios           EUR 251  EUR 249"
Enter
Type "  4  Lisbon (LIS)     06-10 -> 06-14  4 nights   EUR 189  Alma Lusa Baixa-Chiado     EUR 385  EUR 115"
Enter
Type "  5  Milan (MXP)      06-15 -> 06-18  3 nights   EUR 109  Biocity Hotel              EUR 244  EUR 256"
Enter
Set TypingSpeed 30ms

Sleep 3s

# ── ACT 2: Multi-city — saves EUR 174 ──

Type "# Cheapest order for 3 cities?"
Enter
Sleep 300ms

Type "trvl multi-city HEL --visit PRG,KRK,MAD --dates 2026-07-01,2026-07-21"
Enter
Sleep 300ms

Set TypingSpeed 0
Type "Optimal route (6 permutations): HEL -> KRK -> PRG -> MAD -> HEL"
Enter
Enter
Type "  Helsinki -> Krakow      EUR 89"
Enter
Type "  Krakow -> Prague        EUR 34"
Enter
Type "  Prague -> Madrid        EUR 112"
Enter
Type "  Madrid -> Helsinki      EUR 148"
Enter
Type "  ─────────────────────────────"
Enter
Type "  Total: EUR 383  (saves EUR 174 vs worst order)"
Enter
Set TypingSpeed 30ms

Sleep 3s

# ── ACT 3: Calendar-aware — skips busy dates ──

Type "# Barcelona, skip July 11-13."
Enter
Sleep 300ms

Type "trvl when --to BCN --until 2026-08-31 --busy 2026-07-11:2026-07-13"
Enter
Sleep 300ms

Set TypingSpeed 0
Type "Best windows HEL -> BCN (avoiding Jul 11-13)"
Enter
Enter
Type "  #  Dates                  Nights  Flight   Hotel          Total"
Enter
Type "+----+------------------------+--------+---------+----------------+---------+"
Enter
Type "  1  Jun 23 -> Jun 26        3       EUR 127  Casa Bonay     EUR 268"
Enter
Type "  2  Jul 07 -> Jul 10        3       EUR 149  Yurbban        EUR 284"
Enter
Type "  3  Aug 18 -> Aug 22        4       EUR 144  Sir Victor     EUR 320"
Enter
Set TypingSpeed 30ms

Sleep 3s

# ── ACT 4: MCP — real data for AI agents ──

Type "# Works as an MCP server. Claude gets real prices."
Enter
Sleep 300ms

Type "trvl mcp install --client claude-code"
Enter
Sleep 300ms

Set TypingSpeed 0
Type "Installed trvl MCP server for claude-code"
Enter
Type "  34 tools · 5 prompts · 9 resources"
Enter
Type "  Restart claude-code to activate."
Enter
Set TypingSpeed 30ms

Sleep 2s

# ── CLOSING ──

Sleep 300ms
Type "# 34 MCP tools · 33 CLI commands · 18 providers · No API keys"
Enter
Sleep 300ms
Type "# brew install MikkoParkkola/tap/trvl"
Enter
Sleep 2s
