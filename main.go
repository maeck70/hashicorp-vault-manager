// Package main is the entry point for the HashiCorp Vault manager application.
//
// It supports two execution modes:
//  1. CLI Secret Retrieval: when -get <path> or positional arguments are passed,
//     the secret is fetched directly to stdout and the application exits without starting the TUI.
//  2. Interactive TUI: launches the full Bubble Tea terminal interface in alternate screen buffer mode.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"vault-experiment/internal/cli"
	"vault-experiment/internal/config"
	"vault-experiment/internal/tui"
	"vault-experiment/internal/vault"
)

func main() {
	// 1. Load application configuration from CLI flags, .env, and environment variables.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// 2. Instantiate the pure Go standard library HTTP client for Vault.
	client := vault.NewClient(cfg.Address, cfg.Token, cfg.Mount)

	// 3. CLI Mode: If a secret path is requested, retrieve and output directly without initiating TUI.
	if cfg.GetPath != "" {
		if err := cli.RunGetSecret(client, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 4. Build the root Bubble Tea model with configuration and Vault client.
	model := tui.NewModel(cfg, client)

	// 5. Run the Bubble Tea program with alternate screen buffer support.
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running Vault TUI: %v\n", err)
		os.Exit(1)
	}
}
