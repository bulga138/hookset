package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

// hookEventOrder defines the preferred display order for git hook events.
// Events not in this list are sorted alphabetically after the known ones.
var hookEventOrder = []string{
	"pre-commit",
	"pre-push",
	"commit-msg",
	"prepare-commit-msg",
	"pre-rebase",
	"post-checkout",
	"post-merge",
	"post-rewrite",
}

var (
	listEventFlag     string
	listJSONFlag      bool
	listPorcelainFlag bool
)

// listRow is a flattened hook record used for rendering.
type listRow struct {
	Name    string   `json:"name"`
	Event   string   `json:"event"`
	Scope   string   `json:"scope"`
	Enabled bool     `json:"enabled"`
	Matches []string `json:"matches,omitempty"`
	Command string   `json:"command"`
}

var listCmd = &cobra.Command{
	Use:   "list [event]",
	Short: "List configured hooks",
	Long: `List all hooks configured in git config (local and global scopes).

Hooks are grouped and sorted by event in canonical order:
  pre-commit, pre-push, commit-msg, prepare-commit-msg,
  pre-rebase, post-checkout, post-merge, post-rewrite,
  then remaining events alphabetically.

Within each event, hooks are sorted alphabetically by name.

Examples:
  hookset list                  # table view (all hooks)
  hookset list pre-commit       # filter by event
  hookset list --json           # machine-readable JSON array
  hookset list --porcelain      # tab-separated, no header (for scripting)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filterEvent := listEventFlag
		if len(args) > 0 {
			filterEvent = args[0]
		}

		rows, err := collectRows(filterEvent)
		if err != nil {
			return err
		}

		switch {
		case listJSONFlag:
			return renderJSON(rows)
		case listPorcelainFlag:
			renderPorcelain(rows)
		default:
			renderTable(rows)
			if drift := detectDrift(); drift != nil {
				renderDrift(drift)
			}
		}
		return nil
	},
}

// driftReport holds entries that are in one source but not the other.
type driftReport struct {
	ManifestOnly []string // declared in .hookset.toml but not in git config
	GitOnly      []string // in git config but not in .hookset.toml
}

// detectDrift compares installed gitconfig hook names against .hookset.toml.
// Returns nil if there is no manifest or no drift. Coordinator hooks
// (prefixed with "hookset-") are always excluded from drift reports since
// they are internal to hookset's parallel mode.
func detectDrift() *driftReport {
	root, err := git.RepoRoot()
	if err != nil {
		return nil
	}
	tomlPath := filepath.Join(root, manifest.Filename)
	if _, err := os.Stat(tomlPath); os.IsNotExist(err) {
		return nil
	}
	entries, err := manifest.Read(tomlPath, manifest.ReadOptions{})
	if err != nil {
		return nil
	}

	manifestNames := make(map[string]bool, len(entries))
	for _, e := range entries {
		manifestNames[e.Name] = true
	}

	configHooks, err := gitconfig.GetHooks("", git.ScopeLocal)
	if err != nil {
		return nil
	}
	configNames := make(map[string]bool, len(configHooks))
	for _, h := range configHooks {
		if !strings.HasPrefix(h.Name, "hookset-") {
			configNames[h.Name] = true
		}
	}

	var rep driftReport
	for name := range manifestNames {
		if !configNames[name] {
			rep.ManifestOnly = append(rep.ManifestOnly, name)
		}
	}
	for name := range configNames {
		if !strings.Contains(name, "hookset") && !manifestNames[name] {
			rep.GitOnly = append(rep.GitOnly, name)
		}
	}
	sort.Strings(rep.ManifestOnly)
	sort.Strings(rep.GitOnly)
	if len(rep.ManifestOnly) == 0 && len(rep.GitOnly) == 0 {
		return nil
	}
	return &rep
}

// collectRows gathers hooks from local and global git config, deduplicates
// (local wins over global for same name), and sorts them.
func collectRows(filterEvent string) ([]listRow, error) {
	byName := map[string]listRow{}

	for _, scope := range []git.Scope{git.ScopeGlobal, git.ScopeLocal} {
		hooks, err := gitconfig.GetHooks(filterEvent, scope)
		if err != nil {
			continue // scope may simply have no hooks
		}
		for _, h := range hooks {
			// Local overrides global for the same hook name.
			byName[h.Name] = listRow{
				Name:    h.Name,
				Event:   h.Event,
				Scope:   h.Scope,
				Enabled: h.Enabled,
				Matches: h.Matches,
				Command: h.Command,
			}
		}
	}

	rows := make([]listRow, 0, len(byName))
	for _, r := range byName {
		rows = append(rows, r)
	}
	sortRows(rows)
	return rows, nil
}

// sortRows sorts by canonical event order, then alphabetically by name.
func sortRows(rows []listRow) {
	eventRank := make(map[string]int, len(hookEventOrder))
	for i, e := range hookEventOrder {
		eventRank[e] = i
	}
	sort.SliceStable(rows, func(i, j int) bool {
		ri, rj := rows[i], rows[j]
		oi, okI := eventRank[ri.Event]
		oj, okJ := eventRank[rj.Event]
		switch {
		case okI && okJ:
			if oi != oj {
				return oi < oj
			}
		case okI:
			return true
		case okJ:
			return false
		default:
			if ri.Event != rj.Event {
				return ri.Event < rj.Event
			}
		}
		return ri.Name < rj.Name
	})
}

// renderTable prints a human-readable aligned table grouped by event.
func renderTable(rows []listRow) {
	if len(rows) == 0 {
		fmt.Println("[hookset] No hooks configured.")
		return
	}

	// Compute column widths.
	nameW, eventW, scopeW := len("NAME"), len("EVENT"), len("SCOPE")
	for _, r := range rows {
		if len(r.Name) > nameW {
			nameW = len(r.Name)
		}
		if len(r.Event) > eventW {
			eventW = len(r.Event)
		}
		if len(r.Scope) > scopeW {
			scopeW = len(r.Scope)
		}
	}

	header := fmt.Sprintf("%-*s  %-*s  %-*s  %s",
		nameW, "NAME",
		eventW, "EVENT",
		scopeW, "SCOPE",
		"COMMAND",
	)
	fmt.Println(header)
	fmt.Println(strings.Repeat("─", len(header)+20))

	lastEvent := ""
	for _, r := range rows {
		if r.Event != lastEvent {
			if lastEvent != "" {
				fmt.Println()
			}
			lastEvent = r.Event
		}
		status := " "
		if !r.Enabled {
			status = "✗"
		}
		matchNote := ""
		if len(r.Matches) > 0 {
			matchNote = fmt.Sprintf("  [match: %s]", strings.Join(r.Matches, ", "))
		}
		fmt.Printf("%s %-*s  %-*s  %-*s  %s%s\n",
			status,
			nameW, r.Name,
			eventW, r.Event,
			scopeW, r.Scope,
			r.Command, matchNote,
		)
	}
}

// renderDrift prints a drift warning section below the main table.
func renderDrift(d *driftReport) {
	fmt.Println()
	fmt.Println("── Drift detected ──────────────────────────────────────────────")
	for _, name := range d.ManifestOnly {
		fmt.Printf("  ⚠  %-30s  in .hookset.toml but NOT installed — run: hookset init\n", name)
	}
	for _, name := range d.GitOnly {
		fmt.Printf("  ⚠  %-30s  in git config but NOT in .hookset.toml\n", name)
	}
}

// renderJSON prints a JSON array of hook records.
func renderJSON(rows []listRow) error {
	out, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

// renderPorcelain prints one tab-separated line per hook (no header).
// Format: name\tevent\tscope\tenabled\tcommand
func renderPorcelain(rows []listRow) {
	for _, r := range rows {
		enabled := "true"
		if !r.Enabled {
			enabled = "false"
		}
		fmt.Printf("%s\t%s\t%s\t%s\t%s\n", r.Name, r.Event, r.Scope, enabled, r.Command)
	}
}

func init() {
	removeCmd.Flags().BoolVar(&removeManifest, "manifest", false, "Also remove from .hookset.toml")
	removeCmd.Flags().BoolVar(&removeGlobal, "global", false, "Target global config")
	disableCmd.Flags().BoolVar(&toggleGlobal, "global", false, "Target global config")
	enableCmd.Flags().BoolVar(&toggleGlobal, "global", false, "Target global config")
	listCmd.Flags().StringVar(&listEventFlag, "event", "", "Filter by hook event")
	listCmd.Flags().BoolVar(&listJSONFlag, "json", false, "Output as JSON array")
	listCmd.Flags().BoolVar(&listPorcelainFlag, "porcelain", false, "Output tab-separated (for scripting)")

	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(disableCmd)
	rootCmd.AddCommand(enableCmd)
	rootCmd.AddCommand(listCmd)
}

// GetListCommand returns the list command for external use.
func GetListCommand() *cobra.Command {
	return listCmd
}

// RunList executes the list command directly. Used by main when no subcommand is given.
func RunList() error {
	// Use the same logic as listCmd.RunE but call it directly
	event := listEventFlag
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
}
