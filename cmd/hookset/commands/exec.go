package commands

import (
	"fmt"
	"os"
	"strings"

	internexec "github.com/bulga138/hookset/internal/exec"
	"github.com/spf13/cobra"
)

var (
	execMatches    []string
	execAllowLarge bool
	execNoStash    bool
	execSummary    bool
)

var execCmd = &cobra.Command{
	Use:   "exec [flags] -- <command>",
	Short: "Run a command against staged files (called by git, not directly by users)",
	Long: `hookset exec is the staging engine. Git calls it via the hook command written
by hookset init. You rarely need to call it directly.

It:
  1. Collects staged files (git diff --cached)
  2. Filters by --match patterns (git ls-files)
  3. Stashes unstaged changes (--keep-index)
  4. Runs the command with matched files as arguments
  5. Re-stages any files modified by the command
  [WARN]  NOTE: Step 5 re-stages the ENTIRE file for any file modified by the
     command. If you stage only part of a file (partial staging), the rest
     of the file will be included in the re-stage. Use --no-stash only if
     you need to preserve unstaged changes alongside partial staging.
  6. Pops the stash
  7. Exits with the command's exit code

Environment:
  HOOKSET_SKIP   Comma-separated hook names to skip (e.g., HOOKSET_SKIP=eslint,prettier)

Example (as written into git config by hookset init):
  hookset exec --match "*.ts" --match "*.js" -- npx eslint --cache --fix`,
	RunE: func(cmd *cobra.Command, args []string) error {
		command := findDashDash()
		if command == nil {
			return cmd.Usage()
		}

		// Check HOOKSET_SKIP - extract hook name from command for filtering
		// The hook name is typically the first --match pattern or derived from the command
		hookName := extractHookName(execMatches, command)
		if hookName != "" {
			skipEnv := os.Getenv("HOOKSET_SKIP")
			if skipEnv != "" {
				skipped := strings.Split(skipEnv, ",")
				for _, s := range skipped {
					if strings.TrimSpace(s) == hookName {
						fmt.Printf("[hookset] Skipping %s (HOOKSET_SKIP)\n", hookName)
						os.Exit(0)
						return nil
					}
				}
			}
		}

		code := internexec.Run(internexec.Options{
			Patterns:   execMatches,
			Command:    command,
			Verbose:    verbose,
			AllowLarge: execAllowLarge,
			NoStash:    execNoStash,
			Summary:    execSummary,
		})
		os.Exit(code)
		return nil
	},
}

// extractHookName tries to extract a meaningful hook name for HOOKSET_SKIP matching.
// It looks at the command to determine the hook name.
func extractHookName(matches []string, command []string) string {
	if len(matches) > 0 {
		// Use the first match pattern as a hint - strip glob to get base name
		// e.g., "*.ts" -> "ts", "packages/frontend/*.ts" -> "frontend-ts"
		for _, m := range matches {
			// Return the pattern as-is for matching
			return m
		}
	}
	// Fallback to command name
	if len(command) > 0 {
		return command[0]
	}
	return ""
}

func init() {
	execCmd.Flags().StringArrayVar(&execMatches, "match", nil,
		"File pattern (repeatable). Empty = match all staged files.")
	execCmd.Flags().BoolVar(&execAllowLarge, "allow-large", false,
		"Suppress warning for staged files >10 MB")
	execCmd.Flags().BoolVar(&execNoStash, "no-stash", false,
		"Disable stash/pop cycle (escape hatch for Windows locking issues)")
	execCmd.Flags().BoolVar(&execSummary, "summary", false,
		"Print summary table after hook runs")
	rootCmd.AddCommand(execCmd)
}
