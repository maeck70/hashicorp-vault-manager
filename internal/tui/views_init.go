// views_init.go provides the terminal interface and key handlers for
// initializing an uninitialized Vault instance and unsealing a sealed Vault.

package tui

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleInitKeys processes keyboard navigation and submission on the initialization screen.
func (m Model) handleInitKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "shift+tab", "up", "down":
		if m.initSharesInput.Focused() {
			m.initSharesInput.Blur()
			m.initThresholdInput.Focus()
		} else {
			m.initThresholdInput.Blur()
			m.initSharesInput.Focus()
		}
		return m, nil

	case "enter":
		shares, err1 := strconv.Atoi(m.initSharesInput.Value())
		threshold, err2 := strconv.Atoi(m.initThresholdInput.Value())
		if err1 != nil || shares < 1 {
			m.toast = &Toast{Message: "Invalid key shares (minimum 1)", Type: ToastError}
			return m, clearToastCmd()
		}
		if err2 != nil || threshold < 1 || threshold > shares {
			m.toast = &Toast{Message: "Threshold must be between 1 and key shares", Type: ToastError}
			return m, clearToastCmd()
		}
		m.toast = &Toast{Message: "Initializing Vault server...", Type: ToastInfo}
		return m, m.initializeVaultCmd(shares, threshold)

	case "esc", "q":
		return m, tea.Quit
	}

	var cmd tea.Cmd
	if m.initSharesInput.Focused() {
		m.initSharesInput, cmd = m.initSharesInput.Update(msg)
	} else {
		m.initThresholdInput, cmd = m.initThresholdInput.Update(msg)
	}
	return m, cmd
}

func (m Model) handleUnsealKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		key := m.unsealKeyInput.Value()
		if key == "" {
			m.toast = &Toast{Message: "Please provide a valid unseal key", Type: ToastError}
			return m, clearToastCmd()
		}
		m.toast = &Toast{Message: "Submitting unseal key...", Type: ToastInfo}
		return m, m.unsealVaultCmd(key)

	case "esc", "q":
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.unsealKeyInput, cmd = m.unsealKeyInput.Update(msg)
	return m, cmd
}

func (m Model) renderInitView() string {
	title := StyleTitle.Render("⚡ VAULT INITIALIZATION WIZARD")
	subtitle := StyleSubtitle.Render(fmt.Sprintf("Target: %s", m.client.Address()))

	cardContent := fmt.Sprintf(
		"%s\n\n%s\n\n%s\n%s\n\n%s\n\n%s",
		lipgloss.NewStyle().Bold(true).Foreground(ColorWarning).Render("⚠ Vault is currently UNINITIALIZED"),
		lipgloss.NewStyle().Foreground(ColorTextDim).Render(
			"For local development/experimentation, 1 share with threshold 1 is recommended.\n"+
				"Upon initialization, your generated root token and unseal key will be\n"+
				"automatically saved to your local .env file and used to unseal.",
		),
		m.initSharesInput.View(),
		m.initThresholdInput.View(),
		lipgloss.NewStyle().Foreground(ColorSecondary).Render("Press [ Enter ] to initialize & unseal automatically"),
		lipgloss.NewStyle().Foreground(ColorMuted).Render("[ Tab ] Switch inputs • [ q / Esc ] Quit"),
	)

	card := StyleCard.Width(68).Render(cardContent)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Center, title, subtitle),
		"",
		card,
	)
}

func (m Model) renderUnsealView() string {
	title := StyleTitle.Render("🔓 VAULT UNSEAL WIZARD")
	subtitle := StyleSubtitle.Render(fmt.Sprintf("Target: %s", m.client.Address()))

	cardContent := fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s\n\n%s",
		BadgeSealed.Render("● VAULT IS SEALED"),
		lipgloss.NewStyle().Foreground(ColorTextDim).Render(
			"Vault cryptographic storage is currently sealed.\n"+
				"Enter an unseal shard key below to unlock the vault.",
		),
		m.unsealKeyInput.View(),
		lipgloss.NewStyle().Foreground(ColorSecondary).Render("Press [ Enter ] to submit unseal shard"),
		lipgloss.NewStyle().Foreground(ColorMuted).Render("[ q / Esc ] Quit"),
	)

	card := StyleCard.Width(68).Render(cardContent)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Center, title, subtitle),
		"",
		card,
	)
}
