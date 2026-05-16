package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/bulga138/hookset/internal/git"
	"github.com/bulga138/hookset/internal/gitconfig"
	"github.com/bulga138/hookset/internal/manifest"
	"github.com/spf13/cobra"
)

var (
	addMatches  []string
	addEvent    string
	addManifest bool
	addGlobal   bool
	addDirect   bool
)

var addCmd = &cobra.Command{
	Use:   "add <name> [flags] -- <command>",
	Short: "Add a hook to git config or .hookset.toml",
	Long: `Add a named hook entry.

  Without --manifest: writes directly to .git/config (personal, uncommitted).
  With --manifest:    appends to .hookset.toml only; run hookset init to apply.
  With --global:      writes to ~/.gitconfig (applies to all repos on this machine).

Examples:
  # Personal hook — not shared with the team
  hookset add typecheck --on pre-push -- tsc --noEmit

  # Team hook — add to manifest, then install
  hookset add eslint --match "*.ts" --match "*.js" --manifest -- npx eslint --cache --fix
  hookset init

  # Global hook on this machine only
  hookset add secrets --global --on pre-commit -- detect-secrets scan`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("hook name is required")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		command := findDashDash()
		if command == nil {
			return fmt.Errorf("command is required after --\n  Example: hookset add eslint -- npx eslint --cache --fix")
		}

		entry := manifest.Entry{
			Name:    name,
			Event:   addEvent,
			Match:   addMatches,
			Command: strings.Join(command, " "),
		}

		if addManifest {
			return addToManifest(entry)
		}

		scope := git.ScopeLocal
		if addGlobal {
			scope = git.ScopeGlobal
		}
		return addToConfig(entry, scope)
	},
}

func init() {
	addCmd.Flags().StringArrayVar(&addMatches, "match", nil,
		"File pattern to match (repeatable). Uses git pathspec globs.")
	addCmd.Flags().StringVar(&addEvent, "on", "pre-commit",
		"Git hook event (pre-commit, pre-push, commit-msg, …)")
	addCmd.Flags().BoolVar(&addManifest, "manifest", false,
		"Append to .hookset.toml instead of writing git config")
	addCmd.Flags().BoolVar(&addGlobal, "global", false,
		"Write to ~/.gitconfig (ignored when --manifest is set)")
	addCmd.Flags().BoolVar(&addDirect, "direct", false,
		"Generate simple command without hookset wrapper")
	rootCmd.AddCommand(addCmd)
}

func addToManifest(entry manifest.Entry) error {
	root, err := git.RepoRoot()
	if err != nil {
		return err
	}
	tomlPath := root + "/" + manifest.Filename
	if err := manifest.Upsert(tomlPath, entry); err != nil {
		return fmt.Errorf("updating manifest: %w", err)
	}
	fmt.Printf("[hookset] Added %q to %s\n", entry.Name, manifest.Filename)
	fmt.Println("         Run hookset init to install.")
	return nil
}

func addToConfig(entry manifest.Entry, scope git.Scope) error {
	wrapCmd := buildWrapperCommand(entry, addDirect)
	h := gitconfig.Hook{
		Name:    entry.Name,
		Event:   entry.Event,
		Command: wrapCmd,
		Matches: entry.Match,
		Enabled: true,
	}
	if err := gitconfig.AddHook(h, scope); err != nil {
		return fmt.Errorf("writing git config: %w", err)
	}
	scopeLabel := "local"
	if scope == git.ScopeGlobal {
		scopeLabel = "global"
	}
	fmt.Printf("[hookset] Added hook %q (%s) to %s git config\n", entry.Name, entry.Event, scopeLabel)
	return nil
}

// findDashDash scans os.Args for the -- separator and returns everything after it.
func findDashDash() []string {
	for i, a := range os.Args {
		if a == "--" && i+1 < len(os.Args) {
			return os.Args[i+1:]
		}
	}
	return nil
}
