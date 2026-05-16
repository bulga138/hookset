package init_view

import (
	"fmt"
	"strings"

	"github.com/bulga138/hookset/internal/ui/shared_theme"
)

func (m Model) View() string {
	if m.Quitting {
		return "Installation cancelled.\n"
	}
	if m.Confirmed {
		return "Applying configuration...\n"
	}

	var b strings.Builder

	// Dynamic Header
	title := "📦 hookset init"
	subtext := "Select hooks to install:"

	if m.IsBootstrap {
		title = "📦 hookset bootstrap"
		subtext = "I've detected these tools. Create .hookset.toml with:"
	} else if m.DryRun {
		title = "📦 hookset init [DRY RUN]"
	}

	b.WriteString(shared_theme.TitleStyle.Render(title) + "\n\n")
	b.WriteString(shared_theme.HeaderStyle.Render(subtext) + "\n")

	if m.DryRun {
		b.WriteString(shared_theme.WarningStyle.Render("⚠️  Simulation mode: no changes will be written") + "\n\n")
	} else {
		b.WriteString(shared_theme.DimStyle.Render("space: toggle • g: group • a: all • enter: apply") + "\n\n")
	}

	// Hook List
	for i, entry := range m.Entries {
		cursor := "  "
		if m.Cursor == i {
			cursor = shared_theme.SelectStyle.Render("> ")
		}

		// Render the correct state icon
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
			matches = shared_theme.DimStyle.Render("(no file filter)")
		}

		fmt.Fprintf(&b, "%s%s %s %s\n",
			cursor, checked, label, matches)

		if m.IsBootstrap {
			_ = label + shared_theme.DimStyle.Render(" (suggested)") // unused but documents intent
		}
	}

	// Command Preview
	current := m.Entries[m.Cursor]
	preview := "\n" + shared_theme.HeaderStyle.Render("Command Preview:") + "\n"
	preview += shared_theme.DimStyle.Render(fmt.Sprintf("hookset exec -- %s", current.Command))

	return shared_theme.BorderStyle.Render(b.String() + preview)
}
