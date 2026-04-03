package models

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// UseColor controls whether ANSI color codes are emitted.
// Set to false when output is piped (not a terminal).
var UseColor = true

// Green wraps s in ANSI green (for positive values like savings).
func Green(s string) string {
	if !UseColor {
		return s
	}
	return "\033[32m" + s + "\033[0m"
}

// Red wraps s in ANSI red (for warnings or high prices).
func Red(s string) string {
	if !UseColor {
		return s
	}
	return "\033[31m" + s + "\033[0m"
}

// Yellow wraps s in ANSI yellow (for cautions or moderate values).
func Yellow(s string) string {
	if !UseColor {
		return s
	}
	return "\033[33m" + s + "\033[0m"
}

// Bold wraps s in ANSI bold.
func Bold(s string) string {
	if !UseColor {
		return s
	}
	return "\033[1m" + s + "\033[0m"
}

// Dim wraps s in ANSI dim/faint.
func Dim(s string) string {
	if !UseColor {
		return s
	}
	return "\033[2m" + s + "\033[0m"
}

// Cyan wraps s in ANSI cyan.
func Cyan(s string) string {
	if !UseColor {
		return s
	}
	return "\033[36m" + s + "\033[0m"
}

// Banner prints a styled box header to w.
//
//	╭─ ✈️  Flights AMS → HEL · 2026-04-08 ───────────────╮
//	│  Found 73 flights (round_trip)                       │
//	╰──────────────────────────────────────────────────────╯
func Banner(w io.Writer, icon, title, subtitle string) {
	titleLine := fmt.Sprintf(" %s  %s", icon, title)
	width := max(len(titleLine)+4, len(subtitle)+6, 56)

	topPad := width - len(titleLine) - 3
	if topPad < 1 {
		topPad = 1
	}

	fmt.Fprintf(w, "╭─%s%s╮\n", titleLine, strings.Repeat("─", topPad))

	if subtitle != "" {
		subPad := width - len(subtitle) - 5
		if subPad < 1 {
			subPad = 1
		}
		fmt.Fprintf(w, "│  %s%s│\n", subtitle, strings.Repeat(" ", subPad))
	}

	fmt.Fprintf(w, "╰%s╯\n", strings.Repeat("─", width-2))
}

// Summary prints a dimmed summary line after a table.
func Summary(w io.Writer, text string) {
	fmt.Fprintf(w, "\n  %s\n", Dim(text))
}

// BookingHint prints a hint about getting booking URLs.
func BookingHint(w io.Writer) {
	fmt.Fprintf(w, "  %s\n", Dim("Tip: --format json | jq '.flights[0].booking_url' for direct booking links"))
}

func max(a, b, c int) int {
	if b > a { a = b }
	if c > a { a = c }
	return a
}

// FormatJSON writes v as pretty-printed JSON to w.
func FormatJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// FormatTable writes a formatted ASCII table to w with aligned columns.
// Each column width is determined by the widest value in that column,
// with one space of padding on each side.
func FormatTable(w io.Writer, headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}

	// Compute column widths from headers and all rows.
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i := range min(len(row), len(widths)) {
			if len(row[i]) > widths[i] {
				widths[i] = len(row[i])
			}
		}
	}

	// Print header row.
	printRow(w, headers, widths)

	// Print separator.
	parts := make([]string, len(widths))
	for i, width := range widths {
		parts[i] = strings.Repeat("-", width+2) // +2 for padding
	}
	fmt.Fprintf(w, "+%s+\n", strings.Join(parts, "+"))

	// Print data rows.
	for _, row := range rows {
		printRow(w, row, widths)
	}
}

// printRow writes a single pipe-delimited row with padded columns.
func printRow(w io.Writer, cells []string, widths []int) {
	parts := make([]string, len(widths))
	for i, width := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		parts[i] = fmt.Sprintf(" %-*s ", width, cell)
	}
	fmt.Fprintf(w, "|%s|\n", strings.Join(parts, "|"))
}
