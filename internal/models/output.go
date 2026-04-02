package models

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

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
