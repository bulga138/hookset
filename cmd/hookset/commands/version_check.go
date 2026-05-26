package commands

import (
	"fmt"

	"github.com/bulga138/hookset/internal/git"
)

// requireGit254 checks that git version is at least 2.54.
// It is called by mutating commands (add, init, migrate, etc.) to enforce the minimum version.
func requireGit254() error {
	if err := git.CheckMinVersion(2, 54); err != nil {
		return fmt.Errorf("hookset requires git ≥ 2.54: %w", err)
	}
	return nil
}
