package commands

import (
	"fmt"

	"github.com/bulga138/hookset/internal/git"
	"github.com/bulga138/hookset/internal/gitconfig"
	"github.com/bulga138/hookset/internal/manifest"
	"github.com/spf13/cobra"
)

// ── remove ────────────────────────────────────────────────────────────────────

var (
	removeManifest bool
	removeGlobal   bool
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a hook from git config (and optionally the manifest)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		scope := git.ScopeLocal
		if removeGlobal {
			scope = git.ScopeGlobal
		}

		if err := gitconfig.RemoveHook(name, scope); err != nil {
			return err
		}
		fmt.Printf("[hookset] Removed hook %q from %s config\n", name, scope)

		if removeManifest {
			root, err := git.RepoRoot()
			if err != nil {
				return err
			}
			if err := manifest.Remove(root+"/"+manifest.Filename, name); err != nil {
				return fmt.Errorf("removing from manifest: %w", err)
			}
			fmt.Printf("[hookset] Removed hook %q from %s\n", name, manifest.Filename)
		}
		return nil
	},
}

// ── disable / enable ──────────────────────────────────────────────────────────

var toggleGlobal bool

var disableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a hook without removing it (sets hook.<name>.enabled = false)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return toggleHook(args[0], false)
	},
}

var enableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Re-enable a previously disabled hook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return toggleHook(args[0], true)
	},
}

func toggleHook(name string, enabled bool) error {
	scope := git.ScopeLocal
	if toggleGlobal {
		scope = git.ScopeGlobal
	}
	if err := gitconfig.EnableHook(name, scope, enabled); err != nil {
		return err
	}
	state := "disabled"
	if enabled {
		state = "enabled"
	}
	fmt.Printf("[hookset] Hook %q %s in %s config\n", name, state, scope)
	return nil
}

// ── list ──────────────────────────────────────────────────────────────────────

var listEvent string

var listCmd = &cobra.Command{
	Use:   "list [event]",
	Short: "List configured hooks (wraps git hook list)",
	Long: `List hooks, optionally filtered by event.

Output mirrors 'git hook list' but with hookset match patterns included.
Shows scope (local/global/system) for each hook.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		event := listEvent
		if len(args) > 0 {
			event = args[0]
		}

		// Show git's own hook list first.
		fmt.Println(git.HookList(event))

		// Augment with match pattern info from our config.
		for _, scope := range []git.Scope{git.ScopeLocal, git.ScopeGlobal} {
			hooks, err := gitconfig.GetHooks(event, scope)
			if err != nil || len(hooks) == 0 {
				continue
			}
			for _, h := range hooks {
				if len(h.Matches) > 0 {
					fmt.Printf("  [hookset] %s matches: %v\n", h.Name, h.Matches)
				}
			}
		}
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVar(&removeManifest, "manifest", false, "Also remove from .hookset.toml")
	removeCmd.Flags().BoolVar(&removeGlobal, "global", false, "Target global config")
	disableCmd.Flags().BoolVar(&toggleGlobal, "global", false, "Target global config")
	enableCmd.Flags().BoolVar(&toggleGlobal, "global", false, "Target global config")
	listCmd.Flags().StringVar(&listEvent, "event", "", "Filter by hook event")

	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(disableCmd)
	rootCmd.AddCommand(enableCmd)
	rootCmd.AddCommand(listCmd)
}
