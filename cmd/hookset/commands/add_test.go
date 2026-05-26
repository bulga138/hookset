package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bulga138/hookset/internal/manifest"
)

func TestFindDashDash(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    []string
		wantNil bool
	}{
		{
			name:    "no dash dash",
			args:    []string{"add", "test"},
			wantNil: true,
		},
		{
			name:    "dash dash at end",
			args:    []string{"add", "test", "--"},
			wantNil: true,
		},
		{
			name:    "dash dash with command",
			args:    []string{"add", "test", "--", "npx", "eslint"},
			want:    []string{"npx", "eslint"},
			wantNil: false,
		},
		{
			name:    "dash dash in middle",
			args:    []string{"--match", "*.ts", "--", "echo", "hello"},
			want:    []string{"echo", "hello"},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args
			original := os.Args
			defer func() { os.Args = original }()

			os.Args = tt.args
			got := findDashDash()

			if tt.wantNil && got != nil {
				t.Errorf("findDashDash() = %v, want nil", got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("findDashDash() = nil, want %v", tt.want)
			}
			if !tt.wantNil && got != nil {
				if len(got) != len(tt.want) {
					t.Errorf("findDashDash() len = %d, want %d", len(got), len(tt.want))
				}
				for i, v := range got {
					if i >= len(tt.want) || v != tt.want[i] {
						t.Errorf("findDashDash()[%d] = %q, want %q", i, v, tt.want[i])
					}
				}
			}
		})
	}
}

func TestAddToManifest(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Test that manifest.Upsert works (this is what addToManifest calls internally)
	entry := manifest.Entry{
		Name:    "test-hook",
		Event:   "pre-commit",
		Match:   []string{"*.ts", "*.js"},
		Command: "npx eslint",
	}

	manifestPath := filepath.Join(tmpDir, manifest.Filename)

	// This is what addToManifest does internally
	err := manifest.Upsert(manifestPath, entry)
	if err != nil {
		t.Errorf("manifest.Upsert() error = %v", err)
	}

	// Verify manifest was created
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("manifest file was not created")
	}

	// Verify we can read it back
	entries, err := manifest.Read(manifestPath, manifest.ReadOptions{})
	if err != nil {
		t.Errorf("failed to read manifest: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "test-hook" {
		t.Errorf("expected name 'test-hook', got %q", entries[0].Name)
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"with spaces", "'with spaces'"},
		{"with\ttab", "'with\ttab'"},
		{"with\"quote", "'with\"quote'"},
		{"with'quote", "'with'\\''quote'"},
		{"*.ts", "*.ts"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := shellQuote(tt.input)
			if got != tt.want {
				t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildWrapperCommand(t *testing.T) {
	tests := []struct {
		name       string
		entry      manifest.Entry
		wantExec   bool // whether exec should contain "hookset exec"
		wantBinary string
	}{
		{
			name: "file-filtering hook",
			entry: manifest.Entry{
				Name:    "eslint",
				Event:   "pre-commit",
				Match:   []string{"*.ts"},
				Command: "npx eslint --fix",
			},
			wantExec:   true,
			wantBinary: "hookset",
		},
		{
			name: "non-filtering hook",
			entry: manifest.Entry{
				Name:    "test",
				Event:   "pre-push",
				Match:   nil,
				Command: "go test ./...",
			},
			wantExec:   false,
			wantBinary: "go",
		},
		{
			name: "empty match",
			entry: manifest.Entry{
				Name:    "test",
				Event:   "pre-push",
				Match:   []string{},
				Command: "go test ./...",
			},
			wantExec:   false,
			wantBinary: "go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildWrapperCommand(tt.entry, false)

			if tt.wantExec {
				if !contains(got, "hookset exec") {
					t.Errorf("buildWrapperCommand() should contain 'hookset exec', got %q", got)
				}
			}
			if !contains(got, tt.wantBinary) {
				t.Errorf("buildWrapperCommand() should contain %q, got %q", tt.wantBinary, got)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
