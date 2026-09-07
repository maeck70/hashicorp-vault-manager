// Package tui provides the terminal user interface for HashiCorp Vault management,
// built using the Charm ecosystem (Bubble Tea, Bubbles, and Lip Gloss).
package tui

import "github.com/charmbracelet/lipgloss"

// Palette holds the vibrant, modern color scheme for the Vault TUI.
var (
	// Colors
	ColorPrimary   = lipgloss.Color("#b388ff") // Neon Purple / Lilac
	ColorPrimaryDark = lipgloss.Color("#7c4dff")
	ColorSecondary = lipgloss.Color("#00f0ff") // Cyber Cyan
	ColorSuccess   = lipgloss.Color("#00e676") // Vibrant Green
	ColorWarning   = lipgloss.Color("#ffb300") // Amber
	ColorDanger    = lipgloss.Color("#ff1744") // Coral Red
	ColorMuted     = lipgloss.Color("#6b7280") // Subtle Slate
	ColorText      = lipgloss.Color("#f8fafc") // Bright White
	ColorTextDim   = lipgloss.Color("#94a3b8") // Dim Silver
	ColorBgDark    = lipgloss.Color("#0f172a") // Deep Navy Slate
	ColorBgCard    = lipgloss.Color("#1e293b") // Surface Navy
	ColorHighlight = lipgloss.Color("#ffd600") // Bright Yellow
	ColorBorder    = lipgloss.Color("#334155") // Subtle Border Slate

	// Header Styles
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(ColorPrimaryDark).
			Padding(0, 1).
			MarginRight(1)

	StyleSubtitle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	// Status Badges
	BadgeOnline = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(ColorSuccess).
			Padding(0, 1)

	BadgeSealed = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(ColorWarning).
			Padding(0, 1)

	BadgeUninit = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(ColorDanger).
			Padding(0, 1)

	BadgeInfo = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color("#3b82f6")).
			Padding(0, 1)

	// Panels & Boxes
	StyleCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	StyleCardDim = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	StyleDialog = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorWarning).
			Padding(1, 3).
			Align(lipgloss.Center)

	// List & Item Styles
	StyleSelectedItem = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary).
				Background(ColorBgCard).
				PaddingLeft(1)

	StyleNormalItem = lipgloss.NewStyle().
			Foreground(ColorText).
			PaddingLeft(2)

	// Key Value Table Styles
	StyleKeyHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Underline(true)

	StyleValHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			Underline(true)

	StyleKeyCell = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	StyleValCell = lipgloss.NewStyle().
			Foreground(ColorText)

	StyleSelectedRow = lipgloss.NewStyle().
				Background(ColorBgCard).
				Bold(true)

	// Help & Keybinding Badges
	StyleKeyPill = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(ColorSecondary).
			Padding(0, 1)

	StyleKeyDesc = lipgloss.NewStyle().
			Foreground(ColorTextDim)

	// Toast Messages
	StyleToastSuccess = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	StyleToastError = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	StyleToastInfo = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	// JSON & Preset Badges
	BadgeJSON = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(ColorSecondary).
			Padding(0, 1)

	PresetActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(ColorPrimary).
			Padding(0, 1)

	PresetInactive = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Background(ColorBgCard).
			Padding(0, 1)

	StyleJSONPreview = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorSecondary).
				Background(ColorBgDark).
				Padding(0, 1)
)

