package shared_theme

import (
	"charm.land/lipgloss/v2"
)

var (
	// Colors
	PrimaryColor   = lipgloss.Color("#00C9A7") // Teal
	SecondaryColor = lipgloss.Color("#9D54FF") // Purple
	WarningColor   = lipgloss.Color("#FFB86C") // Orange for dry-run
	ErrorColor     = lipgloss.Color("#FF5555") // Red
	TextColor      = lipgloss.Color("#F8F8F2") // Light gray
	DimColor       = lipgloss.Color("#6272A4") // Dim gray

	// Base styles
	BaseStyle = lipgloss.NewStyle().
			Foreground(TextColor).
			Background(lipgloss.Color("#282A36")) // Dark background

	// Header style
	HeaderStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			Padding(0, 0, 1, 0)

	// Title style (for main title bar)
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(TextColor).
			Background(PrimaryColor).
			Padding(0, 1)

	// Warning style for dry-run banner
	WarningStyle = lipgloss.NewStyle().
			Foreground(WarningColor).
			Bold(true).
			Padding(0, 0, 1, 0)

	// Error style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	// Selected checkbox
	CheckboxSelected = lipgloss.NewStyle().
				Foreground(PrimaryColor).
				Render("[✓]")

	// Unselected checkbox
	CheckboxUnselected = lipgloss.NewStyle().
				Foreground(DimColor).
				Render("[ ]")

	// Uninstall checkbox
	CheckboxUninstall = lipgloss.NewStyle().
				Foreground(WarningColor).
				Render("[*]")

	// Hook name style
	HookNameStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	// Event style
	EventStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BD93F9")) // Light purple

	// Match pattern style
	MatchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B")) // Green

	// Dim style for secondary text
	DimStyle = lipgloss.NewStyle().
			Foreground(DimColor)

	// Select style for cursor and highlighted items
	SelectStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true)

	// Border style
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(1, 2).
			Margin(0, 0, 1, 0).
			Width(80)
)

func init() {
	// Enable true color if supported (optional)
	// termenv.EnableColor()
}
