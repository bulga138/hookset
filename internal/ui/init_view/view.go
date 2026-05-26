package init_view

import (
	"fmt"
	"strings"

	"github.com/bulga138/hookset/internal/ui/shared_theme"
	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model (C1 unified view dispatcher).
func (m Model) View() string {
	if m.Quitting {
		return "Installation cancelled.\n"
	}
	if m.Confirmed {
		return "Applying configuration...\n"
	}

	switch m.Phase {
	case PhaseTemplatePick:
		if m.Picker != nil {
			return m.Picker.View()
		}
		return ""
	case PhaseHookReview:
		return m.viewHookReview()
	case PhaseConfirm:
		return m.viewConfirm()
	}
	return ""
}

// ── PhaseHookReview view ──────────────────────────────────────────────────────

func (m Model) viewHookReview() string {
	var b strings.Builder

	title := "hookset init"
	if m.IsBootstrap {
		title = "hookset bootstrap"
	} else if m.DryRun {
		title = "hookset init [DRY RUN]"
	}
	b.WriteString(shared_theme.TitleStyle.Render(title) + "\n\n")

	subtext := "Select hooks to install:"
	if m.IsBootstrap {
		subtext = "Detected hooks — select which to install:"
	}
	b.WriteString(shared_theme.HeaderStyle.Render(subtext) + "\n")

	if m.DryRun {
		b.WriteString(shared_theme.WarningStyle.Render("⚠  Simulation mode: no changes will be written") + "\n\n")
	} else {
		b.WriteString(shared_theme.DimStyle.Render("space: toggle  j/k or ↑↓: move  g/G: top/bottom  a: all  enter: continue") + "\n\n")
	}

	for i, entry := range m.Entries {
		cursor := "  "
		if m.Cursor == i {
			cursor = shared_theme.SelectStyle.Render("> ")
		}

		var checked string
		switch m.Actions[i] {
		case ActionInstall:
			checked = shared_theme.CheckboxSelected
		case ActionIgnore:
			checked = shared_theme.CheckboxUnselected
		case ActionUninstall:
			checked = shared_theme.CheckboxUninstall
		}

		label := fmt.Sprintf("%-15s %-12s", entry.Name, entry.Event)
		if m.Cursor == i {
			label = shared_theme.SelectStyle.Render(label)
		}

		matches := ""
		if len(entry.Match) > 0 {
			matches = shared_theme.DimStyle.Render(strings.Join(entry.Match, ", "))
		} else {
			matches = shared_theme.DimStyle.Render("(all files)")
		}

		fmt.Fprintf(&b, "%s%s %s %s\n", cursor, checked, label, matches)
	}

	// Command preview for current entry.
	if len(m.Entries) > 0 {
		current := m.Entries[m.Cursor]
		preview := "\n" + shared_theme.HeaderStyle.Render("Command Preview:") + "\n"
		preview += shared_theme.DimStyle.Render(fmt.Sprintf("hookset exec -- %s", current.Command))
		b.WriteString(preview)
	}

	w := 80
	if m.TermWidth > 20 {
		w = m.TermWidth - 4
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(shared_theme.PrimaryColor).
		Padding(1, 2).
		Margin(0, 0, 1, 0).
		Width(w).
		Render(b.String())
}

// ── PhaseConfirm view (C4 diff preview) ──────────────────────────────────────

func (m Model) viewConfirm() string {
	var b strings.Builder

	b.WriteString(shared_theme.TitleStyle.Render("hookset — confirm changes") + "\n\n")
	b.WriteString(shared_theme.HeaderStyle.Render("The following hooks will be written to git config:") + "\n\n")

	addStyle := lipgloss.NewStyle().Foreground(shared_theme.PrimaryColor)
	removeStyle := lipgloss.NewStyle().Foreground(shared_theme.ErrorColor)
	dimStyle := shared_theme.DimStyle

	if len(m.DiffLines) == 0 {
		b.WriteString(dimStyle.Render("  (no changes)") + "\n")
	} else {
		for _, dl := range m.DiffLines {
			switch dl.Op {
			case "+":
				b.WriteString(addStyle.Render("+  ") + dl.Text + "\n")
			case "-":
				b.WriteString(removeStyle.Render("-  ") + dl.Text + "\n")
			default:
				b.WriteString("   " + dl.Text + "\n")
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(shared_theme.DimStyle.Render("y / enter: apply  n: cancel  esc: back to review") + "\n")

	w := 80
	if m.TermWidth > 20 {
		w = m.TermWidth - 4
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(shared_theme.PrimaryColor).
		Padding(1, 2).
		Margin(0, 0, 1, 0).
		Width(w).
		Render(b.String())
}
