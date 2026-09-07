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
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	client := vault.NewClient(cfg.Address, cfg.Token, cfg.Mount)
	model := tui.NewModel(cfg, client)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running Vault TUI: %v\n", err)
		os.Exit(1)
	}
}
