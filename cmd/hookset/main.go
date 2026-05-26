package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bulga138/hookset/cmd/hookset/commands"
	"github.com/bulga138/hookset/internal/git"
	"github.com/bulga138/hookset/internal/gitconfig"
	"github.com/bulga138/hookset/internal/manifest"
)

func main() {
	// Skip version check for `hookset version` and `hookset help`.
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version" ||
		os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h") {
		commands.Execute()
		return
	}

	if err := git.CheckMinVersion(2, 54); err != nil {
		fmt.Fprintln(os.Stderr, "[hookset] error:", err)
		os.Exit(1)
	}

	// Check if hookset version has changed and update hooks if needed
	// (do this before any other operation)
	commands.CheckAndUpdateHooks()

	// Handle default behavior: no subcommand given
	if len(os.Args) == 1 {
		handleDefault()
		return
	}

	commands.Execute()
}

// handleDefault determines what to do when no subcommand is provided.
// If inside a git repo with hooks configured OR with a .hookset.toml file, show the list.
// Otherwise, show the help.
func handleDefault() {
	// Check if we're inside a git repo
	root, err := git.RepoRoot()
	if err != nil {
		// Not in a git repo - show help
		commands.Execute()
		return
	}

	// Check for .hookset.toml file
	tomlPath := filepath.Join(root, manifest.Filename)
	_, err = os.Stat(tomlPath)
	hasToml := err == nil

	// Check for configured hooks
	hooks, err := gitconfig.GetAllHookNames(git.ScopeLocal)
	hasHooks := err == nil && len(hooks) > 0

	// Check global hooks as well
	globalHooks, err := gitconfig.GetAllHookNames(git.ScopeGlobal)
	hasGlobalHooks := err == nil && len(globalHooks) > 0

	if hasToml || hasHooks || hasGlobalHooks {
		// Show the list of configured hooks
		if err := commands.RunList(); err != nil {
			fmt.Fprintln(os.Stderr, "[hookset] error:", err)
			os.Exit(1)
		}
	} else {
		// No hooks configured and no .hookset.toml - show help
		commands.Execute()
	}
}
