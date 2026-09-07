package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) handleConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		path := m.pendingDeletePath
		m.pendingDeletePath = ""
		m.toast = &Toast{Message: fmt.Sprintf("Deleting secret '%s'...", path), Type: ToastInfo}
		return m, m.deleteSecretCmd(path)

	case "n", "N", "esc", "q":
		m.pendingDeletePath = ""
		if m.selectedSecret != nil {
			m.state = StateDetail
		} else {
			m.state = StateList
		}
		return m, nil
	}
	return m, nil
}

func (m Model) renderConfirmDialog() string {
	dialogTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorDanger).
		Render("⚠️  CONFIRM SECRET DELETION")

	warningText := lipgloss.NewStyle().
		Foreground(ColorText).
		Render(fmt.Sprintf(
			"Are you sure you want to delete the secret version at:\n\n%s\n\nThis will soft-delete the current version in KV v2.",
			lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render("▶ "+m.client.Mount()+"/"+m.pendingDeletePath),
		))

	btnConfirm := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(ColorDanger).
		Padding(0, 2).
		Render("Yes, Delete [ y ]")

	btnCancel := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Padding(0, 2).
		Render("Cancel [ n / Esc ]")

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, btnConfirm, "    ", btnCancel)

	content := fmt.Sprintf("%s\n\n%s\n\n%s", dialogTitle, warningText, buttons)
	return StyleDialog.Width(64).Render(content)
}
