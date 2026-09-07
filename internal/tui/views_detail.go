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

	case "y", "c":
		// Copy selected value to clipboard
		if m.selectedSecret != nil && len(m.sortedKeys) > 0 && m.detailRowIndex < len(m.sortedKeys) {
			key := m.sortedKeys[m.detailRowIndex]
			val := m.selectedSecret.Data[key]
			return m.copyToClipboard(val)
		}
		return m, nil

	case "g":
		// Generate external Go retrieval file
		if m.selectedSecret != nil {
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

	versionBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorText).
		Background(ColorPrimaryDark).
		Padding(0, 1).
		Render(fmt.Sprintf("v%d", m.selectedSecret.Version))

	header := lipgloss.JoinHorizontal(lipgloss.Center, title, pathBadge, " ", versionBadge)

	// Metadata
	metaInfo := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Render(fmt.Sprintf("Created: %s", m.selectedSecret.CreatedTime.Format("2006-01-02 15:04:05 MST")))

	// Key-Value Table
	keys := m.sortedKeys
	if len(keys) == 0 {
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
		var parsedMap map[string]interface{}
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
	boxWidth := 76
	if m.width > 80 {
		boxWidth = min(m.width-6, 104)
	}

	tableBox := StyleCard.Width(boxWidth).Render(tableContent)

	helpHints := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Render("[ m ] Mask • [ y ] Copy Val • [ g ] Gen Go File • [ G ] Copy Go Code • [ e ] Edit • [ d ] Del • [ Esc ] Back")

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

