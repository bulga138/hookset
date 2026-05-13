package init_view

import (
	"github.com/bulga138/hookset/internal/manifest"
	tea "github.com/charmbracelet/bubbletea"
)

// RunBootstrapWizard is the entry point when no .hookset.toml exists.
func RunBootstrapWizard(suggestions []manifest.Entry) (map[int]HookAction, error) {
	// 1. Initialize model with suggestions
	// We mark all suggested hooks as selected by default
	m := InitialModel(suggestions)
	m.IsBootstrap = true // We add this flag to the model to change the Title in View()

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	res := finalModel.(Model)
	if res.Quitting || !res.Confirmed {
		return nil, nil
	}

	return res.Actions, nil
}
