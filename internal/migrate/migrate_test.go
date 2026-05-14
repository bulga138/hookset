package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandGlob(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"*.ts", []string{"*.ts"}},
		{"src/**", []string{"src/**"}},
		{"*.{ts,js}", []string{"*.ts", "*.js"}},
		{"src/*.{ts,tsx,js,jsx}", []string{"src/*.ts", "src/*.tsx", "src/*.js", "src/*.jsx"}},
		{"a/{x,y}/z", []string{"a/x/z", "a/y/z"}},
		{"no-brace-here", []string{"no-brace-here"}},
		{"{a,b,c}", []string{"a", "b", "c"}},
		// Missing closing brace — returns as single entry.
		{"*.{ts", []string{"*.{ts"}},
		// Empty alternatives.
		{"{,.ts}", []string{"", ".ts"}},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := expandGlob(tc.input)
			if len(got) != len(tc.want) {
				t.Errorf("expandGlob(%q): got %d entries, want %d", tc.input, len(got), len(tc.want))
				return
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("expandGlob(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestExtractHuskyCommands(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   []string
	}{
		{
			name: "shebang and boilerplate stripped",
			script: `#!/usr/bin/env sh
. "$(dirname -- "$0")/_/husky.sh"
npx eslint --fix
`,
			want: []string{"npx eslint --fix"},
		},
		{
			name: "comment lines stripped",
			script: `# This is a comment
# Another comment line
npx prettier --write
`,
			want: []string{"npx prettier --write"},
		},
		{
			name: "multiple commands",
			script: `#!/bin/sh
. "$(dirname "$0")/_/husky.sh"
echo "Running lint"
npx eslint --cache
npx prettier --check
`,
			want: []string{"echo \"Running lint\"", "npx eslint --cache", "npx prettier --check"},
		},
		{
			name:   "empty script",
			script: "",
			want:   nil,
		},
		{
			name: "only comments and shebang",
			script: `#!/bin/sh
# comment
. husky.sh
`,
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractHuskyCommands(tc.script)
			if len(got) != len(tc.want) {
				t.Errorf("extractHuskyCommands: got %d, want %d", len(got), len(tc.want))
				return
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("extractHuskyCommands[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestLooksLikeFileFilter(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		// Filter keywords — look like file filters.
		{"npx eslint --fix src/", true},
		{"npx prettier --write", true},
		{"stylelint \"**/*.css\"", true},
		{"tslint -p tsconfig.json", true},
		{"npx biome check --write .", true},
		{"oxlint", true},
		// Non-filter keywords — project-wide commands.
		{"tsc --noEmit", false},
		{"tsc --project tsconfig.json", false},
		{"jest", false},
		{"vitest run", false},
		{"mocha", false},
		{"go test ./...", false},
		// Unknown commands — default false.
		{"make build", false},
		{"echo hello", false},
	}

	for _, tc := range tests {
		t.Run(tc.cmd, func(t *testing.T) {
			got := looksLikeFileFilter(tc.cmd)
			if got != tc.want {
				t.Errorf("looksLikeFileFilter(%q) = %v, want %v", tc.cmd, got, tc.want)
			}
		})
	}
}

func TestMigrateHusky(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, repoRoot string)
		wantTypes []string // Command substrings we expect in entries
		wantWarn  int      // minimum expected warnings
	}{
		{
			name: "lint-staged delegation",
			setup: func(t *testing.T, repoRoot string) {
				huskyDir := filepath.Join(repoRoot, ".husky")
				if err := os.MkdirAll(huskyDir, 0755); err != nil {
					t.Fatalf("setup husky dir: %v", err)
				}
				script := `#!/bin/sh
npx lint-staged
`
				if err := os.WriteFile(filepath.Join(huskyDir, "pre-commit"), []byte(script), 0755); err != nil {
					t.Fatalf("write pre-commit: %v", err)
				}
				// Also write a lint-staged config so the delegation works.
				if err := os.WriteFile(filepath.Join(repoRoot, ".lintstagedrc"), []byte(`{
  "*.ts": "tsc --noEmit"
}
`), 0644); err != nil {
					t.Fatalf("write lintstagedrc: %v", err)
				}
			},
			wantTypes: []string{"tsc --noEmit"},
			wantWarn:  0,
		},
		{
			name: "eslint hook gets file filter warning",
			setup: func(t *testing.T, repoRoot string) {
				huskyDir := filepath.Join(repoRoot, ".husky")
				if err := os.MkdirAll(huskyDir, 0755); err != nil {
					t.Fatalf("setup husky dir: %v", err)
				}
				script := "npx eslint --cache --fix\n"
				if err := os.WriteFile(filepath.Join(huskyDir, "pre-commit"), []byte(script), 0755); err != nil {
					t.Fatalf("write pre-commit: %v", err)
				}
			},
			wantTypes: []string{"eslint"},
			wantWarn:  1, // looksLikeFileFilter warning
		},
		{
			name: "tsc --noEmit gets no match",
			setup: func(t *testing.T, repoRoot string) {
				huskyDir := filepath.Join(repoRoot, ".husky")
				if err := os.MkdirAll(huskyDir, 0755); err != nil {
					t.Fatalf("setup husky dir: %v", err)
				}
				script := "npx tsc --noEmit\n"
				if err := os.WriteFile(filepath.Join(huskyDir, "pre-commit"), []byte(script), 0755); err != nil {
					t.Fatalf("write pre-commit: %v", err)
				}
			},
			wantTypes: []string{"tsc --noEmit"},
			wantWarn:  0,
		},
		{
			name: "unknown command",
			setup: func(t *testing.T, repoRoot string) {
				huskyDir := filepath.Join(repoRoot, ".husky")
				if err := os.MkdirAll(huskyDir, 0755); err != nil {
					t.Fatalf("setup husky dir: %v", err)
				}
				script := "make build\n"
				if err := os.WriteFile(filepath.Join(huskyDir, "pre-push"), []byte(script), 0755); err != nil {
					t.Fatalf("write pre-push: %v", err)
				}
			},
			wantTypes: []string{"make build"},
			wantWarn:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			tc.setup(t, repoRoot)

			res, err := MigrateHusky(repoRoot)
			if err != nil {
				t.Fatalf("MigrateHusky: %v", err)
			}
			if len(res.Entries) == 0 {
				t.Fatal("MigrateHusky: got 0 entries, want at least 1")
			}

			// Check command substrings are present.
			for _, want := range tc.wantTypes {
				found := false
				for _, e := range res.Entries {
					if containsCmd(e.Command, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("MigrateHusky: command containing %q not found in %v", want, res.Entries)
				}
			}

			// Check warning count (at least tc.wantWarn).
			if len(res.Warnings) < tc.wantWarn {
				t.Errorf("MigrateHusky: got %d warnings, want at least %d", len(res.Warnings), tc.wantWarn)
			}
		})
	}
}

func containsCmd(cmd, substr string) bool {
	for i := 0; i <= len(cmd)-len(substr); i++ {
		if cmd[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Export MigrateHusky for testing — it's internal but tests need it.
func MigrateHusky(repoRoot string) (*Result, error) {
	return migrateHusky(repoRoot)
}
