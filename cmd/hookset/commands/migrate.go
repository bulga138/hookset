package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/bulga138/hookset/internal/git"
	"github.com/bulga138/hookset/internal/manifest"
	"github.com/bulga138/hookset/internal/migrate"
	"github.com/spf13/cobra"
)

var (
	migrateFrom   string
	migrateDryRun bool
	migrateYes    bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Convert an existing hook setup to .hookset.toml",
	Long: `hookset migrate reads your current hook configuration (lint-staged, husky,
or lefthook) and produces equivalent .hookset.toml entries.

Examples:
  hookset migrate --from lint-staged --dry-run
  hookset migrate --from husky --yes
  hookset migrate --from lefthook --yes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if migrateFrom == "" {
			return fmt.Errorf("--from is required (lint-staged, husky, or lefthook)")
		}

		root, err := git.RepoRoot()
		if err != nil {
			return err
		}

		source := migrate.Source(migrateFrom)
		result, err := migrate.Migrate(source, root)
		if err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}

		// Print warnings.
		for _, w := range result.Warnings {
			fmt.Fprintln(os.Stderr, "[hookset] warning:", w)
		}

		if len(result.Entries) == 0 {
			fmt.Println("[hookset] No hook entries found to migrate.")
			return nil
		}

		// Show what would be written.
		fmt.Printf("[hookset] Migrating %d hook(s) from %s:\n\n", len(result.Entries), migrateFrom)
		for _, e := range result.Entries {
			fmt.Printf("  [[hooks]]\n")
			fmt.Printf("  name    = %q\n", e.Name)
			fmt.Printf("  event   = %q\n", e.Event)
			if len(e.Match) > 0 {
				fmt.Printf("  match   = [%s]\n", joinQuoted(e.Match))
			}
			fmt.Printf("  command = %q\n\n", e.Command)
		}

		if migrateDryRun {
			fmt.Println("[hookset] Dry run — no files written. Remove --dry-run to apply.")
			return nil
		}

		if !migrateYes {
			fmt.Print("Write to .hookset.toml and run hookset init? [y/N] ")
			var answer string
			if _, err := fmt.Scanln(&answer); err != nil {
				// EOF or error - treat as no answer
				answer = ""
			}
			if strings.ToLower(strings.TrimSpace(answer)) != "y" {
				fmt.Println("[hookset] Aborted.")
				return nil
			}
		}

		// Write manifest.
		tomlPath := root + "/" + manifest.Filename
		for _, e := range result.Entries {
			if err := manifest.Upsert(tomlPath, e); err != nil {
				return fmt.Errorf("writing manifest: %w", err)
			}
		}
		fmt.Printf("[hookset] Wrote %s\n", manifest.Filename)

		// Run init to install hooks.
		fmt.Println("[hookset] Running hookset init…")
		return runInit()
	},
}

func init() {
	migrateCmd.Flags().StringVar(&migrateFrom, "from", "",
		"Source tool: lint-staged, husky, lefthook")
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false,
		"Print what would be migrated without writing any files")
	migrateCmd.Flags().BoolVar(&migrateYes, "yes", false,
		"Write .hookset.toml and run hookset init without prompting")
	rootCmd.AddCommand(migrateCmd)
}

func joinQuoted(ss []string) string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(out, ", ")
}

// runInit re-uses the init command logic programmatically.
func runInit() error {
	return initCmd.RunE(initCmd, nil)
}
