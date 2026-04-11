package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/MikkoParkkola/trvl/internal/lounges"
	"github.com/MikkoParkkola/trvl/internal/models"
	"github.com/MikkoParkkola/trvl/internal/preferences"
	"github.com/spf13/cobra"
)

func loungesCmd() *cobra.Command {
	var formatOut string

	cmd := &cobra.Command{
		Use:   "lounges AIRPORT",
		Short: "Search airport lounges",
		Long:  "Search for airport lounges at a given IATA airport code.\nAnnotates results with your lounge cards and frequent flyer status from preferences.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			airport := args[0]

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			result, err := lounges.SearchLounges(ctx, airport)
			if err != nil {
				return fmt.Errorf("lounge search: %w", err)
			}

			// Load preferences for access annotation.
			prefs, _ := preferences.Load()
			if prefs != nil && (len(prefs.LoungeCards) > 0 || len(prefs.FrequentFlyerPrograms) > 0) {
				var ffCards []string
				for _, ff := range prefs.FrequentFlyerPrograms {
					if ff.Alliance != "" && ff.Tier != "" {
						ffCards = append(ffCards, ff.Alliance+" "+ff.Tier)
					}
				}
				lounges.AnnotateAccessFull(result, prefs.LoungeCards, ffCards)
			}

			if formatOut == "json" {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			if result.Count == 0 {
				fmt.Printf("No lounges found at %s.\n", airport)
				return nil
			}

			fmt.Printf("Airport lounges at %s (%d found, source: %s)\n\n", result.Airport, result.Count, result.Source)

			headers := []string{"Name", "Terminal", "Type", "Hours", "Access"}
			var rows [][]string
			for _, l := range result.Lounges {
				access := ""
				if len(l.AccessibleWith) > 0 {
					access = "✓ " + l.AccessibleWith[0]
					if len(l.AccessibleWith) > 1 {
						access += fmt.Sprintf(" +%d", len(l.AccessibleWith)-1)
					}
				}
				rows = append(rows, []string{
					l.Name,
					l.Terminal,
					l.Type,
					l.OpenHours,
					access,
				})
			}

			models.FormatTable(os.Stdout, headers, rows)
			return nil
		},
	}

	cmd.Flags().StringVar(&formatOut, "format", "table", "Output format: table, json")
	return cmd
}
