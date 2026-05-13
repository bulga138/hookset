// Package git wraps git subprocess calls used throughout hookset.
package git

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Scope maps to git config --local / --global / --system.
type Scope string

const (
	ScopeLocal  Scope = "--local"
	ScopeGlobal Scope = "--global"
	ScopeSystem Scope = "--system"
)

// Result holds output and success state of a git call.
type Result struct {
	Stdout string
	Stderr string
	OK     bool
}

// Run executes git with the given arguments in the current working directory.
func Run(args ...string) Result {
	cmd := exec.Command("git", args...)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	var errStr string
	if ee, ok := err.(*exec.ExitError); ok {
		errStr = strings.TrimSpace(string(ee.Stderr))
	}
	return Result{
		Stdout: strings.TrimSpace(string(out)),
		Stderr: errStr,
		OK:     err == nil,
	}
}

// RunInherit runs git with stdio inherited — output goes straight to terminal.
func RunInherit(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}

// RepoRoot returns the absolute path of the current git repository root.
func RepoRoot() (string, error) {
	r := Run("rev-parse", "--show-toplevel")
	if !r.OK {
		return "", fmt.Errorf("not inside a git repository")
	}
	return r.Stdout, nil
}

// Version returns the numeric git version string, e.g. "2.54.0".
func Version() string {
	r := Run("--version") // "git version 2.54.0"
	parts := strings.Fields(r.Stdout)
	if len(parts) >= 3 {
		return parts[2]
	}
	return "unknown"
}

// CheckMinVersion returns an error if git is older than major.minor.
func CheckMinVersion(major, minor int) error {
	ver := Version()
	parts := strings.Split(ver, ".")
	if len(parts) < 2 {
		return fmt.Errorf("could not parse git version %q", ver)
	}
	maj, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return fmt.Errorf("could not parse git version %q", ver)
	}
	if maj > major || (maj == major && min >= minor) {
		return nil
	}
	return fmt.Errorf("git %s is too old: hookset requires git ≥ %d.%d\nUpgrade: https://git-scm.com/downloads", ver, major, minor)
}

// StagedFiles returns ACMR-filtered staged file paths.
func StagedFiles() []string {
	r := Run("diff", "--cached", "--name-only", "--diff-filter=ACMR")
	if !r.OK || r.Stdout == "" {
		return nil
	}
	var files []string
	for _, f := range strings.Split(r.Stdout, "\n") {
		if f = strings.TrimSpace(f); f != "" {
			files = append(files, f)
		}
	}
	return files
}

// MatchStagedFiles returns the subset of staged that match the given pathspec
// patterns, using git ls-files for exact Git-native glob matching.
func MatchStagedFiles(staged []string, patterns []string) ([]string, error) {
	if len(staged) == 0 || len(patterns) == 0 {
		return nil, nil
	}
	args := []string{"ls-files", "--cached", "--"}
	args = append(args, patterns...)
	cmd := exec.Command("git", args...)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}
	stagedSet := make(map[string]struct{}, len(staged))
	for _, f := range staged {
		stagedSet[f] = struct{}{}
	}
	var matched []string
	for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if f = strings.TrimSpace(f); f != "" {
			if _, ok := stagedSet[f]; ok {
				matched = append(matched, f)
			}
		}
	}
	return matched, nil
}

// HasUnstagedChanges returns true if the working tree differs from the index.
func HasUnstagedChanges() bool {
	return !Run("diff", "--quiet").OK
}

// HasUntrackedFiles returns true if there are untracked files.
func HasUntrackedFiles() bool {
	r := Run("ls-files", "--others", "--exclude-standard")
	return r.OK && r.Stdout != ""
}

// StashPushKeepIndex stashes working-tree changes while preserving the index.
func StashPushKeepIndex(message string) error {
	r := Run("stash", "push", "--quiet", "--keep-index",
		"--include-untracked", "--message", message)
	if !r.OK {
		return fmt.Errorf("git stash push failed: %s", r.Stderr)
	}
	return nil
}

// StashPop restores the most recent stash, restoring the index state.
func StashPop() error {
	r := Run("stash", "pop", "--quiet", "--index")
	if !r.OK {
		return fmt.Errorf("git stash pop failed: %s", r.Stderr)
	}
	return nil
}

// Add stages the given files.
func Add(files []string) error {
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"add", "--"}, files...)
	r := Run(args...)
	if !r.OK {
		return fmt.Errorf("git add failed: %s", r.Stderr)
	}
	return nil
}

// RmCached removes files from the index without deleting from disk.
func RmCached(files []string) error {
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"rm", "--cached", "--"}, files...)
	r := Run(args...)
	if !r.OK {
		return fmt.Errorf("git rm --cached failed: %s", r.Stderr)
	}
	return nil
}

// HookList returns the raw output of `git hook list <event>`.
func HookList(event string) string {
	r := Run("hook", "list", event)
	return r.Stdout
}

// IsInsideWorkTree returns true when running inside a git repo.
func IsInsideWorkTree() bool {
	r := Run("rev-parse", "--is-inside-work-tree")
	return r.OK && strings.TrimSpace(r.Stdout) == "true"
}

// IsLinkedWorktree returns true if the current repo is a linked worktree.
// Linked worktrees have a .git file (not dir) pointing into .git/worktrees/<name>.
func IsLinkedWorktree() bool {
	r := Run("rev-parse", "--git-dir")
	if !r.OK {
		return false
	}
	return strings.Contains(r.Stdout, "worktrees")
}
