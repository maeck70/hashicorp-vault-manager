// Package main is the entry point for the HashiCorp Vault TUI application.
//
// It parses CLI flags, merges environment variables from .env, initializes
// the standard-library Vault HTTP client, and launches the Bubble Tea
// terminal user interface in alternate screen buffer mode.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

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

	// 3. Build the root Bubble Tea model with configuration and Vault client.
	model := tui.NewModel(cfg, client)

	// 4. Run the Bubble Tea program with alternate screen buffer support.
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running Vault TUI: %v\n", err)
		os.Exit(1)
	}
}
