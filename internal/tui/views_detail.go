// views_detail.go renders the secret key-value inspection view, including
// formatted sub-trees for JSON objects, masking toggles, and clipboard actions.

package tui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vault-experiment/internal/generator"
)

// handleDetailKeys handles keyboard actions while viewing secret details.
func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace", "left", "h":
		m.state = StateList
		return m, nil

	case "up", "k":
		if m.detailRowIndex > 0 {
			m.detailRowIndex--
		}
		return m, nil

	case "down", "j":
		if m.detailRowIndex < len(m.sortedKeys)-1 {
			m.detailRowIndex++
		}
		return m, nil

	case "m":
		// Toggle masking
		m.maskValues = !m.maskValues
		if m.maskValues {
			m.toast = &Toast{Message: "Secret values masked (••••)", Type: ToastInfo}
		} else {
			m.toast = &Toast{Message: "Secret values revealed", Type: ToastInfo}
		}
		return m, clearToastCmd()

	case "u":
		// Undelete / restore soft-deleted secret version
		if m.selectedSecret != nil && m.selectedSecret.IsDeleted {
			m.toast = &Toast{Message: fmt.Sprintf("Restoring secret '%s'...", m.selectedSecret.Path), Type: ToastInfo}
			return m, m.undeleteSecretCmd(m.selectedSecret.Path, m.selectedSecret.Version)
		}
		return m, nil

	case "y", "c":
		// Copy selected value to clipboard
		if m.selectedSecret != nil {
			if m.selectedSecret.IsDeleted {
				m.toast = &Toast{Message: "Secret is deleted (no active value to copy)", Type: ToastWarning}
				return m, clearToastCmd()
			}
			if len(m.sortedKeys) > 0 && m.detailRowIndex < len(m.sortedKeys) {
				key := m.sortedKeys[m.detailRowIndex]
				val := m.selectedSecret.Data[key]
				return m.copyToClipboard(val)
			}
		}
		return m, nil

	case "g":
		// Generate external Go retrieval file
		if m.selectedSecret != nil {
			if m.selectedSecret.IsDeleted {
				m.toast = &Toast{Message: "Cannot generate retrieval code for a deleted secret", Type: ToastWarning}
				return m, clearToastCmd()
			}
			code := generator.GenerateGoRetrievalCode(m.client.Address(), m.client.Mount(), m.selectedSecret.Path, m.selectedSecret.Data)
			filename := generator.SanitizeFilename(m.selectedSecret.Path)
			filePath, err := generator.SaveRetrievalFile("examples", filename, code)
			if err != nil {
				m.toast = &Toast{Message: fmt.Sprintf("Failed to save Go file: %v", err), Type: ToastError}
			} else {
				m.toast = &Toast{Message: fmt.Sprintf("✓ Generated Go file: %s", filePath), Type: ToastSuccess}
			}
			return m, clearToastCmd()
		}
		return m, nil

	case "G":
		// Copy generated Go code directly to clipboard
		if m.selectedSecret != nil {
			if m.selectedSecret.IsDeleted {
				m.toast = &Toast{Message: "Cannot generate retrieval code for a deleted secret", Type: ToastWarning}
				return m, clearToastCmd()
			}
			code := generator.GenerateGoRetrievalCode(m.client.Address(), m.client.Mount(), m.selectedSecret.Path, m.selectedSecret.Data)
			return m.copyToClipboard(code)
		}
		return m, nil

	case "e":
		// Populate editor with existing secret data
		if m.selectedSecret != nil {
			m.isEditingExisting = true
			m.editPathInput.SetValue(m.selectedSecret.Path)
			m.editPathInput.Blur()

			keys := make([]string, 0, len(m.selectedSecret.Data))
			for k := range m.selectedSecret.Data {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			m.initEditorRows(max(1, len(keys)))
			for i, k := range keys {
				m.editKeyInputs[i].SetValue(k)
				m.editValInputs[i].SetValue(m.selectedSecret.Data[k])
			}
			m.activeField = 1 // focus first key
			m.editKeyInputs[0].Focus()
			m.state = StateEditor
		}
		return m, nil

	case "d":
		// Delete
		if m.selectedSecret != nil {
			m.pendingDeletePath = m.selectedSecret.Path
			m.state = StateConfirm
		}
		return m, nil

	case "q":
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) renderDetailView() string {
	if m.selectedSecret == nil {
		return "No secret selected."
	}

	title := StyleTitle.Render("🔑 SECRET VIEWER")
	pathBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render(fmt.Sprintf("%s/%s", m.client.Mount(), m.selectedSecret.Path))

	versionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorText).
		Background(ColorPrimaryDark).
		Padding(0, 1)

	versionText := fmt.Sprintf("v%d", m.selectedSecret.Version)
	if m.selectedSecret.IsDeleted {
		versionStyle = BadgeDanger
		versionText = fmt.Sprintf("v%d ● SOFT-DELETED", m.selectedSecret.Version)
	}
	versionBadge := versionStyle.Render(versionText)

	header := lipgloss.JoinHorizontal(lipgloss.Center, title, pathBadge, " ", versionBadge)

	// Metadata
	metaStr := fmt.Sprintf("Created: %s", m.selectedSecret.CreatedTime.Format("2006-01-02 15:04:05 MST"))
	if m.selectedSecret.IsDeleted && !m.selectedSecret.DeletionTime.IsZero() {
		metaStr += fmt.Sprintf("  •  Deleted: %s", m.selectedSecret.DeletionTime.Format("2006-01-02 15:04:05 MST"))
	}
	metaInfo := lipgloss.NewStyle().Foreground(ColorTextDim).Render(metaStr)

	boxWidth := 76
	if m.width > 80 {
		boxWidth = min(m.width-6, 104)
	}

	var tableBox string
	var helpHints string

	if m.selectedSecret.IsDeleted {
		warnContent := fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorDanger).Render("⚠️  THIS SECRET VERSION WAS SOFT-DELETED IN VAULT"),
			lipgloss.NewStyle().Foreground(ColorText).Render("The current version has been marked as deleted and contains no active secret data."),
			lipgloss.NewStyle().Foreground(ColorSecondary).Render("Options:\n  • Press [ d ] to permanently destroy metadata (purges it from the list)\n  • Press [ u ] to restore (undelete) this secret version\n  • Press [ e ] to edit and commit a new version\n  • Press [ Esc ] to return to secret list"),
		)
		tableBox = StyleCard.Width(boxWidth).BorderForeground(ColorDanger).Render(warnContent)
		helpHints = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Render("[ d ] Permanently Destroy • [ u ] Undelete • [ e ] Re-create • [ Esc ] Back")
	} else {
		// Key-Value Table
		keys := m.sortedKeys
		if len(keys) == 0 {
			keys = make([]string, 0, len(m.selectedSecret.Data))
			for k := range m.selectedSecret.Data {
				keys = append(keys, k)
			}
			sort.Strings(keys)
		}

		var rows []string
		headerRow := fmt.Sprintf(
			"  %-24s   %-40s",
			StyleKeyHeader.Render("KEY"),
			StyleValHeader.Render("VALUE"),
		)
		rows = append(rows, headerRow)

		for i, k := range keys {
			val := m.selectedSecret.Data[k]

			cursor := "  "
			rowStyle := StyleNormalItem
			if i == m.detailRowIndex {
				cursor = "▶ "
				rowStyle = StyleSelectedItem
			}

			// Check if value is a JSON string
			var parsedMap map[string]any
			isJSON := isJSONString(val) && json.Unmarshal([]byte(val), &parsedMap) == nil

			if isJSON {
				keyCol := StyleKeyCell.Render(fmt.Sprintf("%-24s", k)) + " " + BadgeJSON.Render("JSON")
				row := rowStyle.Render(fmt.Sprintf("%s%s", cursor, keyCol))
				rows = append(rows, row)

				// Sub-fields
				subKeys := make([]string, 0, len(parsedMap))
				for sk := range parsedMap {
					subKeys = append(subKeys, sk)
				}
				sort.Strings(subKeys)

				for _, sk := range subKeys {
					sval := fmt.Sprintf("%v", parsedMap[sk])
					displaySVal := sval
					if m.maskValues && (strings.Contains(strings.ToLower(sk), "pass") || strings.Contains(strings.ToLower(sk), "secret") || strings.Contains(strings.ToLower(sk), "token")) {
						displaySVal = "••••••••"
					}

					subLine := fmt.Sprintf(
						"      %s %s: %s",
						lipgloss.NewStyle().Foreground(ColorSecondary).Render("↳"),
						lipgloss.NewStyle().Foreground(ColorTextDim).Render(sk),
						lipgloss.NewStyle().Foreground(ColorText).Render(displaySVal),
					)
					rows = append(rows, subLine)
				}
			} else {
				displayVal := val
				if m.maskValues {
					displayVal = strings.Repeat("•", min(16, max(8, len(val))))
				}

				keyCol := StyleKeyCell.Render(fmt.Sprintf("%-24s", k))
				valCol := StyleValCell.Render(fmt.Sprintf("%-40s", displayVal))
				row := rowStyle.Render(fmt.Sprintf("%s%s   %s", cursor, keyCol, valCol))
				rows = append(rows, row)
			}
		}

		tableContent := strings.Join(rows, "\n")
		tableBox = StyleCard.Width(boxWidth).Render(tableContent)
		helpHints = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Render("[ m ] Mask • [ y ] Copy Val • [ g ] Gen Go File • [ G ] Copy Go Code • [ e ] Edit • [ d ] Del • [ Esc ] Back")
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		metaInfo,
		"",
		tableBox,
		"",
		helpHints,
	)
}
