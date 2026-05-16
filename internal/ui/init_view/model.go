package init_view

import (
	"github.com/bulga138/hookset/internal/manifest"
	tea "github.com/charmbracelet/bubbletea"
)

// Define the three states
type HookAction int

const (
	ActionInstall   HookAction = iota // [✓]
	ActionIgnore                      // [ ]
	ActionUninstall                   // [*]
)

type Model struct {
	Entries     []manifest.Entry
	Actions     map[int]HookAction // Changed from map[int]bool
	Cursor      int
	Confirmed   bool
	Quitting    bool
	DryRun      bool
	IsBootstrap bool
}

// InitialModel fixes the "undefined: InitialModel" error
func InitialModel(entries []manifest.Entry) Model {
	actions := make(map[int]HookAction)
	for i := range entries {
		actions[i] = ActionInstall // Default to install
	}
	return Model{
		Entries: entries,
		Actions: actions,
	}
}

// Init fixes the "missing method Init" error
func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.Cursor < len(m.Entries)-1 {
				m.Cursor++
			}
		case " ":
			// Cycle: Install -> Ignore -> Uninstall -> Install
			current := m.Actions[m.Cursor]
			m.Actions[m.Cursor] = (current + 1) % 3
		case "g":
			// Group Toggle: toggle between Install and Ignore for all hooks with same event
			targetEvent := m.Entries[m.Cursor].Event
			current := m.Actions[m.Cursor]
			var newAction HookAction
			if current == ActionInstall {
				newAction = ActionIgnore
			} else {
				newAction = ActionInstall
			}
			for i, e := range m.Entries {
				if e.Event == targetEvent {
					m.Actions[i] = newAction
				}
			}
		case "a":
			// Toggle All: set all to Install if any are not Install, else set all to Ignore
			allInstall := true
			for _, act := range m.Actions {
				if act != ActionInstall {
					allInstall = false
					break
				}
			}
			var newAction HookAction
			if allInstall {
				newAction = ActionIgnore
			} else {
				newAction = ActionInstall
			}
			for i := range m.Entries {
				m.Actions[i] = newAction
			}
		case "enter":
			m.Confirmed = true
			return m, tea.Quit
		}
	}
	return m, nil
}
