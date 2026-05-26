package init_view

import (
	"fmt"
	"strings"

	"github.com/bulga138/hookset/internal/manifest"
	"github.com/bulga138/hookset/internal/templates"
	"github.com/bulga138/hookset/internal/ui/shared_theme"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ── HookEditor (C2: uses bubbles/textinput) ───────────────────────────────────

// hookEditorField indices
const (
	fieldName    = 0
	fieldEvent   = 1
	fieldMatch   = 2
	fieldCommand = 3
	fieldCount   = 4
)

var knownEvents = []string{
	"pre-commit", "pre-push", "commit-msg", "prepare-commit-msg",
	"pre-rebase", "post-checkout", "post-merge", "post-rewrite",
	"applypatch-msg",
}

// HookEditor is a bubbletea model for creating or editing a single hook entry.
// It uses charmbracelet/bubbles textinput for all text fields (C2).
type HookEditor struct {
	inputs    [fieldCount]textinput.Model
	eventIdx  int // index into knownEvents for the event field
	focusIdx  int
	Quitting  bool
	Confirmed bool
	// For C5 (edit mode): if editing, Original holds the original values.
	Original *manifest.Entry
}

// InitialHookEditor creates a new empty hook editor.
func InitialHookEditor() HookEditor {
	return newHookEditor(nil)
}

// InitialHookEditorEdit creates an editor pre-populated with an existing entry.
func InitialHookEditorEdit(e manifest.Entry) HookEditor {
	return newHookEditor(&e)
}

func newHookEditor(orig *manifest.Entry) HookEditor {
	makeInput := func(placeholder string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = 256
		ti.Width = 60
		return ti
	}

	var inputs [fieldCount]textinput.Model
	inputs[fieldName] = makeInput("e.g. eslint")
	inputs[fieldMatch] = makeInput("*.ts,*.js  (blank = all files)")
	inputs[fieldCommand] = makeInput("e.g. npx eslint --cache --fix")
	// Event field placeholder (not used for typing, cycled with enter/space).
	inputs[fieldEvent] = makeInput("")

	eventIdx := 0
	if orig != nil {
		inputs[fieldName].SetValue(orig.Name)
		inputs[fieldMatch].SetValue(strings.Join(orig.Match, ","))
		inputs[fieldCommand].SetValue(orig.Command)
		for i, e := range knownEvents {
			if e == orig.Event {
				eventIdx = i
				break
			}
		}
	}
	inputs[fieldEvent].SetValue(knownEvents[eventIdx])
	inputs[fieldName].Focus()

	return HookEditor{
		inputs:   inputs,
		eventIdx: eventIdx,
		Original: orig,
	}
}

// Init implements tea.Model.
func (m HookEditor) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (m HookEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.Quitting = true
			return m, tea.Quit

		case "tab", "down", "j":
			// Advance focus to next field (skip event field for typing).
			m.inputs[m.focusIdx].Blur()
			m.focusIdx = (m.focusIdx + 1) % fieldCount
			m.inputs[m.focusIdx].Focus()
			return m, textinput.Blink

		case "shift+tab", "up", "k":
			m.inputs[m.focusIdx].Blur()
			m.focusIdx = (m.focusIdx - 1 + fieldCount) % fieldCount
			m.inputs[m.focusIdx].Focus()
			return m, textinput.Blink

		case "enter":
			if m.focusIdx == fieldEvent {
				// Cycle event.
				m.eventIdx = (m.eventIdx + 1) % len(knownEvents)
				m.inputs[fieldEvent].SetValue(knownEvents[m.eventIdx])
				return m, nil
			}
			if m.focusIdx < fieldCount-1 {
				// Move to next field.
				m.inputs[m.focusIdx].Blur()
				m.focusIdx++
				m.inputs[m.focusIdx].Focus()
				return m, textinput.Blink
			}
			// Last field — save.
			if m.inputs[fieldName].Value() != "" && m.inputs[fieldCommand].Value() != "" {
				m.Confirmed = true
				return m, tea.Quit
			}

		case " ":
			if m.focusIdx == fieldEvent {
				m.eventIdx = (m.eventIdx + 1) % len(knownEvents)
				m.inputs[fieldEvent].SetValue(knownEvents[m.eventIdx])
				return m, nil
			}
		}
	}

	// Delegate key input to the focused textinput.
	var cmd tea.Cmd
	m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m HookEditor) View() string {
	var b strings.Builder
	title := "Create hook"
	if m.Original != nil {
		title = fmt.Sprintf("Edit hook: %s", m.Original.Name)
	}
	b.WriteString(shared_theme.HeaderStyle.Render(title) + "\n\n")
	b.WriteString(shared_theme.DimStyle.Render("tab/↑↓: move between fields  enter: next/save  esc: cancel") + "\n\n")

	renderField := func(idx int, label string) {
		cursor := "  "
		if m.focusIdx == idx {
			cursor = shared_theme.SelectStyle.Render("> ")
		}
		fmt.Fprintf(&b, "%s%s:  %s\n",
			cursor,
			shared_theme.HookNameStyle.Render(label),
			m.inputs[idx].View())
	}

	renderField(fieldName, "Name   ")
	renderField(fieldEvent, "Event  ")
	b.WriteString(shared_theme.DimStyle.Render("       (enter or space to cycle event)") + "\n")
	renderField(fieldMatch, "Match  ")
	b.WriteString(shared_theme.DimStyle.Render("       (comma-separated globs, blank = all)") + "\n")
	renderField(fieldCommand, "Command")
	b.WriteString("\n" + shared_theme.DimStyle.Render("enter on Command to save"))
	return b.String()
}

// ToEntry converts the editor state into a manifest.Entry.
func (m HookEditor) ToEntry() *manifest.Entry {
	if !m.Confirmed {
		return nil
	}
	var match []string
	raw := m.inputs[fieldMatch].Value()
	if raw != "" {
		for _, p := range strings.Split(raw, ",") {
			if p = strings.TrimSpace(p); p != "" {
				match = append(match, p)
			}
		}
	}
	return &manifest.Entry{
		Name:    m.inputs[fieldName].Value(),
		Event:   knownEvents[m.eventIdx],
		Match:   match,
		Command: m.inputs[fieldCommand].Value(),
	}
}

// RunHookEditor launches the hook editor TUI (standalone, for the template picker).
func RunHookEditor() (*manifest.Entry, error) {
	m := InitialHookEditor()
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	return final.(HookEditor).ToEntry(), nil
}

// ── TemplatePicker ────────────────────────────────────────────────────────────

// TemplatePicker lets the user choose which language templates to include
// and which individual hooks within each template to keep.
type TemplatePicker struct {
	Templates    []templates.Template
	Selected     map[int]bool
	HookSelected map[int]map[int]bool
	Cursor       int
	HookCursor   int
	Mode         int // 0 = template list, 1 = hook list within template
	CustomHooks  []manifest.Entry
	Quitting     bool
	Confirmed    bool
}

// InitialTemplatePicker returns the opening state for the template picker.
func InitialTemplatePicker() TemplatePicker {
	all := templates.All()
	selected := make(map[int]bool, len(all))
	hookSelected := make(map[int]map[int]bool, len(all))

	if len(all) > 0 {
		selected[0] = true
		hookSelected[0] = make(map[int]bool)
		for i := range all[0].Hooks {
			hookSelected[0][i] = true
		}
	}
	return TemplatePicker{
		Templates:    all,
		Selected:     selected,
		HookSelected: hookSelected,
	}
}

// Init implements tea.Model.
func (m TemplatePicker) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m TemplatePicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "ctrl+c", "esc":
		m.Quitting = true
		return m, tea.Quit

	case "up", "k":
		if m.Mode == 0 {
			if m.Cursor > 0 {
				m.Cursor--
			}
		} else {
			if m.HookCursor > 0 {
				m.HookCursor--
			}
		}

	case "down", "j":
		if m.Mode == 0 {
			max := len(m.Templates) + 1 // +1 for "Create custom hook"
			if m.Cursor < max-1 {
				m.Cursor++
			}
		} else {
			if m.HookCursor < len(m.Templates[m.Cursor].Hooks)-1 {
				m.HookCursor++
			}
		}

	case "G": // jump to bottom (C3)
		if m.Mode == 0 {
			m.Cursor = len(m.Templates) // last item = "create custom"
		} else {
			m.HookCursor = len(m.Templates[m.Cursor].Hooks) - 1
		}

	case "g": // jump to top (C3: simplified single-key gg)
		if m.Mode == 0 {
			m.Cursor = 0
		} else {
			m.HookCursor = 0
		}

	case " ":
		if m.Mode == 0 {
			if m.Cursor < len(m.Templates) {
				m.Selected[m.Cursor] = !m.Selected[m.Cursor]
				if m.Selected[m.Cursor] {
					if m.HookSelected[m.Cursor] == nil {
						m.HookSelected[m.Cursor] = make(map[int]bool)
					}
					for i := range m.Templates[m.Cursor].Hooks {
						m.HookSelected[m.Cursor][i] = true
					}
				}
			}
		} else {
			if m.HookSelected[m.Cursor] == nil {
				m.HookSelected[m.Cursor] = make(map[int]bool)
			}
			m.HookSelected[m.Cursor][m.HookCursor] = !m.HookSelected[m.Cursor][m.HookCursor]
		}

	case "a":
		if m.Mode == 0 {
			allSel := true
			for i := range m.Templates {
				if !m.Selected[i] {
					allSel = false
					break
				}
			}
			for i := range m.Templates {
				m.Selected[i] = !allSel
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

	case "c":
		if m.Mode == 0 && m.Cursor == len(m.Templates) {
			entry, err := RunHookEditor()
			if err == nil && entry != nil {
				m.CustomHooks = append(m.CustomHooks, *entry)
			}
		}

	case "enter":
		if m.Mode == 0 {
			hasSel := len(m.CustomHooks) > 0
			for _, s := range m.Selected {
				if s {
					hasSel = true
					break
				}
			}
			if hasSel {
				// Drill into first selected template's hooks.
				m.Mode = 1
				m.HookCursor = 0
				for m.Cursor < len(m.Templates) && !m.Selected[m.Cursor] {
					m.Cursor++
				}
				if m.Cursor >= len(m.Templates) {
					m.Cursor = 0
					m.Confirmed = true
					return m, tea.Quit
				}
			}
		} else {
			m.Confirmed = true
			return m, tea.Quit
		}

	case "backspace":
		if m.Mode == 1 {
			m.Mode = 0
			m.HookCursor = 0
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m TemplatePicker) View() string {
	var b strings.Builder

	if m.Mode == 0 {
		b.WriteString(shared_theme.HeaderStyle.Render("Select templates for your project") + "\n\n")
		b.WriteString(shared_theme.DimStyle.Render("space: toggle  a: all  c: create custom  j/k or ↑↓: move  g/G: top/bottom") + "\n")
		b.WriteString(shared_theme.DimStyle.Render("enter: continue") + "\n\n")

		for i, t := range m.Templates {
			cursor := "  "
			if m.Cursor == i {
				cursor = shared_theme.SelectStyle.Render("> ")
			}
			checkbox := shared_theme.CheckboxUnselected
			if m.Selected[i] {
				checkbox = shared_theme.CheckboxSelected
			}
			hookCount := 0
			if hs := m.HookSelected[i]; hs != nil {
				for _, s := range hs {
					if s {
						hookCount++
					}
				}
			}
			hookInfo := ""
			if m.Selected[i] {
				hookInfo = shared_theme.DimStyle.Render(fmt.Sprintf(" (%d hooks)", hookCount))
			}
			fmt.Fprintf(&b, "%s%s %s — %s%s\n", cursor, checkbox,
				shared_theme.HookNameStyle.Render(t.Name),
				shared_theme.DimStyle.Render(t.Description),
				hookInfo)
		}

		// Custom hook option.
		cursor := "  "
		if m.Cursor == len(m.Templates) {
			cursor = shared_theme.SelectStyle.Render("> ")
		}
		customInfo := ""
		if len(m.CustomHooks) > 0 {
			customInfo = shared_theme.DimStyle.Render(fmt.Sprintf(" (%d created)", len(m.CustomHooks)))
		}
		fmt.Fprintf(&b, "%s%s %s%s\n", cursor, shared_theme.CheckboxUnselected,
			shared_theme.HookNameStyle.Render("+ Create custom hook"), customInfo)

	} else {
		t := m.Templates[m.Cursor]
		b.WriteString(shared_theme.HeaderStyle.Render(fmt.Sprintf("Select hooks for %s", t.Name)) + "\n\n")
		b.WriteString(shared_theme.DimStyle.Render("space: toggle  backspace: back  enter: continue") + "\n\n")
		for i, h := range t.Hooks {
			cursor := "  "
			if m.HookCursor == i {
				cursor = shared_theme.SelectStyle.Render("> ")
			}
			checkbox := shared_theme.CheckboxUnselected
			if hs := m.HookSelected[m.Cursor]; hs != nil && hs[i] {
				checkbox = shared_theme.CheckboxSelected
			}
			fmt.Fprintf(&b, "%s%s %s (%s)\n", cursor, checkbox,
				shared_theme.HookNameStyle.Render(h.Name),
				shared_theme.EventStyle.Render(h.Event))
		}
	}

	b.WriteString("\n")
	// Status bar.
	var selNames []string
	for i, s := range m.Selected {
		if s {
			selNames = append(selNames, m.Templates[i].Name)
		}
	}
	if len(selNames) == 0 && len(m.CustomHooks) == 0 {
		b.WriteString(shared_theme.DimStyle.Render("Selected: (none)"))
	} else {
		parts := selNames
		if len(m.CustomHooks) > 0 {
			parts = append(parts, fmt.Sprintf("%d custom", len(m.CustomHooks)))
		}
		b.WriteString(shared_theme.DimStyle.Render("Selected: " + strings.Join(parts, ", ")))
	}

	return b.String()
}

// SelectedEntries returns the manifest entries for all selected templates/hooks
// including custom hooks. Called by the unified Model when advancing phases.
func (m TemplatePicker) SelectedEntries() []manifest.Entry {
	var entries []manifest.Entry
	entries = append(entries, m.CustomHooks...)
	for ti, sel := range m.Selected {
		if !sel {
			continue
		}
		t := m.Templates[ti]
		for hi, h := range t.Hooks {
			if hs := m.HookSelected[ti]; hs != nil && hs[hi] {
				entries = append(entries, h)
			}
		}
	}
	return entries
}

// RunTemplatePicker launches the template picker as a standalone program.
// Used when hookset init is called with an empty manifest.
func RunTemplatePicker() ([]manifest.Entry, error) {
	m := InitialTemplatePicker()
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	res := final.(TemplatePicker)
	if res.Quitting || !res.Confirmed {
		return nil, nil
	}
	return res.SelectedEntries(), nil
}
