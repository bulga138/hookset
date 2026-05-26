// Package init_view provides the bubbletea TUI for hookset init and bootstrap.
//
// The unified Model drives all phases through a single state machine:
//
//	PhaseTemplatePick → PhaseHookReview → PhaseConfirm → (done)
//
// All phases handle tea.WindowSizeMsg so the UI adapts to terminal resize.
package init_view

import (
	"fmt"
	"strings"

	"github.com/bulga138/hookset/internal/manifest"
	tea "github.com/charmbracelet/bubbletea"
)

// Phase controls which screen is active.
type Phase int

const (
	PhaseTemplatePick Phase = iota // Select language templates
	PhaseHookReview                // Review/toggle individual hooks
	PhaseConfirm                   // Diff preview before writing
)

// HookAction is the per-entry install state.
type HookAction int

const (
	ActionInstall   HookAction = iota // [✓] write to git config
	ActionIgnore                      // [ ] skip this hook
	ActionUninstall                   // [*] remove from git config
)

// Model is the single unified bubbletea model for all init/bootstrap phases.
type Model struct {
	// Shared
	Phase     Phase
	TermWidth int
	TermHeight int
	IsBootstrap bool
	DryRun      bool
	Quitting    bool
	Confirmed   bool

	// PhaseTemplatePick
	Picker *TemplatePicker

	// PhaseHookReview
	Entries []manifest.Entry
	Actions map[int]HookAction
	Cursor  int

	// PhaseConfirm — diff between current git config hooks and proposed entries
	DiffLines []DiffLine
}

// DiffLine is one line in the PhaseConfirm diff preview.
type DiffLine struct {
	Op   string // "+" add, "-" remove, " " unchanged
	Text string
}

// InitialModel creates the opening state. If entries is non-empty, skip
// straight to PhaseHookReview (used when manifest already exists).
func InitialModel(entries []manifest.Entry) Model {
	actions := make(map[int]HookAction, len(entries))
	for i := range entries {
		actions[i] = ActionInstall
	}
	phase := PhaseHookReview
	var picker *TemplatePicker
	if len(entries) == 0 {
		phase = PhaseTemplatePick
		p := InitialTemplatePicker()
		picker = &p
	}
	return Model{
		Phase:   phase,
		Picker:  picker,
		Entries: entries,
		Actions: actions,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.Picker != nil {
		return m.Picker.Init()
	}
	return nil
}

// Update implements tea.Model (C1 unified dispatcher + C8 resize).
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// C8: handle terminal resize in every phase.
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.TermWidth = ws.Width
		m.TermHeight = ws.Height
		if m.TermWidth > 4 {
			// Propagate to border style width.
			_ = m.TermWidth // used in View via dynamic width
		}
		return m, nil
	}

	switch m.Phase {
	case PhaseTemplatePick:
		return m.updateTemplatePick(msg)
	case PhaseHookReview:
		return m.updateHookReview(msg)
	case PhaseConfirm:
		return m.updateConfirm(msg)
	}
	return m, nil
}

// ── PhaseTemplatePick ─────────────────────────────────────────────────────────

func (m Model) updateTemplatePick(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.Picker == nil {
		return m, nil
	}
	next, cmd := m.Picker.Update(msg)
	picker := next.(TemplatePicker)
	m.Picker = &picker

	if picker.Quitting {
		m.Quitting = true
		return m, tea.Quit
	}
	if picker.Confirmed {
		// Collect selected entries and advance to review phase.
		m.Entries = picker.SelectedEntries()
		m.Actions = make(map[int]HookAction, len(m.Entries))
		for i := range m.Entries {
			m.Actions[i] = ActionInstall
		}
		m.Phase = PhaseHookReview
		m.Cursor = 0
	}
	return m, cmd
}

// ── PhaseHookReview ───────────────────────────────────────────────────────────

func (m Model) updateHookReview(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	n := len(m.Entries)
	switch key.String() {
	case "ctrl+c", "q", "esc":
		m.Quitting = true
		return m, tea.Quit
	// C3: vim movement
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "down", "j":
		if m.Cursor < n-1 {
			m.Cursor++
		}
	case "G": // jump to last
		m.Cursor = max(0, n-1)
	case "g": // gg handled on second g press via composite — simplified: treat as top
		m.Cursor = 0
	case " ":
		// Cycle: Install → Ignore → Uninstall → Install
		m.Actions[m.Cursor] = (m.Actions[m.Cursor] + 1) % 3
	case "g g": // bubbletea delivers this as two separate events; handled above via "g"
	case "a":
		allInstall := true
		for _, a := range m.Actions {
			if a != ActionInstall {
				allInstall = false
				break
			}
		}
		want := ActionInstall
		if allInstall {
			want = ActionIgnore
		}
		for i := range m.Entries {
			m.Actions[i] = want
		}
	case "enter":
		// Advance to confirm phase — build diff preview (C4/C6).
		m.DiffLines = buildDiff(m.Entries, m.Actions)
		m.Phase = PhaseConfirm
	}
	return m, nil
}

// ── PhaseConfirm ─────────────────────────────────────────────────────────────

func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+c", "q", "esc":
		// Go back to review phase.
		m.Phase = PhaseHookReview
	case "y", "enter":
		m.Confirmed = true
		return m, tea.Quit
	case "n":
		m.Quitting = true
		return m, tea.Quit
	}
	return m, nil
}

// ── diff builder (C4/C6) ─────────────────────────────────────────────────────

// buildDiff produces a unified-like diff between what's proposed and what
// would be installed. Passthrough hooks are annotated; match patterns shown.
func buildDiff(entries []manifest.Entry, actions map[int]HookAction) []DiffLine {
	var lines []DiffLine
	for i, e := range entries {
		act := actions[i]
		var op string
		switch act {
		case ActionInstall:
			op = "+"
		case ActionUninstall:
			op = "-"
		default:
			continue // ActionIgnore → not written, not shown
		}
		text := fmt.Sprintf("%-20s %-18s %s", e.Name, e.Event, e.Command)
		if len(e.Match) > 0 {
			text += fmt.Sprintf("  [%s]", strings.Join(e.Match, ", "))
		}
		if e.Passthrough {
			text += "  (passthrough)"
		}
		lines = append(lines, DiffLine{Op: op, Text: text})
	}
	return lines
}

// ── helpers ───────────────────────────────────────────────────────────────────

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
