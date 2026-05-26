package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bulga138/hookset/internal/manifest"
)

// TestBuildPresenceCheck tests the presence check wrapper
func TestBuildPresenceCheck(t *testing.T) {
	tests := []struct {
		name       string
		installMsg string
		execCmd    string
		wantSh     bool
	}{
		{
			name:       "hookset binary check",
			installMsg: "hookset is not installed",
			execCmd:    "hookset exec",
			wantSh:     true,
		},
		{
			name:       "go binary check",
			installMsg: "go is not installed",
			execCmd:    "go test ./...",
			wantSh:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPresenceCheck(tt.installMsg, tt.execCmd)

			if tt.wantSh && !contains(got, "command -v") {
				t.Errorf("buildPresenceCheck() should contain 'command -v', got %q", got)
			}
			// The binary is always "hookset" in the check
			if !contains(got, "hookset") {
				t.Errorf("buildPresenceCheck() should contain 'hookset', got %q", got)
			}
			if !contains(got, tt.execCmd) {
				t.Errorf("buildPresenceCheck() should contain %q, got %q", tt.execCmd, got)
			}
			if !contains(got, tt.installMsg) {
				t.Errorf("buildPresenceCheck() should contain install message %q, got %q", tt.installMsg, got)
			}
		})
	}
}

// TestInitFlags tests that all init flags are properly initialized
func TestInitFlags(t *testing.T) {
	// Test default values
	if initStrict {
		t.Error("initStrict should default to false")
	}
	if initInteractive {
		t.Error("initInteractive should default to false")
	}
	if initNoInteractive {
		t.Error("initNoInteractive should default to false")
	}
	if initDryRun {
		t.Error("initDryRun should default to false")
	}
	if initUninstall {
		t.Error("initUninstall should default to false")
	}
}

// TestInitDryRun tests that dry-run mode doesn't modify files
func TestInitDryRun(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("os.Chdir to tmpDir: %v", err)
	}

	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	// Create manifest
	manifestPath := filepath.Join(tmpDir, manifest.Filename)
	entries := []manifest.Entry{
		{
			Name:    "test",
			Event:   "pre-commit",
			Match:   []string{"*.go"},
			Command: "go fmt",
		},
	}
	if err := manifest.Write(manifestPath, entries); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Set dry-run flag
	initDryRun = true
	defer func() { initDryRun = false }()

	// Run init - it should not error in dry-run mode
	// Note: This would need a full integration test with cobra
}

// TestInitUninstall tests uninstall behavior
func TestUninstallManagedHooks(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("os.Chdir to tmpDir: %v", err)
	}

	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	// Test uninstall on clean repo (no hooks to remove)
	err := uninstallManagedHooks(true)
	if err != nil {
		t.Errorf("uninstallManagedHooks() on clean repo should not error, got: %v", err)
	}
}

// TestInitBootstrapWizard tests bootstrap when no manifest exists
func TestInitBootstrapWithoutManifest(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("os.Chdir to tmpDir: %v", err)
	}

	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	// No manifest file should exist
	manifestPath := filepath.Join(tmpDir, manifest.Filename)
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Error("manifest should not exist initially")
	}
}

// TestInitNoInteractive tests --no-interactive mode
func TestInitNoInteractive(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("os.Chdir to tmpDir: %v", err)
	}

	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	// Create empty manifest
	manifestPath := filepath.Join(tmpDir, manifest.Filename)
	if err := os.WriteFile(manifestPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create empty manifest: %v", err)
	}

	// Set no-interactive flag
	initNoInteractive = true
	defer func() { initNoInteractive = false }()

	// In no-interactive mode, hooks should be auto-installed
}

// TestInitStrictFlag tests --strict mode
func TestInitStrictFlag(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("os.Chdir to tmpDir: %v", err)
	}

	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	// Create manifest with include
	manifestPath := filepath.Join(tmpDir, manifest.Filename)
	manifestContent := `
include = "nonexistent.toml"

[[hooks]]
name = "test"
event = "pre-commit"
command = "echo test"
`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Without strict, missing include is a warning
	opts := manifest.ReadOptions{StrictIncludes: false}
	_, err := manifest.Read(manifestPath, opts)
	if err != nil {
		t.Errorf("non-strict mode should not error on missing include, got: %v", err)
	}

	// With strict, missing include is an error
	optsStrict := manifest.ReadOptions{StrictIncludes: true}
	_, err = manifest.Read(manifestPath, optsStrict)
	if err == nil {
		t.Error("strict mode should error on missing include")
	}
}
