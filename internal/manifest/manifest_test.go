package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".hookset.toml")

	entries := []Entry{
		{Name: "fmt", Event: "pre-commit", Command: "gofmt -w"},
		{Name: "lint", Event: "pre-commit", Match: []string{"*.go"}, Command: "golangci-lint run"},
	}
	if err := Write(path, entries); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Verify the file exists and is non-empty.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after Write: %v", err)
	}
	if len(data) == 0 {
		t.Error("Write: file is empty")
	}

	// Verify a parseable manifest.
	entries2, err := Read(path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read after Write: %v", err)
	}
	if len(entries2) != 2 {
		t.Errorf("Write: got %d entries, want 2", len(entries2))
	}
}

func TestRead(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".hookset.toml")

	entries := []Entry{
		{Name: "test", Event: "pre-commit", Match: []string{"*.go"}, Command: "echo test"},
	}
	if err := Write(path, entries); err != nil {
		t.Fatalf("Write test fixture: %v", err)
	}

	got, err := Read(path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("Read: got %d entries, want 1", len(got))
	}
	if got[0].Name != "test" {
		t.Errorf("Read: Name = %q, want %q", got[0].Name, "test")
	}
	if got[0].Event != "pre-commit" {
		t.Errorf("Read: Event = %q, want %q", got[0].Event, "pre-commit")
	}
}

func TestRead_missingFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nonexistent.toml")
	_, err := Read(path, ReadOptions{})
	if err == nil {
		t.Error("Read missing file: expected error, got nil")
	}
}

func TestRead_missingIncludeWarning(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".hookset.toml")
	// Write manifest with a missing include — should warn but not error.
	if err := Write(path, []Entry{{Name: "h1", Event: "pre-commit", Command: "echo"}}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Prepend include directive to the file manually.
	data, _ := os.ReadFile(path)
	newContent := "include = \"nonexistent.toml\"\n" + string(data)
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		t.Fatalf("WriteFile with include: %v", err)
	}

	var warned bool
	_, err := Read(path, ReadOptions{
		Warn: func(msg string) {
			if strings.Contains(msg, "skipping include") {
				warned = true
			}
		},
	})
	if err != nil {
		t.Fatalf("Read with missing include: unexpected error: %v", err)
	}
	if !warned {
		t.Error("Read with missing include: expected warning, got none")
	}
}

func TestRead_strictInclude(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".hookset.toml")
	// Prepend include directive to the file.
	if err := Write(path, []Entry{{Name: "h1", Event: "pre-commit", Command: "echo"}}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, _ := os.ReadFile(path)
	newContent := "include = \"missing.toml\"\n" + string(data)
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		t.Fatalf("WriteFile with include: %v", err)
	}

	_, err := Read(path, ReadOptions{StrictIncludes: true})
	if err == nil {
		t.Error("Read with StrictIncludes: expected error for missing file, got nil")
	}
}

func TestUpsert(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".hookset.toml")

	// Upsert a new entry.
	e1 := Entry{Name: "hook-a", Event: "pre-commit", Command: "echo a"}
	if err := Upsert(path, e1); err != nil {
		t.Fatalf("Upsert new: %v", err)
	}

	// Upsert again — should replace.
	e2 := Entry{Name: "hook-a", Event: "pre-commit", Command: "echo a updated"}
	if err := Upsert(path, e2); err != nil {
		t.Fatalf("Upsert replace: %v", err)
	}

	entries, err := Read(path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read after Upsert: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Upsert: got %d entries, want 1", len(entries))
	}
	if entries[0].Command != "echo a updated" {
		t.Errorf("Upsert: Command = %q, want %q", entries[0].Command, "echo a updated")
	}

	// Upsert a different entry — should append.
	e3 := Entry{Name: "hook-b", Event: "pre-commit", Command: "echo b"}
	if err := Upsert(path, e3); err != nil {
		t.Fatalf("Upsert append: %v", err)
	}
	entries, err = Read(path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read after append: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("Upsert append: got %d entries, want 2", len(entries))
	}
}

func TestUpsert_idempotent(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".hookset.toml")
	e := Entry{Name: "idem", Event: "pre-commit", Command: "echo once"}

	if err := Upsert(path, e); err != nil {
		t.Fatalf("Upsert 1: %v", err)
	}
	if err := Upsert(path, e); err != nil {
		t.Fatalf("Upsert 2: %v", err)
	}

	entries, err := Read(path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Upsert idempotent: got %d entries, want 1", len(entries))
	}
}

func TestRemove(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".hookset.toml")

	entries := []Entry{
		{Name: "to-keep", Event: "pre-commit", Command: "echo keep"},
		{Name: "to-remove", Event: "pre-commit", Command: "echo remove"},
	}
	if err := Write(path, entries); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if err := Remove(path, "to-remove"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	remaining, err := Read(path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read after Remove: %v", err)
	}
	if len(remaining) != 1 {
		t.Errorf("Remove: got %d entries, want 1", len(remaining))
	}
	if remaining[0].Name != "to-keep" {
		t.Errorf("Remove: remaining Name = %q, want %q", remaining[0].Name, "to-keep")
	}
}

func TestRemove_nonexistent(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".hookset.toml")
	// Removing a non-existent entry should not error (idempotent).
	if err := Remove(path, "does-not-exist"); err != nil {
		t.Fatalf("Remove nonexistent: %v", err)
	}
}

func TestMergeHooks(t *testing.T) {
	base := []Entry{
		{Name: "shared-fmt", Event: "pre-commit", Command: "gofmt -w"},
		{Name: "shared-lint", Event: "pre-commit", Command: "golangci-lint"},
	}
	overlay := []Entry{
		{Name: "shared-lint", Event: "pre-commit", Command: "golangci-lint --fast"}, // override
		{Name: "project-test", Event: "pre-commit", Command: "go test ./..."},
	}

	merged := mergeHooks(base, overlay)

	if len(merged) != 3 {
		t.Errorf("MergeHooks: got %d entries, want 3", len(merged))
	}

	// Find the overridden entry.
	var lintCmd string
	for _, e := range merged {
		if e.Name == "shared-lint" {
			lintCmd = e.Command
		}
	}
	if lintCmd != "golangci-lint --fast" {
		t.Errorf("MergeHooks: shared-lint Command = %q, want %q", lintCmd, "golangci-lint --fast")
	}
}

func TestMergeHooks_overlayWins(t *testing.T) {
	base := []Entry{
		{Name: "shared", Event: "pre-commit", Command: "base-command"},
	}
	overlay := []Entry{
		{Name: "shared", Event: "pre-commit", Command: "overlay-command"},
	}
	merged := mergeHooks(base, overlay)
	if len(merged) != 1 {
		t.Fatalf("MergeHooks: got %d, want 1", len(merged))
	}
	if merged[0].Command != "overlay-command" {
		t.Errorf("MergeHooks: overlay should win, got %q", merged[0].Command)
	}
}

func TestMergeHooks_emptyBase(t *testing.T) {
	overlay := []Entry{{Name: "only", Event: "pre-commit", Command: "echo"}}
	merged := mergeHooks(nil, overlay)
	if len(merged) != 1 {
		t.Errorf("MergeHooks empty base: got %d, want 1", len(merged))
	}
}

func TestMergeHooks_emptyOverlay(t *testing.T) {
	base := []Entry{{Name: "only", Event: "pre-commit", Command: "echo"}}
	merged := mergeHooks(base, nil)
	if len(merged) != 1 {
		t.Errorf("MergeHooks empty overlay: got %d, want 1", len(merged))
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		entries []Entry
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid entries",
			entries: []Entry{{Name: "h1", Event: "pre-commit", Command: "echo"}},
			wantErr: false,
		},
		{
			name:    "missing name",
			entries: []Entry{{Name: "", Event: "pre-commit", Command: "echo"}},
			wantErr: true,
			errMsg:  "missing required field 'name'",
		},
		{
			name:    "missing event",
			entries: []Entry{{Name: "h1", Event: "", Command: "echo"}},
			wantErr: true,
			errMsg:  "missing required field 'event'",
		},
		{
			name:    "missing command",
			entries: []Entry{{Name: "h1", Event: "pre-commit", Command: ""}},
			wantErr: true,
			errMsg:  "missing required field 'command'",
		},
		{
			name:    "duplicate names",
			entries: []Entry{{Name: "dup", Event: "pre-commit", Command: "echo"}, {Name: "dup", Event: "pre-commit", Command: "echo"}},
			wantErr: true,
			errMsg:  "duplicate hook name",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validate(tc.entries)
			if tc.wantErr {
				if err == nil {
					t.Error("validate: expected error, got nil")
				} else if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("validate: error = %q, want containing %q", err.Error(), tc.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validate: unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidate_duplicateNameAcrossMultiple(t *testing.T) {
	entries := []Entry{
		{Name: "first", Event: "pre-commit", Command: "echo 1"},
		{Name: "second", Event: "pre-commit", Command: "echo 2"},
		{Name: "first", Event: "pre-push", Command: "echo 3"}, // duplicate name
	}
	err := validate(entries)
	if err == nil {
		t.Error("validate: expected duplicate name error, got nil")
	}
}

func TestValidate_allFieldsMissing(t *testing.T) {
	entries := []Entry{{}}
	err := validate(entries)
	if err == nil {
		t.Error("validate: expected error for empty entry, got nil")
	}
}

func TestExpandHome(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"~/.config/hooks"},
		{"relative/path"},
		{"/absolute/path"},
	}

	for _, tc := range tests {
		result := expandHome(tc.input)
		if result == "" {
			t.Errorf("expandHome(%q): got empty string", tc.input)
		}
	}
}

// ── B config parity tests ────────────────────────────────────────────────────

func TestEntry_GlobAliasNormalized(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".hookset.toml")

	// Write TOML with glob= instead of match=.
	toml := `
[[hooks]]
name    = "lint"
event   = "pre-commit"
command = "eslint"
glob    = ["*.ts", "*.js"]
`
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := Read(path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if len(e.Match) != 2 {
		t.Errorf("expected glob values merged into match, got match=%v glob=%v", e.Match, e.Glob)
	}
	if len(e.Glob) != 0 {
		t.Errorf("expected glob cleared after merge, got %v", e.Glob)
	}
}

func TestEntry_ConfigParityFields(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".hookset.toml")

	toml := `
[[hooks]]
name        = "typecheck"
event       = "pre-commit"
command     = "tsc --noEmit"
cwd         = "packages/frontend"
fail_text   = "TypeScript errors on {branch}"
interactive = true
tags        = ["slow", "ts"]
`
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := Read(path, ReadOptions{})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Cwd != "packages/frontend" {
		t.Errorf("Cwd: got %q, want %q", e.Cwd, "packages/frontend")
	}
	if e.FailText != "TypeScript errors on {branch}" {
		t.Errorf("FailText: got %q", e.FailText)
	}
	if !e.Interactive {
		t.Error("Interactive: expected true")
	}
	if len(e.Tags) != 2 || e.Tags[0] != "slow" || e.Tags[1] != "ts" {
		t.Errorf("Tags: got %v", e.Tags)
	}
}

func TestReadPyproject(t *testing.T) {
	tmp := t.TempDir()
	pypath := filepath.Join(tmp, "pyproject.toml")

	content := `
[tool.hookset]
[[tool.hookset.hooks]]
name    = "ruff"
event   = "pre-commit"
command = "ruff check"

[[tool.hookset.hooks]]
name    = "mypy"
event   = "pre-push"
command = "mypy ."
`
	if err := os.WriteFile(pypath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadPyproject(pypath)
	if err != nil {
		t.Fatalf("ReadPyproject: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "ruff" || entries[1].Name != "mypy" {
		t.Errorf("unexpected entries: %v", entries)
	}
}

func TestRead_FallsBackToPyproject(t *testing.T) {
	tmp := t.TempDir()
	// No .hookset.toml — only pyproject.toml.
	pypath := filepath.Join(tmp, "pyproject.toml")
	content := `
[tool.hookset]
[[tool.hookset.hooks]]
name    = "ruff"
event   = "pre-commit"
command = "ruff check"
`
	if err := os.WriteFile(pypath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(tmp, ".hookset.toml")
	var warned []string
	entries, err := Read(manifestPath, ReadOptions{
		Warn: func(msg string) { warned = append(warned, msg) },
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "ruff" {
		t.Errorf("unexpected entries: %v", entries)
	}
	if len(warned) == 0 {
		t.Error("expected a warning about pyproject fallback, got none")
	}
}
