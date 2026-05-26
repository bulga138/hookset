package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bulga138/hookset/internal/git"
	"github.com/bulga138/hookset/internal/gitconfig"
	"github.com/bulga138/hookset/internal/manifest"
	"github.com/bulga138/hookset/internal/orchestrator"
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
	initDirect        bool
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
		// Enforce git ≥ 2.54 floor for all init operations.
		if err := requireGit254(); err != nil {
			return err
		}

		// 1. Handle Uninstall first
		if initUninstall {
			return runUninstall()
		}

		root, err := git.RepoRoot()
		if err != nil {
			return err
		}

		var entries []manifest.Entry
		var hooksetConf manifest.HooksetConf

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
				var mf *manifest.File
				mf, entries, err = manifest.ReadFile(tomlPath, opts)
				if err != nil {
					return fmt.Errorf("reading manifest: %w", err)
				}
				hooksetConf = mf.Hookset

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

		parallelMode := orchestrator.IsEnabled(hooksetConf.Experimental)
		if parallelMode && !initDirect {
			fmt.Fprintln(os.Stderr, orchestrator.AlphaBanner)
			if err := installParallelHooks(entries, actions); err != nil {
				return err
			}
		} else {
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
						Command: buildWrapperCommand(entry, initDirect),
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
		}

		fmt.Println("[hookset] Done. Verify with: hookset list")

		// Store the version that installed these hooks
		storeVersion()

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
		if strings.Contains(h.Command, "hookset exec") || strings.Contains(h.Command, "hookset exec-event") {
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

// installParallelHooks implements D1.c: per-entry [hook] blocks are written with
// enabled = false so `git hook list` still shows them by name, while a single
// [hook "hookset-<event>"] coordinator entry (enabled = true) calls
// `hookset exec-event <event>` and handles parallel/serial dispatch.
//
// This means:
//   - `git hook list pre-commit` shows: eslint, prettier, typecheck, hookset-pre-commit
//   - Only hookset-pre-commit fires; the others are transparent documentation.
func installParallelHooks(entries []manifest.Entry, actions map[int]init_view.HookAction) error {
	// Phase 1: Remove all existing hookset-managed entries (idempotent).
	existingNames, err := gitconfig.GetAllHookNames(git.ScopeLocal)
	if err != nil {
		return fmt.Errorf("listing existing hooks: %w", err)
	}
	for _, name := range existingNames {
		h, err := gitconfig.GetHook(name, git.ScopeLocal)
		if err != nil {
			continue
		}
		if strings.Contains(h.Command, "hookset exec") || strings.Contains(h.Command, "hookset exec-event") {
			if err := gitconfig.RemoveHook(name, git.ScopeLocal); err != nil {
				return fmt.Errorf("removing hook %q: %w", name, err)
			}
		}
	}

	// Phase 2: Write per-entry blocks with enabled=false (visibility only).
	// These let `git hook list <event>` show the real hook names for inspection.
	seenEvents := map[string]bool{}
	var installEvents []string
	for i, e := range entries {
		if actions[i] != init_view.ActionInstall {
			continue
		}
		h := gitconfig.Hook{
			Name:    e.Name,
			Event:   e.Event,
			Command: buildWrapperCommand(e, initDirect),
			Matches: e.Match,
			Enabled: false, // D1.c: git will NOT fire these directly
		}
		if err := gitconfig.AddHook(h, git.ScopeLocal); err != nil {
			return fmt.Errorf("installing hook entry %q: %w", e.Name, err)
		}
		fmt.Printf("  • Registered (visible): %s [%s] — disabled, run via coordinator\n", e.Name, e.Event)
		if !seenEvents[e.Event] {
			seenEvents[e.Event] = true
			installEvents = append(installEvents, e.Event)
		}
	}

	// Phase 3: Write one coordinator entry per event (enabled=true, fires by git).
	for _, event := range installEvents {
		h := gitconfig.Hook{
			Name:    "hookset-" + event,
			Event:   event,
			Command: buildExecEventCommand(event),
			Enabled: true,
		}
		if err := gitconfig.AddHook(h, git.ScopeLocal); err != nil {
			return fmt.Errorf("installing coordinator for event %q: %w", event, err)
		}
		fmt.Printf("  ✓ Installed coordinator: hookset-%s → hookset exec-event %s\n", event, event)
	}

	// Phase 4: Explicit removes for ActionUninstall entries.
	for i, e := range entries {
		if actions[i] == init_view.ActionUninstall {
			if err := gitconfig.RemoveHook(e.Name, git.ScopeLocal); err != nil {
				if !strings.Contains(err.Error(), "No such section") {
					return fmt.Errorf("removing hook %q: %w", e.Name, err)
				}
			}
			fmt.Printf("  × Removed: %s\n", e.Name)
		}
	}
	return nil
}

// buildExecEventCommand constructs the self-checking per-event wrapper that git
// calls when parallel mode is active.  It routes all hooks for the event through
// `hookset exec-event <event>` which handles parallel/serial dispatch.
func buildExecEventCommand(event string) string {
	const installMsg = "hookset is not installed. See: https://bulga138.github.io/hookset/"
	inner := "hookset exec-event " + shellQuote(event) + ` "$@"`
	return buildPresenceCheckPassthrough(installMsg, inner)
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
	initCmd.Flags().BoolVar(&initDirect, "direct", false,
		"Generate simple commands without hookset wrapper (no file filtering, no stash)")
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

// argStyleHooks is the set of git hook events that receive positional arguments
// from git rather than operating on the set of staged files. These hooks must
// bypass the staging engine and forward git's arguments verbatim.
var argStyleHooks = map[string]bool{
	"commit-msg":         true,
	"prepare-commit-msg": true,
	"pre-rebase":         true,
	"post-checkout":      true,
	"post-merge":         true,
	"post-rewrite":       true,
	"applypatch-msg":     true,
}

// buildWrapperCommand constructs the self-checking hook command stored in git config.
//
// All hooks (filtering and non-filtering) are routed through hookset exec so
// that the stash/restage cycle is uniform. The user's command string is passed
// verbatim via sh -c to preserve embedded quoting and shell metacharacters.
//
// Passthrough hooks (arg-style events or entries with passthrough=true) receive
// git's positional arguments via "$@" — the staging engine is bypassed.
//
// If direct is true, returns the command without any hookset wrapper.
//
// Rendered form (Unix, normal):
//
//	sh -c 'command -v hookset >/dev/null 2>&1 || { printf "%s\n" "<msg>" >&2; exit 1; }; exec hookset exec --name n [--match p]... -- sh -c <cmd>'
//
// Rendered form (Unix, passthrough):
//
//	sh -c 'command -v hookset >/dev/null 2>&1 || { printf "%s\n" "<msg>" >&2; exit 1; }; exec hookset exec --name n --event <event> -- sh -c <cmd> "$@"' --
func buildWrapperCommand(entry manifest.Entry, direct bool) string {
	if direct {
		return entry.Command
	}

	const installMsg = "hookset is not installed. See: https://bulga138.github.io/hookset/"

	isPassthrough := entry.Passthrough || argStyleHooks[entry.Event]

	// Build: hookset exec --name <name> [--event <event>] [--match <pat>]... -- sh -c <command> ["$@"]
	// The user's command is wrapped in "sh -c '...'" so that:
	//   1. Shell metacharacters (&&, ;, pipes, subshells) work as expected.
	//   2. No argument-splitting mangling from strings.Fields.
	var execParts []string
	execParts = append(execParts, "hookset", "exec")
	execParts = append(execParts, "--name", shellQuote(entry.Name))
	if isPassthrough {
		execParts = append(execParts, "--event", shellQuote(entry.Event))
	}
	for _, m := range entry.Match {
		execParts = append(execParts, "--match", shellQuote(m))
	}
	if entry.Cwd != "" {
		execParts = append(execParts, "--cwd", shellQuote(entry.Cwd))
	}
	if entry.FailText != "" {
		execParts = append(execParts, "--fail-text", shellQuote(entry.FailText))
	}
	execParts = append(execParts, "--")
	execParts = append(execParts, "sh", "-c", shellQuote(entry.Command))
	if isPassthrough {
		// Append "$@" so git's positional args are forwarded to the user command.
		// The trailing " --" is the argv[0] placeholder required by sh -c.
		execParts = append(execParts, `"$@"`)
	}

	inner := strings.Join(execParts, " ")
	if isPassthrough {
		// The outer sh -c receives git args as $1, $2, … via the trailing "--".
		// We embed them into the inner call via "$@".
		return buildPresenceCheckPassthrough(installMsg, inner)
	}
	return buildPresenceCheck(installMsg, inner)
}

// dqEscape escapes a string for safe embedding in a double-quoted POSIX shell string.
// Inside "..." only ", \, $, and ` are special and must be escaped.
func dqEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, `$`, `\$`)
	s = strings.ReplaceAll(s, "`", "\\`")
	return s
}

// buildPresenceCheck wraps execCmd in a sh -c that aborts with installMsg
// if hookset is not found on PATH.
func buildPresenceCheck(installMsg, execCmd string) string {
	safeMsg := dqEscape(installMsg)
	safeCmd := dqEscape(execCmd)
	return fmt.Sprintf(
		`sh -c "command -v hookset >/dev/null 2>&1 || { printf \"%%s\n\" \"%s\" >&2; exit 1; }; exec %s"`,
		safeMsg, safeCmd,
	)
}

// buildPresenceCheckPassthrough is like buildPresenceCheck but threads git's
// positional arguments ($1, $2, …) through to the inner hookset exec call via "$@".
// The trailing " --" in the rendered string is the argv[0] placeholder for sh -c.
func buildPresenceCheckPassthrough(installMsg, execCmd string) string {
	safeMsg := dqEscape(installMsg)
	safeCmd := dqEscape(execCmd)
	return fmt.Sprintf(
		`sh -c "command -v hookset >/dev/null 2>&1 || { printf \"%%s\n\" \"%s\" >&2; exit 1; }; exec %s" --`,
		safeMsg, safeCmd,
	)
}

func shellQuote(s string) string {
	if !strings.ContainsAny(s, " \t\"'\\") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// versionKey is the git config key where we store the hookset version that installed the hooks.
const versionKey = "hookset.version"

// storeVersion stores the current hookset version in git config.
func storeVersion() {
	version := buildVersion()
	git.Run("config", string(git.ScopeLocal), versionKey, version)
}

// getStoredVersion returns the stored hookset version from git config, or empty if not set.
func getStoredVersion() string {
	r := git.Run("config", string(git.ScopeLocal), versionKey)
	if r.OK {
		return r.Stdout
	}
	return ""
}

// CheckAndUpdateHooks checks if the hookset version has changed and re-runs init if needed.
// This ensures hooks are updated when hookset is upgraded to a new version with different settings.
func CheckAndUpdateHooks() {
	// Only check inside a git repo
	root, err := git.RepoRoot()
	if err != nil {
		return
	}

	// Check if there are any hooks managed by hookset
	hooks, err := gitconfig.GetAllHookNames(git.ScopeLocal)
	if err != nil || len(hooks) == 0 {
		return
	}

	// Check if any hook is managed by hookset (contains "hookset exec")
	hasHooksetHooks := false
	for _, name := range hooks {
		h, err := gitconfig.GetHook(name, git.ScopeLocal)
		if err == nil && strings.Contains(h.Command, "hookset exec") {
			hasHooksetHooks = true
			break
		}
	}
	if !hasHooksetHooks {
		return
	}

	// Compare versions
	currentVersion := buildVersion()
	storedVersion := getStoredVersion()

	if storedVersion != "" && storedVersion != currentVersion {
		fmt.Println("[hookset] Version changed from", storedVersion, "to", currentVersion, "- updating hooks...")
		// Re-run init with existing manifest to update hooks
		// We need to re-read the manifest and re-install hooks
		tomlPath := filepath.Join(root, manifest.Filename)
		if _, err := os.Stat(tomlPath); err == nil {
			opts := manifest.ReadOptions{}
			entries, err := manifest.Read(tomlPath, opts)
			if err == nil && len(entries) > 0 {
				// Re-install all hooks with new wrapper commands
				for _, entry := range entries {
					// Remove old hook
					_ = gitconfig.RemoveHook(entry.Name, git.ScopeLocal)
					// Add new hook with updated wrapper
					h := gitconfig.Hook{
						Name:    entry.Name,
						Event:   entry.Event,
						Command: buildWrapperCommand(entry, false),
						Matches: entry.Match,
						Enabled: true,
					}
					_ = gitconfig.AddHook(h, git.ScopeLocal)
				}
				// Update stored version
				storeVersion()
				fmt.Println("[hookset] Hooks updated successfully.")
			}
		}
	}
}
