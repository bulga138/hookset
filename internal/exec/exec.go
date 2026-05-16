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
	Command []string
	// Verbose enables step-by-step output.
	Verbose bool
	// AllowLarge skips the >10 MB file warning.
	AllowLarge bool
	// NoStash disables the stash/pop cycle (escape hatch for Windows issues).
	NoStash bool
	// Summary enables summary table output after hook runs.
	Summary bool
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

	if len(staged) == 0 {
		log.info("No staged files — skipping")
		return 0
	}

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

	if len(matched) == 0 {
		log.info("No files match patterns %v — skipping", opts.Patterns)
		return 0
	}

	// ── 4. Warn on large files ────────────────────────────────────────────────

	if !opts.AllowLarge {
		if name, size := firstLargeFile(matched); name != "" {
			fmt.Fprintf(os.Stderr,
				"[hookset] warning: %s is %.1f MB — stashing large files may be slow.\n"+
					"         Pass --allow-large to suppress this warning.\n",
				name, float64(size)/1024/1024)
		}
	}

	// ── 5. Stash unstaged changes ─────────────────────────────────────────────

	stashCreated := false
	if !opts.NoStash && (git.HasUnstagedChanges() || git.HasUntrackedFiles()) {
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
				fmt.Fprintln(os.Stderr, "         Run `git stash pop` manually to restore your working tree.")
			}
		}
	}()

	// ── 6. Run the tool ───────────────────────────────────────────────────────

	log.step("Running: %s %s", strings.Join(opts.Command, " "), strings.Join(matched, " "))

	code := runCommand(opts.Command, matched, log)
	if code != 0 {
		exitCode = code
		fmt.Fprintf(os.Stderr, "[hookset] %s exited with code %d — commit aborted\n",
			opts.Command[0], code)
		return exitCode
	}

	// ── 7. Re-stage modified or deleted files ─────────────────────────────────

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

	log.info("Done ✓")

	// Print summary if requested
	if opts.Summary {
		printSummary(opts.Command, matched, exitCode == 0)
	}

	return 0
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
