package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	internexec "github.com/bulga138/hookset/internal/exec"
	"github.com/spf13/cobra"
)

var (
	execMatches    []string
	execName       string
	execEvent      string
	execAllowLarge bool
	execNoStash    bool
	execSummary    bool
	execCwd        string
	execFailText   string
)

// passthroughEvents is the set of git hook events that pass positional args to
// the hook script rather than operating on staged files. When --event is set to
// one of these, hookset exec forwards its trailing command args directly to the
// user command instead of going through the staging engine.
var passthroughEvents = map[string]bool{
	"commit-msg":         true,
	"prepare-commit-msg": true,
	"pre-rebase":         true,
	"post-checkout":      true,
	"post-merge":         true,
	"post-rewrite":       true,
	"applypatch-msg":     true,
}

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

For arg-style hooks (commit-msg, prepare-commit-msg, pre-rebase, post-checkout,
post-merge, post-rewrite, applypatch-msg) pass --event <event>. hookset exec will
bypass the staging engine and run the command with git's positional arguments.

Environment:
  HOOKSET=0            Disable all hooks unconditionally.
  HOOKSET_SKIP         Comma-separated hook names to skip (e.g., HOOKSET_SKIP=eslint,prettier)
  HOOKSET_ONLY         Comma-separated hook names to run; all others are skipped.
  HOOKSET_SKIP_IN_CI=1 Skip all hooks when a CI environment is detected.

Example (as written into git config by hookset init):
  hookset exec --match "*.ts" --match "*.js" -- npx eslint --cache --fix`,
	RunE: func(cmd *cobra.Command, args []string) error {
		command := findDashDash()
		if command == nil {
			return cmd.Usage()
		}

		// ── Global kill-switches (evaluated before per-hook skip logic) ──────────

		// HOOKSET=0  — disable all hooks unconditionally.
		if os.Getenv("HOOKSET") == "0" {
			fmt.Println("[hookset] All hooks disabled (HOOKSET=0)")
			return nil
		}

		// HOOKSET_SKIP_IN_CI=1  — skip all hooks when running in a CI environment.
		// Detected via standard CI env vars set by GitHub Actions, GitLab CI,
		// CircleCI, Jenkins, Travis CI, and similar platforms.
		if os.Getenv("HOOKSET_SKIP_IN_CI") == "1" && isCI() {
			fmt.Println("[hookset] Skipping hooks in CI (HOOKSET_SKIP_IN_CI=1)")
			return nil
		}

		// ── Per-hook skip / only logic ───────────────────────────────────────────

		// Check HOOKSET_SKIP using the explicit --name value injected by buildWrapperCommand.
		hookName := execName
		if hookName == "" {
			// Fallback for manual invocations that don't supply --name.
			hookName = extractHookName(execMatches, command)
		}

		// HOOKSET_ONLY=<name>,<name>  — run only the named hooks; skip everything else.
		if hookName != "" {
			onlyEnv := os.Getenv("HOOKSET_ONLY")
			if onlyEnv != "" {
				allowed := strings.Split(onlyEnv, ",")
				found := false
				for _, a := range allowed {
					if strings.TrimSpace(a) == hookName {
						found = true
						break
					}
				}
				if !found {
					fmt.Printf("[hookset] Skipping %s (not in HOOKSET_ONLY)\n", hookName)
					return nil
				}
			}
		}

		if hookName != "" {
			skipEnv := os.Getenv("HOOKSET_SKIP")
			if skipEnv != "" {
				skipped := strings.Split(skipEnv, ",")
				for _, s := range skipped {
					if strings.TrimSpace(s) == hookName {
						fmt.Printf("[hookset] Skipping %s (HOOKSET_SKIP)\n", hookName)
						return nil
					}
				}
			}
		}

		// Passthrough mode: bypass staging engine, run command with git's args.
		if execEvent != "" && passthroughEvents[execEvent] {
			os.Exit(runPassthrough(command))
			return nil
		}

		code := internexec.Run(internexec.Options{
			Patterns:   execMatches,
			Command:    command,
			Event:      execEvent,
			Verbose:    verbose,
			AllowLarge: execAllowLarge,
			NoStash:    execNoStash,
			Summary:    execSummary,
			Cwd:        execCwd,
			FailText:   execFailText,
		})
		os.Exit(code)
		return nil
	},
}

// runPassthrough executes command directly without the staging engine.
// It inherits stdin/stdout/stderr from the parent process so that tools
// like commitlint can read the commit message file and print diagnostics.
// Any extra args appended after "--" (which were "$@" in the wrapper sh -c)
// are included as trailing arguments to the command.
func runPassthrough(command []string) int {
	if len(command) == 0 {
		return 0
	}
	c := exec.Command(command[0], command[1:]...) //nolint:gosec
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "[hookset] passthrough exec error: %v\n", err)
		return 1
	}
	return 0
}

// extractHookName is a best-effort fallback for manual invocations that omit --name.
// It derives a name from the command binary (e.g. ["npx", "eslint", ...] → "eslint").
// When --name is supplied (the normal case via buildWrapperCommand) this is not called.
func extractHookName(matches []string, command []string) string {
	// Walk command args to find the first non-flag, non-"npx"/"yarn"/"pnpm" token.
	launchers := map[string]bool{"npx": true, "yarn": true, "pnpm": true, "bunx": true}
	for _, token := range command {
		if strings.HasPrefix(token, "-") {
			continue
		}
		if launchers[token] {
			continue
		}
		return token
	}
	return ""
}

// isCI returns true when the process is running inside a known CI environment.
// It checks the standard env vars set by GitHub Actions, GitLab CI, CircleCI,
// Travis CI, Jenkins, Buildkite, Azure Pipelines, and Bitbucket Pipelines.
func isCI() bool {
	ciVars := []string{
		"CI",           // GitHub Actions, CircleCI, Travis CI, GitLab CI, etc.
		"CONTINUOUS_INTEGRATION",
		"GITLAB_CI",
		"JENKINS_URL",
		"TRAVIS",
		"CIRCLECI",
		"BUILDKITE",
		"TF_BUILD",     // Azure Pipelines
		"BITBUCKET_BUILD_NUMBER",
	}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}

func init() {
	execCmd.Flags().StringArrayVar(&execMatches, "match", nil,
		"File pattern (repeatable). Empty = match all staged files.")
	execCmd.Flags().StringVar(&execName, "name", "",
		"Hook name used for HOOKSET_SKIP matching (injected by hookset init)")
	execCmd.Flags().StringVar(&execEvent, "event", "",
		"Git hook event name (injected by hookset init for arg-style hooks)")
	execCmd.Flags().BoolVar(&execAllowLarge, "allow-large", false,
		"Suppress warning for staged files >10 MB")
	execCmd.Flags().BoolVar(&execNoStash, "no-stash", false,
		"Disable stash/pop cycle (escape hatch for Windows locking issues)")
	execCmd.Flags().BoolVar(&execSummary, "summary", false,
		"Print summary table after hook runs")
	execCmd.Flags().StringVar(&execCwd, "cwd", "",
		"Working directory for the command, relative to repo root")
	execCmd.Flags().StringVar(&execFailText, "fail-text", "",
		"Custom failure message shown when this hook fails")
	rootCmd.AddCommand(execCmd)
}
