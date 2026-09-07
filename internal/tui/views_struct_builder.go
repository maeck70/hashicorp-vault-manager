// views_struct_builder.go implements the interactive sub-form wizard for
// creating and editing structured JSON secrets (e.g. RabbitMQ, PostgreSQL, Redis).

package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleStructBuilderKeys handles preset cycling, sub-field editing, and applying JSON strings.
func (m Model) handleStructBuilderKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = StateEditor
		return m, nil

	case "ctrl+p":
		// Cycle presets
		nextPreset := (m.structPreset + 1) % 4
		m.applyStructPreset(nextPreset)
		m.toast = &Toast{Message: fmt.Sprintf("Preset switched to %s", m.structPresetName(nextPreset)), Type: ToastInfo}
		return m, clearToastCmd()

	case "ctrl+s":
		return m.commitStructApply()

	case "ctrl+n", "ctrl+a":
		m.appendStructRow()
		m.toast = &Toast{Message: "Added sub-field", Type: ToastInfo}
		return m, clearToastCmd()

	case "ctrl+d":
		if len(m.structKeyInputs) > 1 {
			m.removeCurrentStructRow()
			m.toast = &Toast{Message: "Removed sub-field", Type: ToastInfo}
			return m, clearToastCmd()
		}
		return m, nil

	case "tab":
		m.cycleStructFocus(1)
		return m, nil

	case "shift+tab":
		m.cycleStructFocus(-1)
		return m, nil

	case "enter":
		saveButtonIndex := 1 + len(m.structKeyInputs)*2
		if m.structActiveField == saveButtonIndex {
			return m.commitStructApply()
		}
		m.cycleStructFocus(1)
		return m, nil
	}

	// Update active text input
	var cmd tea.Cmd
	if m.structActiveField == 0 {
		m.structKeyInput, cmd = m.structKeyInput.Update(msg)
		return m, cmd
	}

	rowIdx := (m.structActiveField - 1) / 2
	isVal := (m.structActiveField-1)%2 == 1

	if rowIdx < len(m.structKeyInputs) {
		if isVal {
			m.structValInputs[rowIdx], cmd = m.structValInputs[rowIdx].Update(msg)
		} else {
			m.structKeyInputs[rowIdx], cmd = m.structKeyInputs[rowIdx].Update(msg)
		}
	}

	return m, cmd
}

func (m *Model) cycleStructFocus(direction int) {
	// 0: Key Name
	// 1..2M: sub-field key/val pairs
	// 2M+1: Save Button
	totalFields := 1 + len(m.structKeyInputs)*2 + 1
	m.structActiveField = (m.structActiveField + direction + totalFields) % totalFields
	m.focusStructActiveField()
}

func (m *Model) focusStructActiveField() {
	m.structKeyInput.Blur()
	for i := range m.structKeyInputs {
		m.structKeyInputs[i].Blur()
		m.structValInputs[i].Blur()
	}

	if m.structActiveField == 0 {
		m.structKeyInput.Focus()
	} else if m.structActiveField <= len(m.structKeyInputs)*2 {
		rowIdx := (m.structActiveField - 1) / 2
		isVal := (m.structActiveField-1)%2 == 1
		if isVal {
			m.structValInputs[rowIdx].Focus()
		} else {
			m.structKeyInputs[rowIdx].Focus()
		}
	}
}

func (m Model) commitStructApply() (tea.Model, tea.Cmd) {
	keyName := strings.TrimSpace(m.structKeyInput.Value())
	if keyName == "" {
		m.toast = &Toast{Message: "JSON key name cannot be empty", Type: ToastError}
		return m, clearToastCmd()
	}

	jsonStr := m.buildStructJSONString()

	if m.structTargetRow >= 0 && m.structTargetRow < len(m.editKeyInputs) {
		m.editKeyInputs[m.structTargetRow].SetValue(keyName)
		m.editValInputs[m.structTargetRow].SetValue(jsonStr)
	} else {
		// If the editor only has one row and it's empty, use it; otherwise append
		if len(m.editKeyInputs) == 1 && m.editKeyInputs[0].Value() == "" && m.editValInputs[0].Value() == "" {
			m.editKeyInputs[0].SetValue(keyName)
			m.editValInputs[0].SetValue(jsonStr)
		} else {
			m.appendEditorRow()
			lastIdx := len(m.editKeyInputs) - 1
			m.editKeyInputs[lastIdx].SetValue(keyName)
			m.editValInputs[lastIdx].SetValue(jsonStr)
		}
	}

	m.toast = &Toast{Message: fmt.Sprintf("✓ JSON key '%s' added to editor!", keyName), Type: ToastSuccess}
	m.state = StateEditor
	return m, clearToastCmd()
}

func (m Model) structPresetName(p StructPreset) string {
	switch p {
	case PresetRabbitMQ:
		return "RabbitMQ"
	case PresetPostgres:
		return "PostgreSQL"
	case PresetRedis:
		return "Redis"
	case PresetCustom:
		return "Custom"
	}
	return "Preset"
}

func (m Model) renderStructBuilderView() string {
	title := StyleTitle.Render("🧩 JSON STRUCTURE BUILDER")
	helpPill := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Render("Define nested fields • captured and saved as a JSON string")

	header := lipgloss.JoinHorizontal(lipgloss.Center, title, helpPill)

	// Preset Bar
	presets := []struct {
		p    StructPreset
		name string
	}{
		{PresetRabbitMQ, "RabbitMQ"},
		{PresetPostgres, "PostgreSQL"},
		{PresetRedis, "Redis"},
		{PresetCustom, "Custom"},
	}

	var presetTabs []string
	for _, ps := range presets {
		if ps.p == m.structPreset {
			presetTabs = append(presetTabs, PresetActive.Render("● "+ps.name))
		} else {
			presetTabs = append(presetTabs, PresetInactive.Render(ps.name))
		}
	}
	presetBar := lipgloss.JoinHorizontal(lipgloss.Center, strings.Join(presetTabs, "  "), "    ", lipgloss.NewStyle().Foreground(ColorMuted).Render("[Ctrl+P] Cycle Preset"))

	// Key input box
	keyBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Render(m.structKeyInput.View())

	// Sub-field Rows
	var subRows []string
	for i := range m.structKeyInputs {
		kView := m.structKeyInputs[i].View()
		vView := m.structValInputs[i].View()

		rowBorder := ColorBorder
		if m.structActiveField == 1+i*2 || m.structActiveField == 1+i*2+1 {
			rowBorder = ColorSecondary
		}

		rowContent := lipgloss.JoinHorizontal(lipgloss.Center, kView, "   ", vView)
		boxedRow := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(rowBorder).
			Padding(0, 1).
			Render(rowContent)

		subRows = append(subRows, boxedRow)
	}

	// Live JSON Preview Card
	data := m.collectStructData()
	prettyJSON, _ := json.MarshalIndent(data, "", "  ")
	previewCard := StyleJSONPreview.Render(
		fmt.Sprintf(
			"%s\n%s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("Live JSON String Preview:"),
			lipgloss.NewStyle().Foreground(ColorHighlight).Render(string(prettyJSON)),
		),
	)

	// Action Buttons
	saveStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#000000")).
		Background(ColorSuccess).
		Padding(0, 2)

	saveBtnIndex := 1 + len(m.structKeyInputs)*2
	if m.structActiveField == saveBtnIndex {
		saveStyle = saveStyle.Border(lipgloss.NormalBorder()).BorderForeground(ColorHighlight)
	}

	saveBtn := saveStyle.Render("✔ APPLY TO SECRET (Ctrl+S)")

	cancelBtn := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Render("[ Esc ] Cancel")

	btnRow := lipgloss.JoinHorizontal(lipgloss.Center, saveBtn, "   ", cancelBtn)

	shortcuts := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render("[ Tab ] Next field • [ Ctrl+P ] Preset • [ Ctrl+N ] Add Field • [ Ctrl+D ] Del Field • [ Esc ] Cancel")

	boxWidth := 76
	if m.width > 80 {
		boxWidth = min(m.width-6, 104)
	}

	mainCard := StyleCard.Width(boxWidth).Render(
		fmt.Sprintf(
			"%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s",
			presetBar,
			keyBox,
			strings.Join(subRows, "\n"),
			previewCard,
			btnRow,
			shortcuts,
		),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		mainCard,
	)
}
