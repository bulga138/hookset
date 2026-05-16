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
	"github.com/bulga138/hookset/internal/templates"
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
	initRecursive     bool
	initTemplate      string
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
hook entry (idempotent).

Use --recursive to discover .hookset.toml files in subdirectories and merge
them into a single hook configuration.`,
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

		opts := manifest.ReadOptions{
			StrictIncludes: initStrict,
			Warn: func(msg string) {
				fmt.Fprintln(os.Stderr, "[hookset] warning:", msg)
			},
		}

		// Handle recursive mode
		if initRecursive {
			entries, err = discoverAndMergeRecursive(root, opts)
			if err != nil {
				return fmt.Errorf("recursive init failed: %w", err)
			}
		} else if initTemplate != "" {
			// Handle template mode
			tomlPath := filepath.Join(root, manifest.Filename)

			// Parse template names (comma-separated)
			templateNames := strings.Split(initTemplate, ",")
			for i := range templateNames {
				templateNames[i] = strings.TrimSpace(templateNames[i])
			}

			// Generate TOML content
			content, err := templates.Generate(templateNames)
			if err != nil {
				return fmt.Errorf("template error: %w", err)
			}

			// Check if file exists
			exists := false
			if _, err := os.Stat(tomlPath); err == nil {
				exists = true
			}

			if exists && !initDryRun && !initNoInteractive {
				fmt.Printf("[hookset] %s already exists. Overwrite? [y/N] ", manifest.Filename)
				var answer string
				if _, err := fmt.Scanln(&answer); err != nil {
					answer = ""
				}
				if strings.ToLower(strings.TrimSpace(answer)) != "y" {
					fmt.Println("[hookset] Aborted.")
					return nil
				}
			}

			if initDryRun {
				fmt.Println("[hookset] DRY RUN: Generated config:")
				fmt.Println(content)
				return nil
			}

			if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("writing template: %w", err)
			}
			fmt.Printf("[hookset] Created %s from template(s): %s\n", manifest.Filename, initTemplate)

			// Read back the generated entries
			entries, err = manifest.Read(tomlPath, opts)
			if err != nil {
				return fmt.Errorf("reading generated manifest: %w", err)
			}
		} else {
			// Standard single-file mode
			tomlPath := filepath.Join(root, manifest.Filename)

			// 2. Handle Bootstrap if file is missing
			if _, err := os.Stat(tomlPath); os.IsNotExist(err) {
				if initNoInteractive {
					return fmt.Errorf("no .hookset.toml found and running in non-interactive mode")
				}

				// Ask user: auto-detect or select templates?
				fmt.Println("[hookset] No manifest found.")
				fmt.Println("Choose how to set up hooks:")
				fmt.Println("  1) Auto-detect project type (scanner)")
				fmt.Println("  2) Select from templates")
				fmt.Print("Enter choice (1/2): ")

				var choice string
				if _, err := fmt.Scanln(&choice); err != nil {
					choice = "1" // default to auto-detect
				}
				choice = strings.TrimSpace(choice)

				var selected []manifest.Entry

				if choice == "2" {
					// Template picker mode - allows selecting individual hooks
					selectedEntries, err := init_view.RunTemplatePicker()
					if err != nil {
						return fmt.Errorf("template picker failed: %w", err)
					}
					if selectedEntries == nil {
						fmt.Println("[hookset] Template selection cancelled.")
						return nil
					}

					// Write the selected entries to manifest
					if err := manifest.Write(tomlPath, selectedEntries); err != nil {
						return fmt.Errorf("writing manifest: %w", err)
					}
					fmt.Printf("[hookset] Created %s with %d hook(s)\n", manifest.Filename, len(selectedEntries))

					entries = selectedEntries
				} else {
					// Auto-detect mode (default)
					fmt.Println("[hookset] Scanning project...")
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
				}
			} else {
				entries, err = manifest.Read(tomlPath, opts)
				if err != nil {
					return fmt.Errorf("reading manifest: %w", err)
				}

				// Offer to add more hooks if file exists
				if !initNoInteractive && !initInteractive && isatty.IsTerminal(os.Stdout.Fd()) {
					fmt.Printf("[hookset] %s already exists with %d hook(s).\n", manifest.Filename, len(entries))
					fmt.Println("Choose action:")
					fmt.Println("  1) Edit existing hooks")
					fmt.Println("  2) Add more hooks from templates")
					fmt.Print("Enter choice (1/2): ")

					var choice string
					if _, err := fmt.Scanln(&choice); err != nil {
						choice = "1"
					}
					choice = strings.TrimSpace(choice)

					if choice == "2" {
						// Run template picker to add more hooks
						newEntries, err := init_view.RunTemplatePicker()
						if err != nil {
							return fmt.Errorf("template picker failed: %w", err)
						}
						if len(newEntries) > 0 {
							// Merge with existing entries (avoid duplicates by name)
							existingNames := map[string]bool{}
							for _, e := range entries {
								existingNames[e.Name] = true
							}
							for _, e := range newEntries {
								if !existingNames[e.Name] {
									entries = append(entries, e)
									existingNames[e.Name] = true
								}
							}
							// Save merged entries
							if err := manifest.Write(tomlPath, entries); err != nil {
								return fmt.Errorf("writing manifest: %w", err)
							}
							fmt.Printf("[hookset] Added %d new hook(s) to %s\n", len(newEntries), manifest.Filename)
						}
					}
				}
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
	initCmd.Flags().BoolVar(&initRecursive, "recursive", false,
		"Discover .hookset.toml in subdirectories and merge them")
	initCmd.Flags().StringVar(&initTemplate, "template", "",
		"Generate config from template (typescript, python, go, rust, monorepo)")
	rootCmd.AddCommand(initCmd)
}

// discoverAndMergeRecursive walks the repository tree, finds all .hookset.toml
// files, and merges them into a single list of entries. Subdirectory paths are
// prepended to match patterns so that "*.ts" in "packages/frontend" becomes
// "packages/frontend/*.ts" when registered in Git's config.
func discoverAndMergeRecursive(root string, opts manifest.ReadOptions) ([]manifest.Entry, error) {
	fmt.Println("[hookset] Discovering .hookset.toml files...")

	// Find all .hookset.toml files
	var tomlFiles []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		if !info.IsDir() && info.Name() == manifest.Filename {
			tomlFiles = append(tomlFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking directory: %w", err)
	}

	if len(tomlFiles) == 0 {
		return nil, fmt.Errorf("no %s files found in repository", manifest.Filename)
	}

	fmt.Printf("[hookset] Found %d config file(s)\n", len(tomlFiles))

	// Merge entries from all files
	// Deeper paths override shallower ones with same name
	type configEntry struct {
		entry   manifest.Entry
		relPath string // relative to root, for priority
		depth   int    // path depth for priority
	}

	var allEntries []configEntry
	for _, tf := range tomlFiles {
		relPath, err := filepath.Rel(root, tf)
		if err != nil {
			continue
		}

		entries, err := manifest.Read(tf, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[hookset] warning: reading %s: %v\n", relPath, err)
			continue
		}

		dir := filepath.Dir(relPath)
		depth := strings.Count(dir, string(filepath.Separator)) + 1

		for _, e := range entries {
			// Prepend subdirectory to match patterns
			adjusted := e
			if dir != "." && len(e.Match) > 0 {
				newMatch := make([]string, len(e.Match))
				for i, m := range e.Match {
					newMatch[i] = filepath.Join(dir, m)
				}
				adjusted.Match = newMatch
			}
			allEntries = append(allEntries, configEntry{
				entry:   adjusted,
				relPath: relPath,
				depth:   depth,
			})
		}
	}

	// Deduplicate by name, deeper wins
	seen := map[string]configEntry{}
	for _, ce := range allEntries {
		existing, exists := seen[ce.entry.Name]
		if !exists || ce.depth > existing.depth {
			seen[ce.entry.Name] = ce
		}
	}

	// Convert back to slice
	entries := make([]manifest.Entry, 0, len(seen))
	for _, ce := range seen {
		entries = append(entries, ce.entry)
	}

	// Print summary
	fmt.Printf("[hookset] Merged into %d hook(s)\n", len(entries))
	for _, e := range entries {
		fmt.Printf("  - %s (%s)\n", e.Name, e.Event)
	}

	return entries, nil
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
