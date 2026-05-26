// Package orchestrator implements the alpha parallel hook runner (A3+A4).
//
// Gate: opt-in only. Enable via .hookset.toml:
//
//	[hookset]
//	experimental = ["parallel"]
//
// or via environment variable:
//
//	HOOKSET_EXPERIMENTAL=parallel
//
// When enabled, all hooks for an event run concurrently. Output is buffered
// per hook and flushed in definition order so the terminal output is deterministic.
//
// Constraints (validated before running):
//   - interactive = true is incompatible with parallel mode on the same event.
//   - Passthrough hooks (arg-style events) run serially even in parallel mode
//     because they accept git positional args that cannot be shared.
package orchestrator

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/bulga138/hookset/internal/manifest"
)

// AlphaBanner is printed once when parallel mode is activated.
const AlphaBanner = "[hookset] \u26a0 alpha: parallel mode enabled — behaviour may change in future versions"

// Result holds the outcome of a single hook run.
type Result struct {
	Entry    manifest.Entry
	ExitCode int
	Output   []byte // combined stdout+stderr, buffered
}

// Options configures an orchestrated run.
type Options struct {
	Event   string
	Entries []manifest.Entry
	// Verbose enables per-hook step output.
	Verbose bool
}

// IsEnabled returns true when parallel mode is active for the given
// experimental slice and/or the HOOKSET_EXPERIMENTAL env var.
func IsEnabled(experimental []string) bool {
	for _, f := range experimental {
		if strings.TrimSpace(strings.ToLower(f)) == "parallel" {
			return true
		}
	}
	env := os.Getenv("HOOKSET_EXPERIMENTAL")
	for _, f := range strings.Split(env, ",") {
		if strings.TrimSpace(strings.ToLower(f)) == "parallel" {
			return true
		}
	}
	return false
}

// ValidateEntries checks that no interactive hook is mixed with parallel mode.
// Returns a non-nil error listing all conflicting hook names.
func ValidateEntries(entries []manifest.Entry) error {
	var conflicts []string
	for _, e := range entries {
		if e.Interactive {
			conflicts = append(conflicts, e.Name)
		}
	}
	if len(conflicts) > 0 {
		return fmt.Errorf(
			"parallel mode is incompatible with interactive = true on hook(s): %s\n"+
				"         Set interactive = false or remove experimental = [\"parallel\"]",
			strings.Join(conflicts, ", "))
	}
	return nil
}

// Run executes all entries for the event in parallel, returns the ordered
// results. Passthrough hooks are run serially (they receive git positional
// args that cannot be parallelised).
func Run(opts Options) []Result {
	entries := opts.Entries
	if len(entries) == 0 {
		return nil
	}

	results := make([]Result, len(entries))
	var wg sync.WaitGroup
	mu := sync.Mutex{} // guards writes to results

	for i, e := range entries {
		i, e := i, e // capture
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, out := runOne(e, opts.Verbose)
			mu.Lock()
			results[i] = Result{Entry: e, ExitCode: code, Output: out}
			mu.Unlock()
		}()
	}
	wg.Wait()

	return results
}

// FlushResults writes each result's buffered output to w in definition order.
// Returns the highest non-zero exit code seen (0 if all succeeded).
func FlushResults(results []Result, w io.Writer) int {
	worst := 0
	for _, r := range results {
		if len(r.Output) > 0 {
			fmt.Fprintf(w, "[hookset] %s:\n", r.Entry.Name)
			w.Write(r.Output) //nolint:errcheck
			if !bytes.HasSuffix(r.Output, []byte("\n")) {
				fmt.Fprintln(w)
			}
		}
		if r.ExitCode != 0 && r.ExitCode > worst {
			worst = r.ExitCode
		}
	}
	return worst
}

// runOne runs a single hook entry, capturing its combined output.
func runOne(e manifest.Entry, verbose bool) (int, []byte) {
	cmdStr := buildExecCommand(e)
	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.Command("cmd", "/C", cmdStr) //nolint:gosec
	} else {
		c = exec.Command("sh", "-c", cmdStr) //nolint:gosec
	}
	c.Env = os.Environ()
	if e.Cwd != "" {
		c.Dir = e.Cwd
	}

	var buf bytes.Buffer
	c.Stdout = &buf
	c.Stderr = &buf

	if err := c.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), buf.Bytes()
		}
		fmt.Fprintf(&buf, "[hookset] error running %s: %v\n", e.Name, err)
		return 1, buf.Bytes()
	}
	return 0, buf.Bytes()
}

// buildExecCommand assembles the hookset exec invocation for a single entry.
// Replicates the essential parts of buildWrapperCommand without the
// presence-check wrapper — the orchestrator calls hookset exec directly.
func buildExecCommand(e manifest.Entry) string {
	var parts []string
	parts = append(parts, "hookset", "exec")
	parts = append(parts, "--name", shellQuote(e.Name))
	if isPassthrough(e) {
		parts = append(parts, "--event", shellQuote(e.Event))
	}
	for _, m := range e.Match {
		parts = append(parts, "--match", shellQuote(m))
	}
	if e.Cwd != "" {
		parts = append(parts, "--cwd", shellQuote(e.Cwd))
	}
	if e.FailText != "" {
		parts = append(parts, "--fail-text", shellQuote(e.FailText))
	}
	parts = append(parts, "--", "sh", "-c", shellQuote(e.Command))
	if isPassthrough(e) {
		parts = append(parts, `"$@"`)
	}
	return strings.Join(parts, " ")
}

func isPassthrough(e manifest.Entry) bool {
	if e.Passthrough {
		return true
	}
	switch e.Event {
	case "commit-msg", "prepare-commit-msg", "pre-rebase",
		"post-checkout", "post-merge", "post-rewrite", "applypatch-msg":
		return true
	}
	return false
}

func shellQuote(s string) string {
	if !strings.ContainsAny(s, " \t\n\"'\\$`|&;()<>{}[]!#~*?") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
