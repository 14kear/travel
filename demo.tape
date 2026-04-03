Output demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 900
Set Height 500
Set Theme "Catppuccin Mocha"
Set Padding 10

Type "# Search flights Helsinki → Barcelona"
Enter
Sleep 500ms

Type "./bin/trvl flights HEL BCN 2026-07-01 --return 2026-07-08"
Enter
Sleep 12s

Type ""
Enter
Sleep 500ms

Type "# Search hotels in Barcelona"
Enter
Sleep 500ms

Type "./bin/trvl hotels Barcelona --checkin 2026-07-01 --checkout 2026-07-08"
Enter
Sleep 12s

Type ""
Enter
Sleep 500ms

Type "# Search ground transport London → Paris (Eurostar + FlixBus + DB)"
Enter
Sleep 500ms

Type "./bin/trvl ground London Paris 2026-07-01"
Enter
Sleep 12s

Type ""
Enter
Sleep 500ms

Type "# Set up price watch with alert"
Enter
Sleep 500ms

Type "./bin/trvl watch add HEL BCN --depart 2026-07-01 --return 2026-07-08 --below 200"
Enter
Sleep 5s

Sleep 2s
