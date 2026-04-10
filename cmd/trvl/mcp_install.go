package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// mcpInstallCmd adds trvl to Claude Desktop's MCP configuration file.
//
// This is the single most important friction reducer for non-technical users:
// instead of asking them to find and hand-edit a JSON config file, they run
// one command and trvl installs itself as an MCP server for Claude Desktop.
func mcpInstallCmd() *cobra.Command {
	var (
		client string
		force  bool
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "install-claude-desktop",
		Short: "Install trvl as an MCP server for Claude Desktop (no JSON editing)",
		Long: `Install trvl into Claude Desktop's MCP configuration automatically.

No JSON editing, no file-path hunting. Just run this command and restart
Claude Desktop — trvl will appear as an MCP server ready to search flights
and hotels.

By default, installs into Claude Desktop. Use --client to target other
MCP-aware clients:

  trvl mcp install-claude-desktop
  trvl mcp install-claude-desktop --client cursor
  trvl mcp install-claude-desktop --dry-run    # show what would change
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(client, force, dryRun)
		},
	}

	cmd.Flags().StringVar(&client, "client", "claude-desktop", "MCP client to configure: claude-desktop, cursor, claude-code")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing trvl entry without asking")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the planned change without writing the file")

	return cmd
}

// clientConfigPath returns the MCP config file path for the given client.
func clientConfigPath(client string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	switch strings.ToLower(client) {
	case "claude-desktop", "claude":
		switch runtime.GOOS {
		case "darwin":
			return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), nil
		case "linux":
			return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"), nil
		case "windows":
			appdata := os.Getenv("APPDATA")
			if appdata == "" {
				appdata = filepath.Join(home, "AppData", "Roaming")
			}
			return filepath.Join(appdata, "Claude", "claude_desktop_config.json"), nil
		}
	case "cursor":
		return filepath.Join(home, ".cursor", "mcp.json"), nil
	case "claude-code":
		return filepath.Join(home, ".claude.json"), nil
	}
	return "", fmt.Errorf("unknown client %q (supported: claude-desktop, cursor, claude-code)", client)
}

// trvlBinaryPath returns the absolute path to the currently-running trvl
// binary, so the config points at the exact tool the user just invoked.
func trvlBinaryPath() (string, error) {
	// Prefer argv[0] resolved, then fall back to $PATH lookup.
	if exe, err := os.Executable(); err == nil {
		if abs, err := filepath.Abs(exe); err == nil {
			return abs, nil
		}
	}
	if path, err := exec.LookPath("trvl"); err == nil {
		return filepath.Abs(path)
	}
	return "", fmt.Errorf("cannot locate trvl binary")
}

// runInstall wires trvl into the target client's MCP config.
func runInstall(client string, force, dryRun bool) error {
	cfgPath, err := clientConfigPath(client)
	if err != nil {
		return err
	}
	binary, err := trvlBinaryPath()
	if err != nil {
		return err
	}

	// Load existing config (or start with a fresh one).
	cfg := map[string]any{}
	if data, err := os.ReadFile(cfgPath); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("parse existing config %s: %w (fix the file or use --force to overwrite)", cfgPath, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read config %s: %w", cfgPath, err)
	}

	// Ensure mcpServers section exists.
	servers, _ := cfg["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}

	// Check for existing trvl entry.
	if existing, ok := servers["trvl"]; ok && !force {
		if dryRun {
			fmt.Printf("trvl is already installed in %s\n  existing: %v\n  would not change (use --force to overwrite)\n", cfgPath, existing)
			return nil
		}
		fmt.Printf("trvl is already installed in %s\nUse --force to overwrite.\n", cfgPath)
		return nil
	}

	// Write the trvl entry.
	servers["trvl"] = map[string]any{
		"command": binary,
		"args":    []string{"mcp"},
	}
	cfg["mcpServers"] = servers

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	if dryRun {
		fmt.Printf("Would write to %s:\n\n%s\n", cfgPath, out)
		return nil
	}

	// Create parent directory if missing.
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Backup existing file if present.
	if _, err := os.Stat(cfgPath); err == nil {
		backup := cfgPath + ".trvl.bak"
		if data, err := os.ReadFile(cfgPath); err == nil {
			_ = os.WriteFile(backup, data, 0o644)
		}
	}

	if err := os.WriteFile(cfgPath, out, 0o644); err != nil {
		return fmt.Errorf("write config %s: %w", cfgPath, err)
	}

	fmt.Printf("Installed trvl as MCP server for %s.\n", client)
	fmt.Printf("  config: %s\n", cfgPath)
	fmt.Printf("  binary: %s\n", binary)
	fmt.Println()
	fmt.Println("Restart Claude Desktop (or your MCP client) to pick up the change.")
	fmt.Println("Then ask your assistant: \"Use trvl to find flights from AMS to BCN next month.\"")
	return nil
}
