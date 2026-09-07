// views_editor.go provides the multi-row form and raw JSON mode for creating
// and editing secrets, as well as launching the JSON Structure Builder.

package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleEditorKeys routes keyboard input between path, dynamic key-value rows, and action shortcuts.
func (m Model) handleEditorKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Toggle JSON mode
	if msg.String() == "ctrl+j" {
		m.isRawJSONMode = !m.isRawJSONMode
		if m.isRawJSONMode {
			// Convert current rows to JSON string
			data := m.collectEditorData()
			b, _ := json.MarshalIndent(data, "", "  ")
			m.rawJSONContent = string(b)
			m.toast = &Toast{Message: "Switched to JSON mode. Press [Ctrl+J] to switch back.", Type: ToastInfo}
		} else {
			// Parse JSON string back to rows
			var parsed map[string]string
			if err := json.Unmarshal([]byte(m.rawJSONContent), &parsed); err == nil && len(parsed) > 0 {
				m.initEditorRows(len(parsed))
				i := 0
				for k, v := range parsed {
					m.editKeyInputs[i].SetValue(k)
					m.editValInputs[i].SetValue(v)
					i++
				}
			}
			m.toast = &Toast{Message: "Switched to Key-Value Form mode.", Type: ToastInfo}
		}
		return m, clearToastCmd()
	}

	switch msg.String() {
	case "esc":
		if m.isEditingExisting {
			m.state = StateDetail
		} else {
			m.state = StateList
		}
		return m, nil

	case "ctrl+s":
		return m.commitEditorSave()

	case "ctrl+t":
		// Open Struct Builder for a new key
		m.openStructBuilder(-1, "", PresetRabbitMQ)
		return m, nil

	case "ctrl+e":
		// Open Struct Builder to edit current row
		rowIdx := (m.activeField - 1) / 2
		if rowIdx >= 0 && rowIdx < len(m.editKeyInputs) {
			key := m.editKeyInputs[rowIdx].Value()
			m.openStructBuilder(rowIdx, key, PresetCustom)
			return m, nil
		}
		m.openStructBuilder(-1, "", PresetRabbitMQ)
		return m, nil

	case "ctrl+n", "ctrl+a":
		// Add new key-value row
		m.appendEditorRow()
		m.toast = &Toast{Message: "Added new key-value row", Type: ToastInfo}
		return m, clearToastCmd()

	case "ctrl+d":
		// Delete row if we have more than 1
		if len(m.editKeyInputs) > 1 {
			m.removeCurrentEditorRow()
			m.toast = &Toast{Message: "Removed key-value row", Type: ToastInfo}
			return m, clearToastCmd()
		}
		return m, nil

	case "tab":
		m.cycleEditorFocus(1)
		return m, nil

	case "shift+tab":
		m.cycleEditorFocus(-1)
		return m, nil

	case "enter":
		// If on the last value input or pressing enter on save button
		if m.activeField == 1+len(m.editKeyInputs)*2 {
			return m.commitEditorSave()
		}
		// Move to next field
		m.cycleEditorFocus(1)
		return m, nil
	}

	// Route typing to active field
	var cmd tea.Cmd
	if m.activeField == 0 {
		m.editPathInput, cmd = m.editPathInput.Update(msg)
		return m, cmd
	}

	rowIdx := (m.activeField - 1) / 2
	isVal := (m.activeField-1)%2 == 1

	if rowIdx < len(m.editKeyInputs) {
		if isVal {
			m.editValInputs[rowIdx], cmd = m.editValInputs[rowIdx].Update(msg)
		} else {
			m.editKeyInputs[rowIdx], cmd = m.editKeyInputs[rowIdx].Update(msg)
		}
	}

	return m, cmd
}

func isJSONString(s string) bool {
	trimmed := strings.TrimSpace(s)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return false
	}
	return json.Valid([]byte(trimmed))
}

func (m *Model) appendEditorRow() {
	count := len(m.editKeyInputs) + 1
	newKeys := make([]textinput.Model, count)
	newVals := make([]textinput.Model, count)
	copy(newKeys, m.editKeyInputs)
	copy(newVals, m.editValInputs)

	ki := textinput.New()
	ki.Placeholder = fmt.Sprintf("Key #%d", count)
	ki.Prompt = "  Key: "
	ki.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary)

	vi := textinput.New()
	vi.Placeholder = fmt.Sprintf("Value #%d", count)
	vi.Prompt = "Value: "
	vi.PromptStyle = lipgloss.NewStyle().Foreground(ColorSecondary)

	newKeys[count-1] = ki
	newVals[count-1] = vi

	m.editKeyInputs = newKeys
	m.editValInputs = newVals
}

func (m *Model) removeCurrentEditorRow() {
	if len(m.editKeyInputs) <= 1 {
		return
	}
	rowIdx := max(0, (m.activeField-1)/2)
	if rowIdx >= len(m.editKeyInputs) {
		rowIdx = len(m.editKeyInputs) - 1
	}

	var newKeys []textinput.Model
	var newVals []textinput.Model
	for i := range m.editKeyInputs {
		if i != rowIdx {
			newKeys = append(newKeys, m.editKeyInputs[i])
			newVals = append(newVals, m.editValInputs[i])
		}
	}
	m.editKeyInputs = newKeys
	m.editValInputs = newVals
	m.activeField = min(m.activeField, len(m.editKeyInputs)*2)
	m.focusActiveField()
}

func (m *Model) cycleEditorFocus(direction int) {
	// Total fields:
	// 0: Path
	// 1..2N: Key/Val pairs (N = len(m.editKeyInputs))
	// 2N+1: Save button
	totalFields := 1 + len(m.editKeyInputs)*2 + 1
	m.activeField = (m.activeField + direction + totalFields) % totalFields
	m.focusActiveField()
}

func (m *Model) focusActiveField() {
	m.editPathInput.Blur()
	for i := range m.editKeyInputs {
		m.editKeyInputs[i].Blur()
		m.editValInputs[i].Blur()
	}

	if m.activeField == 0 {
		m.editPathInput.Focus()
	} else if m.activeField <= len(m.editKeyInputs)*2 {
		rowIdx := (m.activeField - 1) / 2
		isVal := (m.activeField-1)%2 == 1
		if isVal {
			m.editValInputs[rowIdx].Focus()
		} else {
			m.editKeyInputs[rowIdx].Focus()
		}
	}
}

func (m Model) collectEditorData() map[string]string {
	data := make(map[string]string, len(m.editKeyInputs))
	for i := range m.editKeyInputs {
		k := strings.TrimSpace(m.editKeyInputs[i].Value())
		v := m.editValInputs[i].Value()
		if k != "" {
			data[k] = v
		}
	}
	return data
}

func (m Model) commitEditorSave() (tea.Model, tea.Cmd) {
	path := strings.Trim(m.editPathInput.Value(), "/")
	if path == "" {
		m.toast = &Toast{Message: "Secret path cannot be empty", Type: ToastError}
		return m, clearToastCmd()
	}

	data := m.collectEditorData()
	if len(data) == 0 {
		m.toast = &Toast{Message: "Must provide at least one key-value pair", Type: ToastError}
		return m, clearToastCmd()
	}

	m.toast = &Toast{Message: fmt.Sprintf("Writing secret '%s' to Vault...", path), Type: ToastInfo}
	return m, m.saveSecretCmd(path, data)
}

func (m Model) renderEditorView() string {
	titleText := "✨ CREATE NEW SECRET"
	if m.isEditingExisting {
		titleText = "✏️  EDIT SECRET"
	}
	title := StyleTitle.Render(titleText)
	mountInfo := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Render(fmt.Sprintf("Mount: %s/", m.client.Mount()))

	header := lipgloss.JoinHorizontal(lipgloss.Center, title, mountInfo)

	// Path Box
	pathBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Render(m.editPathInput.View())

	// Dynamic Rows
	rowViews := make([]string, 0, len(m.editKeyInputs))
	for i := range m.editKeyInputs {
		kView := m.editKeyInputs[i].View()
		vView := m.editValInputs[i].View()

		rowBorder := ColorBorder
		if m.activeField == 1+i*2 || m.activeField == 1+i*2+1 {
			rowBorder = ColorSecondary
		}

		var jsonTag string
		if isJSONString(m.editValInputs[i].Value()) {
			jsonTag = " " + BadgeJSON.Render("JSON")
		}

		rowContent := lipgloss.JoinHorizontal(lipgloss.Center, kView, jsonTag, "   ", vView)
		boxedRow := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(rowBorder).
			Padding(0, 1).
			Render(rowContent)

		rowViews = append(rowViews, boxedRow)
	}

	// Action Buttons
	saveStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#000000")).
		Background(ColorSuccess).
		Padding(0, 2)

	if m.activeField == 1+len(m.editKeyInputs)*2 {
		saveStyle = saveStyle.Border(lipgloss.NormalBorder()).BorderForeground(ColorHighlight)
	}

	saveBtn := saveStyle.Render("✔ SAVE (Ctrl+S)")

	cancelBtn := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Render("[ Esc ] Cancel")

	btnRow := lipgloss.JoinHorizontal(lipgloss.Center, saveBtn, "   ", cancelBtn)

	// Shortcuts Bar
	shortcuts := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render("[ Tab ] Next • [ Ctrl+T ] Struct (RabbitMQ) • [ Ctrl+E ] Edit Struct • [ Ctrl+N ] Add Row • [ Ctrl+J ] JSON")

	boxWidth := 76
	if m.width > 80 {
		boxWidth = min(m.width-6, 104)
	}

	cardContent := fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s",
		pathBox,
		strings.Join(rowViews, "\n"),
		btnRow,
		shortcuts,
	)

	editorCard := StyleCard.Width(boxWidth).Render(cardContent)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		editorCard,
	)
}
