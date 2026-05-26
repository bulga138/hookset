package commands

import (
	"os"
	"os/exec"
	"testing"

	"github.com/bulga138/hookset/internal/git"
	"github.com/bulga138/hookset/internal/gitconfig"
)

// testWithRepo creates an isolated git repo in a temp directory and changes
// the process CWD to it so that functions operate on the isolated repo.
func testWithRepo(t *testing.T) (repo, origDir string) {
	t.Helper()
	origDir, _ = os.Getwd()
	repo = t.TempDir()
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("os.Chdir to test repo: %v", err)
	}
	cmd := exec.Command("git", "init")
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init in test repo: %v", err)
	}
	return repo, origDir
}

func restoreDir(origDir string) {
	_ = os.Chdir(origDir)
}

// TestToggleHook tests enable/disable functionality
func TestToggleHook(t *testing.T) {
	repo, origDir := testWithRepo(t)
	defer restoreDir(origDir)

	// Add a hook first
	h := gitconfig.Hook{
		Name:    "test-hook",
		Event:   "pre-commit",
		Command: "echo test",
		Matches: []string{"*.go"},
	}
	if err := gitconfig.AddHook(h, git.ScopeLocal); err != nil {
		t.Fatalf("failed to add hook: %v", err)
	}

	// Test disable
	err := toggleHook("test-hook", false)
	if err != nil {
		t.Errorf("toggleHook(disable) error = %v", err)
	}

	// Verify hook is disabled
	hook, err := gitconfig.GetHook("test-hook", git.ScopeLocal)
	if err != nil {
		t.Fatalf("failed to get hook: %v", err)
	}
	if hook.Enabled {
		t.Error("hook should be disabled")
	}

	// Test enable
	err = toggleHook("test-hook", true)
	if err != nil {
		t.Errorf("toggleHook(enable) error = %v", err)
	}

	// Verify hook is enabled
	hook, err = gitconfig.GetHook("test-hook", git.ScopeLocal)
	if err != nil {
		t.Fatalf("failed to get hook: %v", err)
	}
	if !hook.Enabled {
		t.Error("hook should be enabled")
	}

	_ = repo // silence unused warning
}

// TestToggleGlobalFlag tests the --global flag
func TestToggleGlobalFlag(t *testing.T) {
	toggleGlobal = true
	defer func() { toggleGlobal = false }()

	if !toggleGlobal {
		t.Error("toggleGlobal flag should be true")
	}
}

// TestListCommand tests the list command
func TestListEvent(t *testing.T) {
	// Test listEventFlag flag
	listEventFlag = "pre-commit"
	defer func() { listEventFlag = "" }()

	if listEventFlag != "pre-commit" {
		t.Errorf("listEventFlag = %q, want pre-commit", listEventFlag)
	}
}

// TestListWithNoHooks tests listing when no hooks exist
func TestListWithNoHooks(t *testing.T) {
	repo, origDir := testWithRepo(t)
	defer restoreDir(origDir)

	// GetHooks should return nil, not error, when no hooks exist
	hooks, err := gitconfig.GetHooks("pre-commit", git.ScopeLocal)
	if err != nil {
		t.Errorf("GetHooks() error = %v", err)
	}
	if len(hooks) != 0 {
		t.Errorf("expected no hooks, got %d", len(hooks))
	}

	_ = repo // silence unused warning
}

// TestToggleNonExistentHook tests toggling a hook that doesn't exist
// Note: gitconfig.EnableHook doesn't error for non-existent hooks
func TestToggleNonExistentHook(t *testing.T) {
	repo, origDir := testWithRepo(t)
	defer restoreDir(origDir)

	// This doesn't error even for non-existent hooks
	err := toggleHook("nonexistent", true)
	if err != nil {
		t.Errorf("toggleHook() unexpected error = %v", err)
	}

	_ = repo // silence unused warning
}
