// Package exec implements the hookset staging engine.
// It is the component Git calls directly via the hook command:
//
//	hookset exec --match "*.ts" --match "*.js" -- npx eslint --cache --fix
//
// Flow: collect staged files → filter by match → stash unstaged → run command
// → re-stage modified files → pop stash → propagate exit code.
package exec

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/bulga138/hookset/internal/git"
)

// Options configures a single hookset exec run.
type Options struct {
	// Patterns are the --match pathspecs (Git-native globs).
	Patterns []string
	// Command is the tool to run, e.g. ["npx", "eslint", "--cache", "--fix"].
	// When the command string (joined) contains a {token}, matched files are
	// NOT appended as trailing arguments — the token handles placement.
	Command []string
	// Event is the git hook event name (e.g. "pre-commit"). Used for {event}
	// token expansion and for bypassing the empty-files check on passthrough hooks.
	Event string
	// Verbose enables step-by-step output.
	Verbose bool
	// AllowLarge skips the >10 MB file warning.
	AllowLarge bool
	// NoStash disables the stash/pop cycle (escape hatch for Windows issues).
	NoStash bool
	// Summary enables summary table output after hook runs.
	Summary bool
	// Cwd sets the working directory for the command, relative to the repo root.
	// If empty, the command runs in the current directory (repo root).
	Cwd string
	// FailText is a human-readable message emitted on non-zero exit.
	// Supports {event}, {branch}, {staged_files} tokens.
	FailText string
}

const largeBytesThreshold = 10 * 1024 * 1024 // 10 MB
// Windows CreateProcess has a safe limit of ~8000 chars for the command line.
const windowsArgLimit = 7500

// Run executes the staging engine with the given options.
// Returns the exit code the caller should propagate (0 = success).
func Run(opts Options) int {
	log := newLogger(opts.Verbose)

	// ── 1. Collect staged files ───────────────────────────────────────────────

	staged := git.StagedFiles()
	log.step("Staged files (%d): %v", len(staged), staged)

	// ── 2. Guard: linked worktrees are not supported in v1 ───────────────────

	if git.IsLinkedWorktree() {
		fmt.Fprintln(os.Stderr, "[hookset] error: linked worktrees are not supported in v1 — skipping hook")
		fmt.Fprintln(os.Stderr, "         See https://bulga138.github.io/hookset/worktrees for details")
		return 1
	}

	// ── 3. Filter by match patterns ───────────────────────────────────────────

	matched, err := git.MatchStagedFiles(staged, opts.Patterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[hookset] error matching files: %v\n", err)
		return 1
	}
	log.step("Matched files (%d): %v", len(matched), matched)
	log.step("Excluded files (%d): %v", len(staged)-len(matched), exclude(staged, matched))

	// ── 4. Token expansion ────────────────────────────────────────────────────
	//
	// Expand {staged_files}, {staged_files_or_default}, {event}, {branch} in
	// the command string. Must happen BEFORE the empty-files early-return so
	// that {staged_files_or_default} gets a chance to expand to "." and the
	// command still runs even when no files matched.
	//
	// When any token is found, files are embedded via the token — do NOT also
	// append them as trailing arguments.

	branch := git.CurrentBranch()
	rawCmd := strings.Join(opts.Command, " ")
	expandedCmd, hadToken := expandTokens(rawCmd, matched, opts.Event, branch)

	// Rebuild command slice from the expanded string for the shell.
	// The user's command is already wrapped in sh -c by buildWrapperCommand so
	// we only need to pass it as a single string token here.
	finalCommand := opts.Command
	appendFiles := !hadToken // append matched files as args unless tokens did it
	if hadToken {
		// Replace the command slice with [sh, -c, expandedCmd].
		finalCommand = []string{"sh", "-c", expandedCmd}
		appendFiles = false
	}

	// ── 5. Empty-files early-return (AFTER token expansion, A9) ──────────────
	//
	// Skip when:
	//   a) no files matched AND the command has no {staged_files_or_default} token
	//      (that token already handles the empty-files case with "." fallback), OR
	//   b) no files staged at all (nothing to do regardless).
	//
	// Passthrough hooks (Event set to an arg-style event) bypass this check
	// entirely — they don't operate on staged files.

	isPassthrough := passthroughEvent(opts.Event)

	if !isPassthrough {
		if len(staged) == 0 {
			log.info("No staged files — skipping")
			return 0
		}
		if len(matched) == 0 && !hadToken {
			log.info("No files match patterns %v — skipping", opts.Patterns)
			return 0
		}
		// If token expansion happened but produced an empty {staged_files}
		// (and the command does not use {staged_files_or_default}), skip.
		if hadToken && len(matched) == 0 && !strings.Contains(rawCmd, "{staged_files_or_default}") {
			log.info("No files match patterns — skipping (token produced empty list)")
			return 0
		}
	}

	// ── 6. Warn on large files ────────────────────────────────────────────────

	if !opts.AllowLarge && len(matched) > 0 {
		if name, size := firstLargeFile(matched); name != "" {
			fmt.Fprintf(os.Stderr,
				"[hookset] warning: %s is %.1f MB — stashing large files may be slow.\n"+
					"         Pass --allow-large to suppress this warning.\n",
				name, float64(size)/1024/1024)
		}
	}

	// ── 7. Stash unstaged changes ─────────────────────────────────────────────

	stashCreated := false
	if !opts.NoStash && !isPassthrough && git.HasUnstagedChanges() {
		msg := fmt.Sprintf("hookset auto-stash %d", time.Now().UnixMilli())
		log.step("Stashing unstaged changes: %s", msg)
		if err := git.StashPushKeepIndex(msg); err != nil {
			fmt.Fprintf(os.Stderr, "[hookset] error: %v\n", err)
			fmt.Fprintln(os.Stderr, "         Your working tree has not been modified.")
			return 1
		}
		stashCreated = true
	}

	// Always pop stash, even on failure.
	exitCode := 0
	defer func() {
		if stashCreated {
			log.step("Popping stash")
			if err := git.StashPop(); err != nil {
				fmt.Fprintf(os.Stderr, "[hookset] warning: stash pop failed: %v\n", err)
				fmt.Fprintln(os.Stderr, "         Your changes are safe in `git stash list` (most recent entry).")
				fmt.Fprintln(os.Stderr, "         Recover with: git stash pop")
			}
		}
		if opts.Summary {
			printSummary(opts.Command, matched, exitCode == 0)
		}
	}()

	// ── 8. Run the tool ───────────────────────────────────────────────────────

	// Apply cwd if set — change directory before running the command.
	// Save origDir so we can restore for re-staging (step 9), since
	// classifyAfterRun and git add/rm --cached need repo-root-relative paths.
	origDir, _ := os.Getwd()
	if opts.Cwd != "" {
		if err := os.Chdir(opts.Cwd); err != nil {
			fmt.Fprintf(os.Stderr, "[hookset] error: cannot chdir to %q: %v\n", opts.Cwd, err)
			exitCode = 1
			return exitCode
		}
	}

	var filesForLog []string
	if appendFiles {
		filesForLog = matched
	}
	log.step("Running: %s %s", strings.Join(finalCommand, " "), strings.Join(filesForLog, " "))

	var code int
	if appendFiles {
		code = runCommand(finalCommand, matched, log)
	} else {
		code = runCommand(finalCommand, nil, log)
	}
	if code != 0 {
		exitCode = code
		if opts.FailText != "" {
			// Expand tokens in fail_text so messages can be context-aware.
			msg, _ := expandTokens(opts.FailText, matched, opts.Event, branch)
			fmt.Fprintf(os.Stderr, "[hookset] %s\n", msg)
		} else {
			fmt.Fprintf(os.Stderr, "[hookset] %s exited with code %d — commit aborted\n",
				opts.Command[0], code)
		}
		return exitCode
	}

	// ── 9. Re-stage modified or deleted files ─────────────────────────────────
	//
	// Restore original CWD first — classifyAfterRun uses os.Stat on
	// repo-root-relative paths, and git add/rm --cached also expect
	// repo-root context.

	if !isPassthrough && opts.Cwd != "" {
		_ = os.Chdir(origDir)
	}

	if !isPassthrough && len(matched) > 0 {
		toAdd, toRemove := classifyAfterRun(matched)
		log.step("Re-staging %d file(s), removing %d file(s)", len(toAdd), len(toRemove))
		if len(toAdd) > 0 {
			log.step("[WARN] Re-staging entire file(s): %v", toAdd)
			log.step("   Partial staging note: all changes in these files will be included")
		}

		if err := git.Add(toAdd); err != nil {
			fmt.Fprintf(os.Stderr, "[hookset] error re-staging files: %v\n", err)
			exitCode = 1
			return exitCode
		}
		if err := git.RmCached(toRemove); err != nil {
			fmt.Fprintf(os.Stderr, "[hookset] error removing deleted files from index: %v\n", err)
			exitCode = 1
			return exitCode
		}
	}

	log.info("Done ✓")

	return 0
}

// passthroughEvent returns true for git hook events that receive positional
// args from git rather than operating on staged files.
func passthroughEvent(event string) bool {
	switch event {
	case "commit-msg", "prepare-commit-msg", "pre-rebase",
		"post-checkout", "post-merge", "post-rewrite", "applypatch-msg":
		return true
	}
	return false
}

// printSummary prints a compact table of hook execution results.
func printSummary(command []string, files []string, success bool) {
	status := "✓"
	if !success {
		status = "✗"
	}
	fmt.Printf("[hookset] %s %s (%d files)\n", status, command[0], len(files))
}

// runCommand executes the tool against the matched files.
// On Windows, if the total command-line length would exceed the safe limit,
// files are chunked across multiple invocations.
func runCommand(command, files []string, log *logger) int {
	if runtime.GOOS == "windows" {
		return runChunked(command, files, log)
	}
	return runOnce(command, files)
}

func runOnce(command, files []string) int {
	args := append(command[1:], files...)
	cmd := exec.Command(command[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "[hookset] command error: %v\n", err)
		return 1
	}
	return 0
}

func runChunked(command, files []string, log *logger) int {
	baseLen := len(strings.Join(command, " ")) + 1
	var chunk []string
	chunkLen := baseLen

	flush := func() int {
		if len(chunk) == 0 {
			return 0
		}
		log.step("Chunk: %d file(s) (Windows arg-limit split)", len(chunk))
		code := runOnce(command, chunk)
		chunk = nil
		chunkLen = baseLen
		return code
	}

	for _, f := range files {
		fLen := len(f) + 1
		if chunkLen+fLen > windowsArgLimit && len(chunk) > 0 {
			if code := flush(); code != 0 {
				return code
			}
		}
		chunk = append(chunk, f)
		chunkLen += fLen
	}
	return flush()
}

// classifyAfterRun separates matched files into still-present (add) and
// deleted-by-tool (rm --cached).
func classifyAfterRun(files []string) (toAdd, toRemove []string) {
	for _, f := range files {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			toRemove = append(toRemove, f)
		} else {
			toAdd = append(toAdd, f)
		}
	}
	return
}

// firstLargeFile returns the first file over the large-file threshold.
func firstLargeFile(files []string) (string, int64) {
	for _, f := range files {
		info, err := os.Stat(f)
		if err == nil && info.Size() > largeBytesThreshold {
			return f, info.Size()
		}
	}
	return "", 0
}

// exclude returns elements in all that are not in subset.
func exclude(all, subset []string) []string {
	set := make(map[string]struct{}, len(subset))
	for _, f := range subset {
		set[f] = struct{}{}
	}
	var out []string
	for _, f := range all {
		if _, ok := set[f]; !ok {
			out = append(out, f)
		}
	}
	return out
}

// ── logger ────────────────────────────────────────────────────────────────────

type logger struct{ verbose bool }

func newLogger(verbose bool) *logger { return &logger{verbose: verbose} }

func (l *logger) step(format string, args ...any) {
	if l.verbose {
		fmt.Printf("[hookset] "+format+"\n", args...)
	}
}

func (l *logger) info(format string, args ...any) {
	fmt.Printf("[hookset] "+format+"\n", args...)
}
