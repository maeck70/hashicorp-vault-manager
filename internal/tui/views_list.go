// views_list.go implements the secret path explorer, supporting real-time
// filtering, path selection, and initiating CRUD operations.

package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleListKeys handles navigation, filtering, and action triggers on the secrets list.
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If currently in filter input mode
	if m.isFiltering {
		switch msg.String() {
		case "esc":
			m.isFiltering = false
			m.filterInput.Blur()
			return m, nil
		case "enter":
			m.isFiltering = false
			m.filterInput.Blur()
			return m, nil
		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			m.applyFilter()
			return m, cmd
		}
	}

	// Normal list navigation mode
	switch msg.String() {
	case "up", "k":
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}
		return m, nil

	case "down", "j":
		if m.selectedIndex < len(m.filteredList)-1 {
			m.selectedIndex++
		}
		return m, nil

	case "enter":
		if len(m.filteredList) > 0 && m.selectedIndex < len(m.filteredList) {
			path := m.filteredList[m.selectedIndex]
			m.toast = &Toast{Message: fmt.Sprintf("Loading secret '%s'...", path), Type: ToastInfo}
			return m, m.loadSecretDetailCmd(path)
		}
		return m, nil

	case "n":
		// New secret
		m.isEditingExisting = false
		m.editPathInput.SetValue("")
		m.initEditorRows(1)
		m.editKeyInputs[0].SetValue("")
		m.editValInputs[0].SetValue("")
		m.activeField = 0
		m.editPathInput.Focus()
		m.state = StateEditor
		return m, nil

	case "e":
		// Edit selected secret
		if len(m.filteredList) > 0 && m.selectedIndex < len(m.filteredList) {
			path := m.filteredList[m.selectedIndex]
			m.toast = &Toast{Message: fmt.Sprintf("Loading '%s' for edit...", path), Type: ToastInfo}
			// Load detail then transition to editor
			return m, m.loadSecretDetailCmd(path)
		}
		return m, nil

	case "d":
		// Delete selected secret
		if len(m.filteredList) > 0 && m.selectedIndex < len(m.filteredList) {
			path := m.filteredList[m.selectedIndex]
			m.pendingDeletePath = path
			m.state = StateConfirm
		}
		return m, nil

	case "/":
		m.isFiltering = true
		m.filterInput.Focus()
		return m, nil

	case "r":
		m.toast = &Toast{Message: "Refreshing secret list...", Type: ToastInfo}
		return m, m.loadSecretsCmd()

	case "q":
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) renderListView() string {
	// Top Header
	title := StyleTitle.Render("⚡ HASHICORP VAULT")
	mountBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimaryDark).
		Background(ColorPrimary).
		Padding(0, 1).
		Render(fmt.Sprintf("KV: %s/", m.client.Mount()))

	statusBadge := BadgeOnline.Render("● ONLINE")
	if m.health != nil && m.health.Sealed {
		statusBadge = BadgeSealed.Render("● SEALED")
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center, title, mountBadge, " ", statusBadge)

	// Filter / Search Bar
	var filterBar string
	if m.isFiltering {
		filterBar = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSecondary).
			Padding(0, 1).
			Width(64).
			Render(m.filterInput.View())
	} else if m.filterInput.Value() != "" {
		filterBar = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Render(fmt.Sprintf("Filter: %s (Press [/] to edit, [r] to reset)", m.filterInput.Value()))
	} else {
		filterBar = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Render("Press [/] to filter secrets • [n] to create new secret")
	}

	// Secret List Box
	var listContent string
	if len(m.filteredList) == 0 {
		listContent = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Padding(2, 4).
			Render("No secrets found in this mount.\n\nPress [ n ] to create your first secret!\nPress [ r ] to reload from Vault.")
	} else {
		lines := make([]string, 0, len(m.filteredList))
		for i, key := range m.filteredList {
			cleanKey := strings.Trim(key, "/")
			if i == m.selectedIndex {
				row := StyleSelectedItem.Render(fmt.Sprintf("▶ 🔑 %s", cleanKey))
				lines = append(lines, row)
			} else {
				row := StyleNormalItem.Render(fmt.Sprintf("  🔑 %s", cleanKey))
				lines = append(lines, row)
			}
		}
		listContent = strings.Join(lines, "\n")
	}

	boxWidth := 72
	if m.width > 80 {
		boxWidth = min(m.width-6, 100)
	}

	listBox := StyleCard.Width(boxWidth).Render(
		fmt.Sprintf(
			"%s\n\n%s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(fmt.Sprintf("SECRET PATHS (%d)", len(m.filteredList))),
			listContent,
		),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		filterBar,
		"",
		listBox,
	)
}
