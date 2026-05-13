package init_view

import (
	"github.com/bulga138/hookset/internal/manifest"
	tea "github.com/charmbracelet/bubbletea"
)

func Run(entries []manifest.Entry, dryRun bool) (map[int]HookAction, error) {
	// 1. Create the program with our InitialModel
	model := InitialModel(entries)
	model.DryRun = dryRun
	p := tea.NewProgram(model)

	// 2. Run the TUI
	m, err := p.Run()
	if err != nil {
		return nil, err
	}

	// 3. Assert back to our specific Model type to read the state
	finalModel := m.(Model)

	// 4. Handle cancellation
	if finalModel.Quitting || !finalModel.Confirmed {
		return nil, nil
	}

	// 5. Return the actions map for all entries
	return finalModel.Actions, nil
}
