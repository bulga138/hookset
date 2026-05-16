package init_view

import (
	"fmt"
	"strings"

	"github.com/bulga138/hookset/internal/manifest"
	"github.com/bulga138/hookset/internal/templates"
	tea "github.com/charmbracelet/bubbletea"
)

// HookEditor allows users to create or edit a custom hook.
type HookEditor struct {
	Name      string
	Event     string
	Match     string // comma-separated patterns
	Command   string
	Cursor    int
	Quitting  bool
	Confirmed bool
}

// InitialHookEditor creates a new hook editor model.
func InitialHookEditor() HookEditor {
	return HookEditor{
		Event: "pre-commit",
	}
}

// Init implements tea.Model
func (m HookEditor) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m HookEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.Quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < 3 {
				m.Cursor++
			}
		case "enter":
			if m.Cursor < 3 {
				m.Cursor++ // move to next field
			} else {
				// Last field - confirm
				if m.Name != "" && m.Command != "" {
					m.Confirmed = true
					return m, tea.Quit
				}
			}
		case "backspace":
			// Handle backspace based on cursor position
			switch m.Cursor {
			case 0:
				if len(m.Name) > 0 {
					m.Name = m.Name[:len(m.Name)-1]
				}
			case 1:
				// Cycle through events
				events := []string{"pre-commit", "pre-push", "commit-msg", "pre-merge-commit"}
				currentIdx := 0
				for i, e := range events {
					if m.Event == e {
						currentIdx = i
						break
					}
				}
				m.Event = events[(currentIdx+1)%len(events)]
			case 2:
				if len(m.Match) > 0 {
					m.Match = m.Match[:len(m.Match)-1]
				}
			case 3:
				if len(m.Command) > 0 {
					m.Command = m.Command[:len(m.Command)-1]
				}
			}
		default:
			// Type characters into current field
			ch := msg.String()
			if len(ch) == 1 {
				switch m.Cursor {
				case 0:
					m.Name += ch
				case 2:
					m.Match += ch
				case 3:
					m.Command += ch
				}
			}
		}
	}
	return m, nil
}

// View implements tea.Model
func (m HookEditor) View() string {
	var b strings.Builder

	b.WriteString("📝 Create custom hook:\n\n")
	b.WriteString("  Use arrow keys to move between fields, type to enter values\n")
	b.WriteString("  Press Enter to move to next field, Enter on Command to save\n\n")

	// Name field
	cursor := "  "
	if m.Cursor == 0 {
		cursor = "> "
	}
	fmt.Fprintf(&b, "%sName:    %s\n", cursor, m.Name)
	if m.Cursor == 0 {
		b.WriteString("          (hook identifier, e.g., eslint, format)\n")
	}

	// Event field
	cursor = "  "
	if m.Cursor == 1 {
		cursor = "> "
	}
	fmt.Fprintf(&b, "%sEvent:   %s (press Enter to cycle)\n", cursor, m.Event)

	// Match field
	cursor = "  "
	if m.Cursor == 2 {
		cursor = "> "
	}
	matchDisplay := m.Match
	if matchDisplay == "" {
		matchDisplay = "(all files)"
	}
	fmt.Fprintf(&b, "%sMatch:   %s\n", cursor, matchDisplay)
	if m.Cursor == 2 {
		b.WriteString("          (comma-separated globs, e.g., *.ts,*.js)\n")
	}

	// Command field
	cursor = "  "
	if m.Cursor == 3 {
		cursor = "> "
	}
	cmdDisplay := m.Command
	if cmdDisplay == "" {
		cmdDisplay = "(required)"
	}
	fmt.Fprintf(&b, "%sCommand: %s\n", cursor, cmdDisplay)
	if m.Cursor == 3 {
		b.WriteString("          (command to run, e.g., npx eslint --fix)\n")
	}

	b.WriteString("\nPress Enter on Command to save, Esc to cancel")

	return b.String()
}

// RunHookEditor launches the hook editor TUI.
// Returns nil if cancelled, otherwise returns the created manifest entry.
func RunHookEditor() (*manifest.Entry, error) {
	m := InitialHookEditor()
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	res := finalModel.(HookEditor)
	if res.Quitting || !res.Confirmed {
		return nil, nil
	}

	// Parse match patterns
	var match []string
	if res.Match != "" {
		match = strings.Split(res.Match, ",")
		for i := range match {
			match[i] = strings.TrimSpace(match[i])
		}
	}

	return &manifest.Entry{
		Name:    res.Name,
		Event:   res.Event,
		Match:   match,
		Command: res.Command,
	}, nil
}

// TemplatePicker allows users to select which templates and hooks to include.
type TemplatePicker struct {
	Templates []templates.Template
	Selected  map[int]bool // template index -> selected
	// HookSelected maps template index -> hook index -> selected
	HookSelected map[int]map[int]bool
	Cursor       int
	Mode         int // 0 = selecting templates, 1 = selecting hooks within template
	Quitting     bool
	Confirmed    bool
	CustomHooks  []manifest.Entry // user-created hooks
}

// InitialTemplatePicker creates a new template picker model.
func InitialTemplatePicker() TemplatePicker {
	all := templates.All()
	selected := make(map[int]bool, len(all))
	hookSelected := make(map[int]map[int]bool, len(all))

	// Default: select first one (typescript)
	if len(all) > 0 {
		selected[0] = true
		hookSelected[0] = make(map[int]bool)
		for i := range all[0].Hooks {
			hookSelected[0][i] = true // select all hooks in first template
		}
	}

	return TemplatePicker{
		Templates:    all,
		Selected:     selected,
		HookSelected: hookSelected,
		Mode:         0,
	}
}

// Init implements tea.Model
func (m TemplatePicker) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m TemplatePicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.Quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			max := len(m.Templates) + 1 // +1 for "Create custom hook" option
			if m.Mode == 1 {
				max = len(m.Templates[m.Cursor].Hooks)
			}
			if m.Cursor < max-1 {
				m.Cursor++
			}
		case " ":
			if m.Mode == 0 {
				// Toggle template selection (not for custom hook option)
				if m.Cursor < len(m.Templates) {
					m.Selected[m.Cursor] = !m.Selected[m.Cursor]
					if m.Selected[m.Cursor] {
						// Initialize hook selection for this template
						if m.HookSelected[m.Cursor] == nil {
							m.HookSelected[m.Cursor] = make(map[int]bool)
						}
						// Select all hooks by default
						for i := range m.Templates[m.Cursor].Hooks {
							m.HookSelected[m.Cursor][i] = true
						}
					}
				}
			} else {
				// Toggle hook selection within current template
				if m.HookSelected[m.Cursor] == nil {
					m.HookSelected[m.Cursor] = make(map[int]bool)
				}
				m.HookSelected[m.Cursor][m.Cursor] = !m.HookSelected[m.Cursor][m.Cursor]
			}
		case "c":
			if m.Mode == 0 && m.Cursor == len(m.Templates) {
				// "Create custom hook" option selected
				entry, err := RunHookEditor()
				if err != nil {
					// Handle error - stay in current view
					return m, nil
				}
				if entry != nil {
					m.CustomHooks = append(m.CustomHooks, *entry)
				}
			}
		case "a":
			if m.Mode == 0 {
				// Toggle all: select all if any not selected, else deselect all
				allSelected := true
				for i := range m.Templates {
					if !m.Selected[i] {
						allSelected = false
						break
					}
				}
				for i := range m.Templates {
					m.Selected[i] = !allSelected
					if m.Selected[i] {
						if m.HookSelected[i] == nil {
							m.HookSelected[i] = make(map[int]bool)
						}
						for j := range m.Templates[i].Hooks {
							m.HookSelected[i][j] = true
						}
					}
				}
			}
		case "enter":
			if m.Mode == 0 {
				// Check if any template is selected or custom hooks exist
				hasSelection := false
				for _, selected := range m.Selected {
					if selected {
						hasSelection = true
						break
					}
				}
				if len(m.CustomHooks) > 0 {
					hasSelection = true
				}

				if hasSelection {
					m.Mode = 1 // switch to hook selection mode
					// Find first selected template
					m.Cursor = 0
					for m.Cursor < len(m.Templates) && !m.Selected[m.Cursor] {
						m.Cursor++
					}
					// If no template selected but custom hooks exist, stay at 0
					if m.Cursor >= len(m.Templates) {
						m.Cursor = 0
					}
				}
			} else {
				m.Confirmed = true
				return m, tea.Quit
			}
		case "backspace":
			if m.Mode == 1 {
				m.Mode = 0 // go back to template selection
				m.Cursor = 0
			}
		}
	}
	return m, nil
}

// View implements tea.Model
func (m TemplatePicker) View() string {
	var b strings.Builder

	if m.Mode == 0 {
		b.WriteString("📦 Select templates for your project:\n\n")
		b.WriteString("  Use space to toggle, a to select all, c to create custom hook\n")
		b.WriteString("  Enter to continue\n\n")

		for i, t := range m.Templates {
			cursor := "  "
			checkbox := "[ ]"
			if m.Cursor == i {
				cursor = "> "
			}
			if m.Selected[i] {
				checkbox = "[✓]"
			}

			// Count selected hooks
			hookCount := 0
			if hs, ok := m.HookSelected[i]; ok {
				for _, s := range hs {
					if s {
						hookCount++
					}
				}
			}

			hookInfo := ""
			if m.Selected[i] && hookCount > 0 {
				hookInfo = fmt.Sprintf(" (%d hooks)", hookCount)
			}

			fmt.Fprintf(&b, "%s%s %s - %s%s\n", cursor, checkbox, t.Name, t.Description, hookInfo)
		}

		// "Create custom hook" option
		cursor := "  "
		if m.Cursor == len(m.Templates) {
			cursor = "> "
		}
		customCount := len(m.CustomHooks)
		customInfo := ""
		if customCount > 0 {
			customInfo = fmt.Sprintf(" (%d custom hook(s) created)", customCount)
		}
		fmt.Fprintf(&b, "%s+ Create custom hook%s\n", cursor, customInfo)
	} else {
		// Hook selection mode
		t := m.Templates[m.Cursor]
		fmt.Fprintf(&b, "📦 Select hooks for %s:\n\n", t.Name)
		b.WriteString("  Use space to toggle, backspace to go back, enter to continue\n\n")

		for i, h := range t.Hooks {
			cursor := "  "
			checkbox := "[ ]"
			if m.Cursor == i {
				cursor = "> "
			}
			if m.HookSelected[m.Cursor] != nil && m.HookSelected[m.Cursor][i] {
				checkbox = "[✓]"
			}

			fmt.Fprintf(&b, "%s%s %s (%s)\n", cursor, checkbox, h.Name, h.Event)
		}
	}

	b.WriteString("\n")

	if m.Mode == 0 {
		b.WriteString("Selected: ")
		var selectedNames []string
		for i, selected := range m.Selected {
			if selected {
				selectedNames = append(selectedNames, m.Templates[i].Name)
			}
		}
		if len(selectedNames) == 0 && len(m.CustomHooks) == 0 {
			b.WriteString("(none)")
		} else {
			if len(selectedNames) > 0 {
				b.WriteString(strings.Join(selectedNames, ", "))
			}
			if len(m.CustomHooks) > 0 {
				if len(selectedNames) > 0 {
					b.WriteString(", ")
				}
				fmt.Fprintf(&b, "%d custom", len(m.CustomHooks))
			}
		}
	} else {
		// Show selected hooks count
		count := 0
		if hs, ok := m.HookSelected[m.Cursor]; ok {
			for _, s := range hs {
				if s {
					count++
				}
			}
		}
		fmt.Fprintf(&b, "Selected: %d hook(s) in %s", count, m.Templates[m.Cursor].Name)
	}

	return b.String()
}

// RunTemplatePicker launches the template selection TUI.
// Returns nil if cancelled, otherwise returns the selected manifest entries.
func RunTemplatePicker() ([]manifest.Entry, error) {
	m := InitialTemplatePicker()
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	res := finalModel.(TemplatePicker)
	if res.Quitting || !res.Confirmed {
		return nil, nil
	}

	// Collect all selected entries
	var entries []manifest.Entry

	// Add custom hooks first
	entries = append(entries, res.CustomHooks...)

	// Add template hooks
	for ti, templateSelected := range res.Selected {
		if !templateSelected {
			continue
		}
		template := res.Templates[ti]
		hookSel := res.HookSelected[ti]
		for hi, h := range template.Hooks {
			if hookSel != nil && hookSel[hi] {
				entries = append(entries, h)
			}
		}
	}

	return entries, nil
}
