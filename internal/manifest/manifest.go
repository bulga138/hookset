// Package manifest handles reading and writing .hookset.toml.
package manifest

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const Filename = ".hookset.toml"

// File is the top-level structure of .hookset.toml.
type File struct {
	Include string      `toml:"include,omitempty"`
	Hookset HooksetConf `toml:"hookset,omitempty"`
	Hooks   []Entry     `toml:"hooks"`
}

// HooksetConf is the [hookset] global config section.
type HooksetConf struct {
	// Experimental is a list of opt-in alpha feature names.
	// Currently recognised: "parallel"
	Experimental []string `toml:"experimental,omitempty"`
}

// Entry is a single [[hooks]] block.
type Entry struct {
	Name    string   `toml:"name"`
	Event   string   `toml:"event"`
	Match   []string `toml:"match,omitempty"`
	// Glob is an alias for Match — lefthook uses "glob", hookset accepts both.
	// On read, Glob values are merged into Match and Glob is cleared.
	Glob    []string `toml:"glob,omitempty"`
	Command string   `toml:"command"`

	// Passthrough skips the staging engine entirely and forwards git's
	// positional arguments ($1, $2, …) directly to the command.
	// Required for arg-style hooks: commit-msg, prepare-commit-msg,
	// pre-rebase, post-checkout, post-merge, post-rewrite, applypatch-msg.
	Passthrough bool `toml:"passthrough,omitempty"`

	// Cwd sets the working directory for the command, relative to the repo root.
	// Useful in monorepos where the tool must run inside a sub-package directory.
	// Defaults to the repo root if empty.
	Cwd string `toml:"cwd,omitempty"`

	// Tags is a list of arbitrary labels attached to this hook.
	// Reserved for future use (e.g. --tag filter on hookset exec/list).
	Tags []string `toml:"tags,omitempty"`

	// Interactive marks this hook as requiring an attached TTY.
	// Incompatible with parallel = true on the same event (validated at read time).
	// Currently stored but not enforced at runtime — enforcement is planned.
	Interactive bool `toml:"interactive,omitempty"`

	// FailText is a human-readable message shown when this hook fails.
	// Supports the same {event}, {branch}, {staged_files} tokens as Command.
	FailText string `toml:"fail_text,omitempty"`
}

// ReadOptions controls how strict missing includes are treated.
type ReadOptions struct {
	// StrictIncludes causes a missing included file to be a fatal error.
	// Default (false): emit a warning and continue.
	StrictIncludes bool
	// Warn is called for non-fatal warnings. If nil, os.Stderr is used.
	Warn func(msg string)
}

// ReadFile parses the manifest at path and returns the full File struct,
// including the [hookset] config section. Use Read() if you only need entries.
func ReadFile(path string, opts ReadOptions) (*File, []Entry, error) {
	warn := opts.Warn
	if warn == nil {
		warn = func(msg string) {
			fmt.Fprintln(os.Stderr, "[hookset] warning:", msg)
		}
	}

	var f File
	if _, err := toml.DecodeFile(path, &f); err != nil {
		return nil, nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	var included []Entry
	if f.Include != "" {
		incPath := expandHome(f.Include)
		if !filepath.IsAbs(incPath) {
			incPath = filepath.Join(filepath.Dir(path), incPath)
		}
		inc, err := readInclude(incPath)
		if err != nil {
			if opts.StrictIncludes {
				return nil, nil, fmt.Errorf("include %q: %w", f.Include, err)
			}
			warn(fmt.Sprintf("skipping include %q: %v", f.Include, err))
		} else {
			included = inc
		}
	}

	merged := mergeHooks(included, f.Hooks)
	for i := range merged {
		if len(merged[i].Glob) > 0 {
			merged[i].Match = append(merged[i].Match, merged[i].Glob...)
			merged[i].Glob = nil
		}
	}
	if err := validate(merged); err != nil {
		return nil, nil, err
	}
	return &f, merged, nil
}

// Read parses the manifest at path, resolves any include, and returns the
// merged hook list. Project hooks override included hooks by name.
//
// If path does not exist, Read falls back to reading a [tool.hookset] section
// from pyproject.toml in the same directory (read-only; writes always go to
// .hookset.toml).
func Read(path string, opts ReadOptions) ([]Entry, error) {
	warn := opts.Warn
	if warn == nil {
		warn = func(msg string) {
			fmt.Fprintln(os.Stderr, "[hookset] warning:", msg)
		}
	}

	// Fall back to pyproject.toml if .hookset.toml is absent.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		pypath := filepath.Join(filepath.Dir(path), "pyproject.toml")
		if _, perr := os.Stat(pypath); perr == nil {
			entries, perr2 := ReadPyproject(pypath)
			if perr2 == nil && len(entries) > 0 {
				warn(fmt.Sprintf(".hookset.toml not found; using [tool.hookset] from %s (read-only)", pypath))
				return entries, nil
			}
		}
	}

	var f File
	if _, err := toml.DecodeFile(path, &f); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	// Resolve include.
	var included []Entry
	if f.Include != "" {
		incPath := expandHome(f.Include)
		if !filepath.IsAbs(incPath) {
			incPath = filepath.Join(filepath.Dir(path), incPath)
		}
		inc, err := readInclude(incPath)
		if err != nil {
			if opts.StrictIncludes {
				return nil, fmt.Errorf("include %q: %w", f.Include, err)
			}
			warn(fmt.Sprintf("skipping include %q: %v", f.Include, err))
		} else {
			included = inc
		}
	}

	// Merge: project hooks override included hooks by name.
	merged := mergeHooks(included, f.Hooks)

	// Normalise: merge glob alias into match for each entry.
	for i := range merged {
		if len(merged[i].Glob) > 0 {
			merged[i].Match = append(merged[i].Match, merged[i].Glob...)
			merged[i].Glob = nil
		}
	}

	if err := validate(merged); err != nil {
		return nil, err
	}
	return merged, nil
}

// Write serialises entries to path as a .hookset.toml, preserving any
// existing include directive and any leading header comment block.
// Creates the file if it does not exist.
func Write(path string, entries []Entry) error {
	// Preserve existing include and leading comments if file already exists.
	var existing File
	leadingComments := readLeadingComments(path)
	if _, err := toml.DecodeFile(path, &existing); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading existing manifest: %w", err)
	}
	f := File{
		Include: existing.Include,
		Hooks:   entries,
	}

	// Encode to a buffer first.
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	enc.Indent = "  "
	if err := enc.Encode(f); err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}

	fh, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer func() { _ = fh.Close() }()

	// Write leading comments first, then the TOML body.
	if len(leadingComments) > 0 {
		if _, err := fmt.Fprint(fh, leadingComments); err != nil {
			return err
		}
	}
	_, err = fh.Write(buf.Bytes())
	return err
}

// readLeadingComments returns the contiguous block of comment lines at the
// start of path (lines beginning with '#' or blank lines within that block).
// Returns "" if the file does not exist or has no leading comments.
func readLeadingComments(path string) string {
	f, err := os.Open(path) //nolint:gosec
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	var sb strings.Builder
	scanner := bufio.NewScanner(f)
	inHeader := true
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if inHeader && (strings.HasPrefix(trimmed, "#") || trimmed == "") {
			sb.WriteString(line)
			sb.WriteByte('\n')
		} else {
			inHeader = false
		}
	}
	return sb.String()
}

// Upsert adds or replaces a hook entry by name, then writes the file.
// Idempotent: calling twice with the same entry produces the same file.
func Upsert(path string, entry Entry) error {
	entries, err := readRaw(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	replaced := false
	for i, e := range entries {
		if e.Name == entry.Name {
			entries[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		entries = append(entries, entry)
	}
	return Write(path, entries)
}

// Remove deletes the named hook from the manifest and writes the file.
func Remove(path, name string) error {
	entries, err := readRaw(path)
	if err != nil {
		return err
	}
	var keep []Entry
	for _, e := range entries {
		if e.Name != name {
			keep = append(keep, e)
		}
	}
	return Write(path, keep)
}

// ── internal helpers ─────────────────────────────────────────────────────────

// ReadPyproject reads hook entries from a pyproject.toml [tool.hookset] section.
// Returns nil, nil if the section is absent. This is read-only: hookset never
// writes back to pyproject.toml.
func ReadPyproject(path string) ([]Entry, error) {
	var doc struct {
		Tool struct {
			Hookset File `toml:"hookset"`
		} `toml:"tool"`
	}
	if _, err := toml.DecodeFile(path, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return doc.Tool.Hookset.Hooks, nil
}

func readRaw(path string) ([]Entry, error) {
	var f File
	if _, err := toml.DecodeFile(path, &f); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return f.Hooks, nil
}

func readInclude(path string) ([]Entry, error) {
	var f File
	if _, err := toml.DecodeFile(path, &f); err != nil {
		return nil, err
	}
	return f.Hooks, nil
}

// mergeHooks merges base (included) and overlay (project) hooks.
// Overlay entries win on name collision.
func mergeHooks(base, overlay []Entry) []Entry {
	index := make(map[string]int, len(base))
	result := make([]Entry, len(base))
	copy(result, base)
	for i, e := range result {
		index[e.Name] = i
	}
	for _, e := range overlay {
		if i, exists := index[e.Name]; exists {
			result[i] = e
		} else {
			result = append(result, e)
		}
	}
	return result
}

func validate(entries []Entry) error {
	seen := map[string]bool{}
	for _, e := range entries {
		if e.Name == "" {
			return fmt.Errorf("hook entry is missing required field 'name'")
		}
		if e.Event == "" {
			return fmt.Errorf("hook %q is missing required field 'event'", e.Name)
		}
		if e.Command == "" {
			return fmt.Errorf("hook %q is missing required field 'command'", e.Name)
		}
		if seen[e.Name] {
			return fmt.Errorf("duplicate hook name %q in manifest", e.Name)
		}
		seen[e.Name] = true
	}
	return nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
