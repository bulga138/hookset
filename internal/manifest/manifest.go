// Package manifest handles reading and writing .hookset.toml.
package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const Filename = ".hookset.toml"

// File is the top-level structure of .hookset.toml.
type File struct {
	Include string  `toml:"include,omitempty"`
	Hooks   []Entry `toml:"hooks"`
}

// Entry is a single [[hooks]] block.
type Entry struct {
	Name    string   `toml:"name"`
	Event   string   `toml:"event"`
	Match   []string `toml:"match,omitempty"`
	Command string   `toml:"command"`
}

// ReadOptions controls how strict missing includes are treated.
type ReadOptions struct {
	// StrictIncludes causes a missing included file to be a fatal error.
	// Default (false): emit a warning and continue.
	StrictIncludes bool
	// Warn is called for non-fatal warnings. If nil, os.Stderr is used.
	Warn func(msg string)
}

// Read parses the manifest at path, resolves any include, and returns the
// merged hook list. Project hooks override included hooks by name.
func Read(path string, opts ReadOptions) ([]Entry, error) {
	warn := opts.Warn
	if warn == nil {
		warn = func(msg string) {
			fmt.Fprintln(os.Stderr, "[hookset] warning:", msg)
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

	if err := validate(merged); err != nil {
		return nil, err
	}
	return merged, nil
}

// Write serialises entries to path as a .hookset.toml, preserving any
// existing include directive. Creates the file if it does not exist.
func Write(path string, entries []Entry) error {
	// Preserve existing include if file already exists.
	var existing File
	if _, err := toml.DecodeFile(path, &existing); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading existing manifest: %w", err)
	}
	f := File{
		Include: existing.Include,
		Hooks:   entries,
	}
	fh, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer func() { _ = fh.Close() }()
	enc := toml.NewEncoder(fh)
	enc.Indent = "  "
	return enc.Encode(f)
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
