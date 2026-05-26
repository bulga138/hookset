// Package shared_theme provides adaptive colour tokens and pre-built lipgloss
// styles for all hookset TUI views.
//
// Theme selection priority:
//  1. NO_COLOR=1 (or any non-empty value) → plain/no-colour mode
//  2. HOOKSET_THEME=dark|light|high-contrast|plain → explicit theme
//  3. Auto-detection via lipgloss HasDarkBackground (default: "auto")
package shared_theme

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme names
const (
	ThemeAuto          = "auto"
	ThemeDark          = "dark"
	ThemeLight         = "light"
	ThemeHighContrast  = "high-contrast"
	ThemePlain         = "plain"
)

// ActiveTheme is the resolved theme name, set during package init.
var ActiveTheme string

// plain is true when colour output is suppressed.
var plain bool

func init() {
	resolve()
}

// resolve determines the active theme from environment variables.
func resolve() {
	// NO_COLOR (https://no-color.org/) — any non-empty value disables colour.
	if os.Getenv("NO_COLOR") != "" {
		ActiveTheme = ThemePlain
		plain = true
		buildStyles()
		return
	}

	theme := strings.ToLower(strings.TrimSpace(os.Getenv("HOOKSET_THEME")))
	switch theme {
	case ThemeDark, ThemeLight, ThemeHighContrast, ThemePlain:
		ActiveTheme = theme
	default:
		ActiveTheme = ThemeAuto
	}

	if ActiveTheme == ThemePlain {
		plain = true
	}

	buildStyles()
}

// ── Adaptive colour tokens ────────────────────────────────────────────────────
//
// Each token has a dark-background value and a light-background value.
// lipgloss.AdaptiveColor resolves to the appropriate one at render time.

var (
	PrimaryColor   lipgloss.TerminalColor
	SecondaryColor lipgloss.TerminalColor
	WarningColor   lipgloss.TerminalColor
	ErrorColor     lipgloss.TerminalColor
	TextColor      lipgloss.TerminalColor
	DimColor       lipgloss.TerminalColor
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	BaseStyle    lipgloss.Style
	HeaderStyle  lipgloss.Style
	TitleStyle   lipgloss.Style
	WarningStyle lipgloss.Style
	ErrorStyle   lipgloss.Style

	CheckboxSelected   string
	CheckboxUnselected string
	CheckboxUninstall  string

	HookNameStyle lipgloss.Style
	EventStyle    lipgloss.Style
	MatchStyle    lipgloss.Style
	DimStyle      lipgloss.Style
	SelectStyle   lipgloss.Style
	BorderStyle   lipgloss.Style
)

// buildStyles (re-)creates all style vars based on the current theme.
// Called once during init and can be called again after a resize or theme change.
func buildStyles() {
	if plain {
		buildPlainStyles()
		return
	}

	switch ActiveTheme {
	case ThemeHighContrast:
		buildHighContrastStyles()
	case ThemeLight:
		buildLightStyles()
	case ThemeDark:
		buildDarkStyles()
	default: // auto
		buildAdaptiveStyles()
	}
}

// buildAdaptiveStyles uses lipgloss.AdaptiveColor which picks dark or light
// values depending on the terminal's background colour.
func buildAdaptiveStyles() {
	PrimaryColor = lipgloss.AdaptiveColor{Dark: "#00C9A7", Light: "#007A65"}
	SecondaryColor = lipgloss.AdaptiveColor{Dark: "#9D54FF", Light: "#6B21E8"}
	WarningColor = lipgloss.AdaptiveColor{Dark: "#FFB86C", Light: "#C05C00"}
	ErrorColor = lipgloss.AdaptiveColor{Dark: "#FF5555", Light: "#CC0000"}
	TextColor = lipgloss.AdaptiveColor{Dark: "#F8F8F2", Light: "#1A1A2E"}
	DimColor = lipgloss.AdaptiveColor{Dark: "#6272A4", Light: "#8890B0"}
	setStyles()
}

func buildDarkStyles() {
	PrimaryColor = lipgloss.Color("#00C9A7")
	SecondaryColor = lipgloss.Color("#9D54FF")
	WarningColor = lipgloss.Color("#FFB86C")
	ErrorColor = lipgloss.Color("#FF5555")
	TextColor = lipgloss.Color("#F8F8F2")
	DimColor = lipgloss.Color("#6272A4")
	setStyles()
}

func buildLightStyles() {
	PrimaryColor = lipgloss.Color("#007A65")
	SecondaryColor = lipgloss.Color("#6B21E8")
	WarningColor = lipgloss.Color("#C05C00")
	ErrorColor = lipgloss.Color("#CC0000")
	TextColor = lipgloss.Color("#1A1A2E")
	DimColor = lipgloss.Color("#8890B0")
	setStyles()
}

func buildHighContrastStyles() {
	PrimaryColor = lipgloss.Color("#00FF00")  // bright green
	SecondaryColor = lipgloss.Color("#FFFF00") // bright yellow
	WarningColor = lipgloss.Color("#FF8800")
	ErrorColor = lipgloss.Color("#FF0000")
	TextColor = lipgloss.Color("#FFFFFF")
	DimColor = lipgloss.Color("#AAAAAA")
	setStyles()
}

func buildPlainStyles() {
	// All colours are empty — lipgloss renders no escape sequences.
	PrimaryColor = lipgloss.NoColor{}
	SecondaryColor = lipgloss.NoColor{}
	WarningColor = lipgloss.NoColor{}
	ErrorColor = lipgloss.NoColor{}
	TextColor = lipgloss.NoColor{}
	DimColor = lipgloss.NoColor{}
	setStyles()
}

// setStyles builds all compound styles from the current colour tokens.
func setStyles() {
	BaseStyle = lipgloss.NewStyle().Foreground(TextColor)

	HeaderStyle = lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(!plain).
		PaddingBottom(1)

	TitleStyle = lipgloss.NewStyle().
		Bold(!plain).
		Foreground(TextColor).
		Background(PrimaryColor).
		Padding(0, 1)

	WarningStyle = lipgloss.NewStyle().
		Foreground(WarningColor).
		Bold(!plain).
		PaddingBottom(1)

	ErrorStyle = lipgloss.NewStyle().
		Foreground(ErrorColor).
		Bold(!plain)

	CheckboxSelected = lipgloss.NewStyle().Foreground(PrimaryColor).Render("[✓]")
	CheckboxUnselected = lipgloss.NewStyle().Foreground(DimColor).Render("[ ]")
	CheckboxUninstall = lipgloss.NewStyle().Foreground(WarningColor).Render("[*]")

	HookNameStyle = lipgloss.NewStyle().Foreground(SecondaryColor).Bold(!plain)
	EventStyle = lipgloss.NewStyle().Foreground(SecondaryColor)
	MatchStyle = lipgloss.NewStyle().Foreground(PrimaryColor)
	DimStyle = lipgloss.NewStyle().Foreground(DimColor)
	SelectStyle = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(!plain)

	BorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Padding(1, 2).
		Margin(0, 0, 1, 0).
		Width(80)
}
