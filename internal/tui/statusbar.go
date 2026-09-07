// statusbar.go provides the bottom status bar showing Vault connectivity,
// seal status, current mount, toast alerts, and contextual keyboard shortcut hints.

package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// View renders the complete TUI interface based on current state.
func (m Model) View() string {
	var body string

	switch m.state {
	case StateChecking:
		body = m.renderCheckingView()
	case StateInit:
		body = m.renderInitView()
	case StateUnseal:
		body = m.renderUnsealView()
	case StateList:
		body = m.renderListView()
	case StateDetail:
		body = m.renderDetailView()
	case StateEditor:
		body = m.renderEditorView()
	case StateConfirm:
		body = m.renderConfirmDialog()
	case StateStructBuilder:
		body = m.renderStructBuilderView()
	}

	statusBar := m.renderStatusBar()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		body,
		"",
		statusBar,
	)
}

func (m Model) renderCheckingView() string {
	title := StyleTitle.Render("⚡ HASHICORP VAULT TUI")
	desc := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Render(fmt.Sprintf("Connecting to Vault at %s...", m.client.Address()))

	box := StyleCard.Width(64).Render(fmt.Sprintf("%s\n\n%s", title, desc))
	return box
}

func (m Model) renderStatusBar() string {
	width := 76
	if m.width > 80 {
		width = min(m.width-4, 104)
	}

	// Address pill
	addrPill := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Background(ColorBgCard).
		Padding(0, 1).
		Render("🌐 " + m.client.Address())

	// Status badge
	var statusBadge string
	if m.health == nil {
		statusBadge = BadgeInfo.Render("CHECKING")
	} else if !m.health.Initialized {
		statusBadge = BadgeUninit.Render("UNINITIALIZED")
	} else if m.health.Sealed {
		statusBadge = BadgeSealed.Render("SEALED")
	} else {
		statusBadge = BadgeOnline.Render("UNSEALED")
	}

	// Mount badge
	mountBadge := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Background(ColorBgCard).
		Padding(0, 1).
		Render("📁 " + m.client.Mount())

	// Toast message
	var toastStr string
	if m.toast != nil {
		switch m.toast.Type {
		case ToastSuccess:
			toastStr = StyleToastSuccess.Render("  " + m.toast.Message)
		case ToastWarning:
			toastStr = StyleToastWarning.Render("  " + m.toast.Message)
		case ToastError:
			toastStr = StyleToastError.Render("  " + m.toast.Message)
		default:
			toastStr = StyleToastInfo.Render("  " + m.toast.Message)
		}
	}

	left := lipgloss.JoinHorizontal(lipgloss.Center, addrPill, " ", statusBadge, " ", mountBadge, toastStr)

	// Contextual hotkeys on right
	var hotkeys string
	switch m.state {
	case StateList:
		hotkeys = "[n] New  [/] Find  [r] Refresh  [q] Quit"
	case StateDetail:
		hotkeys = "[m] Mask  [y] Copy  [g] GenGoFile  [G] CopyGoCode  [e] Edit  [Esc] Back"
	case StateEditor:
		hotkeys = "[Ctrl+S] Save  [Ctrl+T] Struct  [Ctrl+E] EditStruct  [Ctrl+N] Add  [Esc] Cancel"
	case StateStructBuilder:
		hotkeys = "[Tab] Next  [Ctrl+P] Preset  [Ctrl+N] Add  [Ctrl+S] Apply  [Esc] Cancel"
	case StateInit:
		hotkeys = "[Enter] Initialize  [Tab] Switch  [q] Quit"
	case StateUnseal:
		hotkeys = "[Enter] Unseal  [q] Quit"
	default:
		hotkeys = "[Ctrl+C] Quit"
	}

	right := lipgloss.NewStyle().Foreground(ColorMuted).Render(hotkeys)

	content := lipgloss.JoinHorizontal(
		lipgloss.Center,
		left,
		"    ",
		right,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(ColorBorder).
		Width(width).
		Padding(0, 1).
		Render(content)
}
