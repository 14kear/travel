package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	trvlmcp "github.com/MikkoParkkola/trvl/mcp"
)

var readmeToolMarkers = []string{
	"search_flights",
	"search_dates",
	"search_hotels",
	"hotel_prices",
	"hotel_reviews",
	"hotel_rooms",
	"destination_info",
	"calculate_trip_cost",
	"weekend_getaway",
	"suggest_dates",
	"optimize_multi_city",
	"nearby_places",
	"travel_guide",
	"local_events",
	"search_ground",
	"search_airport_transfers",
	"search_restaurants",
	"search_deals",
	"plan_trip",
	"search_route",
	"get_preferences",
	"detect_travel_hacks",
	"detect_accommodation_hacks",
	"search_natural",
	"list_trips",
	"get_trip",
	"create_trip",
	"add_trip_leg",
	"mark_trip_booked",
	"get_weather",
	"get_baggage_rules",
	"find_trip_window",
}

func bundledSkillMarkdownCount(t *testing.T) int {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join("..", "..", ".claude", "skills"))
	if err != nil {
		t.Fatalf("ReadDir(.claude/skills): %v", err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			count++
		}
	}
	return count
}

func registeredMCPToolCount(t *testing.T) int {
	t.Helper()

	serverValue := reflect.ValueOf(trvlmcp.NewServer())
	if serverValue.Kind() != reflect.Pointer || serverValue.IsNil() {
		t.Fatal("mcp.NewServer should return a non-nil server pointer")
	}

	tools := serverValue.Elem().FieldByName("tools")
	if !tools.IsValid() {
		t.Fatal("mcp.Server should expose an internal tools field")
	}

	return tools.Len()
}

func TestPublicDocsAdvertiseCurrentCounts(t *testing.T) {
	t.Parallel()

	toolCount := registeredMCPToolCount(t)
	cliCommandCount := len(rootCmd.Commands())
	watchSubcommandCount := len(watchCmd().Commands())
	skillCount := bundledSkillMarkdownCount(t)

	checks := []struct {
		path      string
		required  []string
		forbidden []string
	}{
		{
			path: filepath.Join("..", "..", "README.md"),
			required: []string{
				fmt.Sprintf("%d travel tools for your AI assistant", toolCount),
				fmt.Sprintf("standalone CLI with %d commands", cliCommandCount),
				fmt.Sprintf("%d travel tools available", toolCount),
				fmt.Sprintf("Full v2025-11-25 — %d tools", toolCount),
				fmt.Sprintf("%d commands (+ %d watch subcommands)", cliCommandCount, watchSubcommandCount),
				fmt.Sprintf("Full JSON Schema validation for all %d tool responses", toolCount),
				fmt.Sprintf("The repo includes %d Claude Code skill file", skillCount),
			},
			forbidden: []string{
				"31 travel tools for your AI assistant",
				"standalone CLI with 31 commands",
				"31 travel tools available",
				"Full v2025-11-25 — 31 tools",
				"31 commands (+ 6 watch subcommands)",
				"Full JSON Schema validation for all 31 tool responses",
				"29 travel tools for your AI assistant",
				"standalone CLI with 29 commands",
				"29 travel tools available",
				"Full v2025-11-25 — 29 tools",
				"29 commands (+ 6 watch subcommands)",
				"Full JSON Schema validation for all 29 tool responses",
				"The repo includes 4 skill files",
			},
		},
		{
			path: filepath.Join("..", "..", "AGENTS.md"),
			required: []string{
				fmt.Sprintf("trvl is installed with %d MCP tools and %d bundled Claude skill", toolCount, skillCount),
			},
			forbidden: []string{
				"trvl is installed with 32 MCP tools and 5 skills",
				"trvl is installed with 32 MCP tools and 4 skills",
				"trvl is installed with 31 MCP tools and 5 skills",
				"trvl is installed with 31 MCP tools and 4 skills",
				"installed with 22 MCP tools and 5 skills",
				"You now have 22 MCP tools available.",
			},
		},
		{
			path: filepath.Join("..", "..", "demo.tape"),
			required: []string{
				fmt.Sprintf("# %d MCP tools · %d CLI commands · 17 providers · No API keys", toolCount, cliCommandCount),
			},
			forbidden: []string{
				"# 31 MCP tools · 31 CLI commands · 17 providers · No API keys",
				"# 29 MCP tools · 29 CLI commands · 17 providers · No API keys",
			},
		},
		{
			path: filepath.Join("..", "..", ".claude-plugin", "plugin.json"),
			required: []string{
				fmt.Sprintf("%d MCP tools", toolCount),
			},
			forbidden: []string{
				"16 MCP tools",
				"31 MCP tools",
			},
		},
		{
			path: filepath.Join("..", "..", ".claude", "skills", "trvl.md"),
			required: []string{
				fmt.Sprintf("## CORE TOOLS (selected high-signal tools; trvl exposes %d MCP tools overall via gateway_invoke server=\"trvl\")", toolCount),
				"Bus/train/ferry (16 providers)",
				"`search_airport_transfers`",
				"`plan_trip`",
				"`search_route`",
				"`get_weather`",
				"`get_baggage_rules`",
			},
			forbidden: []string{
				"## TOOLS (via gateway_invoke server=\"trvl\")",
				"trvl exposes 31 MCP tools overall",
				"Bus/train (6 providers)",
			},
		},
	}

	for _, check := range checks {
		check := check
		t.Run(filepath.Base(check.path), func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(check.path)
			if err != nil {
				t.Fatalf("ReadFile(%q): %v", check.path, err)
			}
			text := string(data)

			for _, needle := range check.required {
				if !strings.Contains(text, needle) {
					t.Errorf("%s missing required text %q", check.path, needle)
				}
			}
			for _, needle := range check.forbidden {
				if strings.Contains(text, needle) {
					t.Errorf("%s still contains stale text %q", check.path, needle)
				}
			}

			if filepath.Base(check.path) == "README.md" {
				for _, tool := range readmeToolMarkers {
					marker := fmt.Sprintf("**%s**", tool)
					if count := strings.Count(text, marker); count != 1 {
						t.Errorf("%s should mention %s exactly once in the MCP tool table, got %d", check.path, marker, count)
					}
				}
			}
		})
	}
}
