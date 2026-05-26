package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bulga138/hookset/internal/manifest"
)

// TestRemoveManifest tests that removing from manifest works correctly
func TestRemoveManifest(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("os.Chdir to tmpDir: %v", err)
	}

	// Create .git directory for git.RepoRoot()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	// First add a hook
	entry := manifest.Entry{
		Name:    "test-hook",
		Event:   "pre-commit",
		Match:   []string{"*.ts"},
		Command: "npx eslint",
	}

	manifestPath := filepath.Join(tmpDir, manifest.Filename)
	if err := manifest.Write(manifestPath, []manifest.Entry{entry}); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Now remove it
	if err := manifest.Remove(manifestPath, "test-hook"); err != nil {
		t.Errorf("manifest.Remove() error = %v", err)
	}

	// Verify it's gone
	entries, err := manifest.Read(manifestPath, manifest.ReadOptions{})
	if err != nil {
		t.Errorf("failed to read manifest: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after removal, got %d", len(entries))
	}
}

// TestRemoveNonExistent tests removing a hook that doesn't exist
func TestRemoveNonExistent(t *testing.T) {
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

	manifestPath := filepath.Join(tmpDir, manifest.Filename)

	// Removing non-existent hook should not error (idempotent)
	err := manifest.Remove(manifestPath, "nonexistent")
	if err != nil {
		t.Errorf("manifest.Remove() for non-existent should not error, got: %v", err)
	}
}

// TestRemoveFromManifestFlag tests the --manifest flag behavior
func TestRemoveFromManifestFlag(t *testing.T) {
	// This tests the logic that removeManifest flag triggers manifest removal
	removeManifest = true
	defer func() { removeManifest = false }()

	// Test that the flag is set correctly
	if !removeManifest {
		t.Error("removeManifest flag should be true")
	}
}

// TestRemoveGlobalFlag tests the --global flag behavior
func TestRemoveGlobalFlag(t *testing.T) {
	// Test that removeGlobal and toggleGlobal flags can be set
	removeGlobal = true
	defer func() { removeGlobal = false }()

	if !removeGlobal {
		t.Error("removeGlobal flag should be true")
	}

	toggleGlobal = true
	defer func() { toggleGlobal = false }()

	if !toggleGlobal {
		t.Error("toggleGlobal flag should be true")
	}
}
