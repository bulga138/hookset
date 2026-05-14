package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bulga138/hookset/internal/git"
	"github.com/bulga138/hookset/internal/gitconfig"
	"github.com/bulga138/hookset/internal/manifest"
	"github.com/bulga138/hookset/internal/scanner"
	"github.com/bulga138/hookset/internal/ui/init_view"
	isatty "github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	initStrict        bool
	initInteractive   bool
	initNoInteractive bool
	initDryRun        bool
	initUninstall     bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Read .hookset.toml and write hook entries into local git config",
	Long: `hookset init reads .hookset.toml from the repository root, resolves any
include, and writes each hook as a native [hook "name"] entry in .git/config.

Each written command is a self-checking one-liner: if hookset is not installed
the commit will fail with a clear installation message rather than a cryptic
"command not found" error.

Running hookset init multiple times is safe — it removes and rewrites each
hook entry (idempotent).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Handle Uninstall first
		if initUninstall {
			return runUninstall()
		}

		root, err := git.RepoRoot()
		if err != nil {
			return err
		}

		var entries []manifest.Entry
		tomlPath := filepath.Join(root, manifest.Filename)

		opts := manifest.ReadOptions{
			StrictIncludes: initStrict,
			Warn: func(msg string) {
				fmt.Fprintln(os.Stderr, "[hookset] warning:", msg)
			},
		}

		// 2. Handle Bootstrap if file is missing
		if _, err := os.Stat(tomlPath); os.IsNotExist(err) {
			if initNoInteractive {
				return fmt.Errorf("no .hookset.toml found and running in non-interactive mode")
			}

			fmt.Println("[hookset] No manifest found. Scanning project...")
			detected := scanner.Scan(root)

			actions, err := init_view.RunBootstrapWizard(detected)
			if err != nil {
				return err
			}
			if actions == nil {
				fmt.Println("[hookset] Bootstrap cancelled.")
				return nil
			}

			// Filter to only install actions for the manifest
			var selected []manifest.Entry
			for i, entry := range detected {
				if actions[i] == init_view.ActionInstall {
					selected = append(selected, entry)
				}
			}

			if err := manifest.Write(tomlPath, selected); err != nil {
				return fmt.Errorf("failed to save manifest: %w", err)
			}
			fmt.Printf("[hookset] Created %s\n", manifest.Filename)
			entries = selected
		} else {
			entries, err = manifest.Read(tomlPath, opts)
			if err != nil {
				return fmt.Errorf("reading manifest: %w", err)
			}
		}

		if len(entries) == 0 {
			fmt.Println("[hookset] No hook entries found — nothing to install.")
			return nil
		}

		// 3. Determine TUI usage
		useTUI := initInteractive
		if !initNoInteractive && !initInteractive {
			useTUI = isatty.IsTerminal(os.Stdout.Fd())
		}

		var actions map[int]init_view.HookAction
		if useTUI {
			var err error
			actions, err = init_view.Run(entries, initDryRun)
			if err != nil {
				return fmt.Errorf("TUI failed: %w", err)
			}
			if actions == nil {
				fmt.Println("[hookset] Installation cancelled.")
				return nil
			}
		} else {
			// Non-interactive: all entries are set to install
			actions = make(map[int]init_view.HookAction)
			for i := range entries {
				actions[i] = init_view.ActionInstall
			}
		}

		// 4. Handle Dry Run
		if initDryRun {
			fmt.Println("\n[hookset] DRY RUN: The following changes would be made:")
			for i, entry := range entries {
				switch actions[i] {
				case init_view.ActionInstall:
					fmt.Printf("  + Install: %s\n", entry.Name)
				case init_view.ActionUninstall:
					fmt.Printf("  - Remove: %s\n", entry.Name)
				case init_view.ActionIgnore:
					fmt.Printf("    Skip: %s\n", entry.Name)
				}
			}
			fmt.Println("[hookset] No changes were written.")
			return nil
		}

		// 5. Perform actions per hook
		fmt.Printf("[hookset] Processing %d hook(es)\n", len(entries))
		for i, entry := range entries {
			action := actions[i]
			switch action {
			case init_view.ActionInstall:
				// Wipe old and write new
				if err := gitconfig.RemoveHook(entry.Name, git.ScopeLocal); err != nil {
					// Not fatal if it didn't exist
					if !strings.Contains(err.Error(), "No such section") {
						return fmt.Errorf("removing existing hook %q: %w", entry.Name, err)
					}
				}
				h := gitconfig.Hook{
					Name:    entry.Name,
					Event:   entry.Event,
					Command: buildWrapperCommand(entry),
					Matches: entry.Match,
					Enabled: true,
				}
				if err := gitconfig.AddHook(h, git.ScopeLocal); err != nil {
					return fmt.Errorf("installing hook %q: %w", entry.Name, err)
				}
				fmt.Printf("  ✓ Installed: %s\n", entry.Name)

			case init_view.ActionUninstall:
				// Explicit removal
				if err := gitconfig.RemoveHook(entry.Name, git.ScopeLocal); err != nil {
					if !strings.Contains(err.Error(), "No such section") {
						return fmt.Errorf("removing hook %q: %w", entry.Name, err)
					}
				}
				fmt.Printf("  × Removed: %s\n", entry.Name)

			case init_view.ActionIgnore:
				// Do nothing, skip this entry
				fmt.Printf("  • Skipped: %s\n", entry.Name)
			}
		}

		fmt.Println("[hookset] Done. Verify with: hookset list")
		return nil
	},
}

// runUninstall is the entry point for the --uninstall flag
func runUninstall() error {
	fmt.Println("[hookset] Uninstalling all managed hooks from .git/config...")
	if err := uninstallManagedHooks(true); err != nil {
		return err
	}
	fmt.Println("[hookset] Done. Your .git/config is clean.")
	return nil
}

// uninstallManagedHooks finds and removes all hooks containing "hookset exec"
func uninstallManagedHooks(verbose bool) error {
	existingNames, err := gitconfig.GetAllHookNames(git.ScopeLocal)
	if err != nil {
		return fmt.Errorf("listing existing hooks: %w", err)
	}

	removedCount := 0
	for _, name := range existingNames {
		h, err := gitconfig.GetHook(name, git.ScopeLocal)
		if err != nil {
			continue
		}

		// Surgical removal: only touch hooks we own
		if strings.Contains(h.Command, "hookset exec") {
			if err := gitconfig.RemoveHook(name, git.ScopeLocal); err != nil {
				return fmt.Errorf("removing hook %q: %w", name, err)
			}
			if verbose {
				fmt.Printf("  - Removed: %s\n", name)
			}
			removedCount++
		}
	}

	if verbose && removedCount == 0 {
		fmt.Println("  (No hookset-managed hooks found)")
	}
	return nil
}

func init() {
	initCmd.Flags().BoolVar(&initStrict, "strict", false,
		"Treat missing include files as a fatal error instead of a warning")
	initCmd.Flags().BoolVar(&initInteractive, "interactive", false,
		"Force interactive TUI mode")
	initCmd.Flags().BoolVar(&initNoInteractive, "no-interactive", false,
		"Skip TUI and install all hooks non-interactively")
	initCmd.Flags().BoolVar(&initDryRun, "dry-run", false,
		"Show what would be installed without modifying .git/config")
	initCmd.Flags().BoolVar(&initUninstall, "uninstall", false,
		"Remove all hookset-managed hooks from .git/config and exit")
	rootCmd.AddCommand(initCmd)
}

// buildWrapperCommand constructs the self-checking hook command stored in git config.
// It is tailored to the OS running hookset init.
//
// Unix:    sh -c 'command -v hookset >/dev/null 2>&1 || { echo "..." >&2; exit 1; }; exec hookset exec ...'
// Windows: cmd /c "where hookset >nul 2>&1 || (echo ... 1>&2 & exit /b 1) & hookset exec ..."
func buildWrapperCommand(entry manifest.Entry) string {
	hooksetInstallMsg := "hookset is not installed. Visit https://hookset.dev"

	// Build the hookset exec invocation.
	var execParts []string
	execParts = append(execParts, "hookset", "exec")
	for _, m := range entry.Match {
		execParts = append(execParts, "--match", shellQuote(m))
	}
	if len(entry.Match) > 0 {
		// File-filtering hook — append -- command
		execParts = append(execParts, "--")
		execParts = append(execParts, strings.Fields(entry.Command)...)
	} else {
		// Non-filtering hook (e.g., "go test ./...")
		execParts := strings.Fields(entry.Command)
		binary := execParts[0]

		// If we are checking for something other than hookset, use a generic message
		installMsg := hooksetInstallMsg
		if binary != "hookset" {
			installMsg = fmt.Sprintf("%s is not installed and is required for this hook.", binary)
		}

		return buildPresenceCheck(binary, installMsg, strings.Join(execParts, " "))
	}
	return buildPresenceCheck("hookset", hooksetInstallMsg, strings.Join(execParts, " "))
}

func buildPresenceCheck(binary, installMsg, execCmd string) string {
	if runtime.GOOS == "windows" {
		// Git for Windows runs hook commands via sh, so we use sh syntax here too.
		// Check PATH first, then current directory (for development workflows).
		return fmt.Sprintf(
			`sh -c 'command -v %s >/dev/null 2>&1 || { test -f "./%s.exe" && exec "./%s.exe" exec --help >/dev/null 2>&1 || { printf "%%s\n" "%s" >&2; exit 1; }; }; exec %s'`,
			binary, binary, binary, installMsg, execCmd,
		)
	}
	// Unix: check PATH, then current directory
	return fmt.Sprintf(
		`sh -c 'command -v %s >/dev/null 2>&1 || { test -f "./%s" && exec "./%s" exec --help >/dev/null 2>&1 || { printf "%%s\n" "%s" >&2; exit 1; }; }; exec %s'`,
		binary, binary, binary, installMsg, execCmd,
	)
}

func shellQuote(s string) string {
	if !strings.ContainsAny(s, " \t\"'") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
