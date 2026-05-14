package commands

import (
	"os"

	internexec "github.com/bulga138/hookset/internal/exec"
	"github.com/spf13/cobra"
)

var (
	execMatches    []string
	execAllowLarge bool
	execNoStash    bool
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

Example (as written into git config by hookset init):
  hookset exec --match "*.ts" --match "*.js" -- npx eslint --cache --fix`,
	RunE: func(cmd *cobra.Command, args []string) error {
		command := findDashDash()
		if command == nil {
			return cmd.Usage()
		}

		code := internexec.Run(internexec.Options{
			Patterns:   execMatches,
			Command:    command,
			Verbose:    verbose,
			AllowLarge: execAllowLarge,
			NoStash:    execNoStash,
		})
		os.Exit(code)
		return nil
	},
}

func init() {
	execCmd.Flags().StringArrayVar(&execMatches, "match", nil,
		"File pattern (repeatable). Empty = match all staged files.")
	execCmd.Flags().BoolVar(&execAllowLarge, "allow-large", false,
		"Suppress warning for staged files >10 MB")
	execCmd.Flags().BoolVar(&execNoStash, "no-stash", false,
		"Disable stash/pop cycle (escape hatch for Windows locking issues)")
	rootCmd.AddCommand(execCmd)
}
