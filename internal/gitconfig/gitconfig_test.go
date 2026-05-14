package gitconfig

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/bulga138/hookset/internal/git"
)

// runGit runs git in repoRoot and returns stdout (trimmed).
func runGit(repoRoot string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func TestAddHook(t *testing.T) {
	repo, origDir := testWithRepo(t)
	defer restoreDir(origDir)

	h := Hook{
		Name:    "test-eslint",
		Event:   "pre-commit",
		Command: "npx eslint --fix",
		Matches: []string{"*.ts", "*.js"},
		Enabled: true,
	}
	if err := AddHook(h, git.ScopeLocal); err != nil {
		t.Fatalf("AddHook: %v", err)
	}

	// Verify it persists by reading back with git config.
	out := runGit(repo, "config", "--local", "--get", "hook.test-eslint.event")
	if out != "pre-commit" {
		t.Errorf("AddHook: event = %q, want %q", out, "pre-commit")
	}
	out = runGit(repo, "config", "--local", "--get", "hook.test-eslint.command")
	if out != "npx eslint --fix" {
		t.Errorf("AddHook: command = %q, want %q", out, "npx eslint --fix")
	}
	// Matches are multi-value.
	out = runGit(repo, "config", "--local", "--get-all", "hook.test-eslint.match")
	expected := "*.ts\n*.js"
	if out != expected {
		t.Errorf("AddHook: matches = %q, want %q", out, expected)
	}
}

func TestAddHook_disabled(t *testing.T) {
	repo, origDir := testWithRepo(t)
	defer restoreDir(origDir)

	h := Hook{
		Name:    "slow-hook",
		Event:   "pre-commit",
		Command: "make build",
		Enabled: false,
	}
	if err := AddHook(h, git.ScopeLocal); err != nil {
		t.Fatalf("AddHook disabled: %v", err)
	}
	out := runGit(repo, "config", "--local", "--get", "hook.slow-hook.enabled")
	if out != "false" {
		t.Errorf("AddHook disabled: enabled = %q, want %q", out, "false")
	}
}

func TestRemoveHook(t *testing.T) {
	repo, origDir := testWithRepo(t)
	defer restoreDir(origDir)

	h := Hook{Name: "to-remove", Event: "pre-commit", Command: "echo done"}
	if err := AddHook(h, git.ScopeLocal); err != nil {
		t.Fatalf("setup AddHook: %v", err)
	}
	if err := RemoveHook("to-remove", git.ScopeLocal); err != nil {
		t.Fatalf("RemoveHook: %v", err)
	}
	// After removal, the key should be gone.
	out := runGit(repo, "config", "--local", "--get", "hook.to-remove.event")
	if out != "" {
		t.Errorf("RemoveHook: event still present = %q", out)
	}
}

func TestRemoveHook_idempotent(t *testing.T) {
	_, origDir := testWithRepo(t)
	defer restoreDir(origDir)

	// Removing a hook that does not exist should not error.
	// Note: Windows git may return a non-zero exit code with "fatal: no such section",
	// which is handled by the "No such section" check in RemoveHook.
	if err := RemoveHook("nonexistent-hook", git.ScopeLocal); err != nil {
		t.Fatalf("RemoveHook idempotent: %v", err)
	}
}

func TestEnableHook(t *testing.T) {
	repo, origDir := testWithRepo(t)
	defer restoreDir(origDir)

	h := Hook{Name: "toggle-me", Event: "pre-commit", Command: "echo on"}
	if err := AddHook(h, git.ScopeLocal); err != nil {
		t.Fatalf("setup AddHook: %v", err)
	}

	if err := EnableHook("toggle-me", git.ScopeLocal, false); err != nil {
		t.Fatalf("EnableHook false: %v", err)
	}
	out := runGit(repo, "config", "--local", "--get", "hook.toggle-me.enabled")
	if out != "false" {
		t.Errorf("EnableHook: enabled = %q, want %q", out, "false")
	}

	if err := EnableHook("toggle-me", git.ScopeLocal, true); err != nil {
		t.Fatalf("EnableHook true: %v", err)
	}
	out = runGit(repo, "config", "--local", "--get", "hook.toggle-me.enabled")
	if out != "true" {
		t.Errorf("EnableHook: enabled = %q, want %q", out, "true")
	}
}

func TestGetHook(t *testing.T) {
	_, origDir := testWithRepo(t)
	defer restoreDir(origDir)

	h := Hook{
		Name:    "get-test",
		Event:   "commit-msg",
		Command: "commitlint --edit",
		Matches: []string{"commit-msg*.txt"},
		Enabled: true,
	}
	if err := AddHook(h, git.ScopeLocal); err != nil {
		t.Fatalf("setup AddHook: %v", err)
	}

	got, err := GetHook("get-test", git.ScopeLocal)
	if err != nil {
		t.Fatalf("GetHook: %v", err)
	}
	if got.Name != "get-test" {
		t.Errorf("GetHook: Name = %q, want %q", got.Name, "get-test")
	}
	if got.Event != "commit-msg" {
		t.Errorf("GetHook: Event = %q, want %q", got.Event, "commit-msg")
	}
	if got.Command != "commitlint --edit" {
		t.Errorf("GetHook: Command = %q, want %q", got.Command, "commitlint --edit")
	}
	if len(got.Matches) != 1 || got.Matches[0] != "commit-msg*.txt" {
		t.Errorf("GetHook: Matches = %v, want [%q]", got.Matches, "commit-msg*.txt")
	}
	if !got.Enabled {
		t.Errorf("GetHook: Enabled = false, want true")
	}
}

func TestGetHook_notFound(t *testing.T) {
	_, origDir := testWithRepo(t)
	defer restoreDir(origDir)

	_, err := GetHook("does-not-exist", git.ScopeLocal)
	if err == nil {
		t.Error("GetHook not found: expected error, got nil")
	}
}

func TestGetHooks(t *testing.T) {
	_, origDir := testWithRepo(t)
	defer restoreDir(origDir)

	hooks := []Hook{
		{Name: "h1", Event: "pre-commit", Command: "echo 1"},
		{Name: "h2", Event: "pre-commit", Command: "echo 2"},
		{Name: "h3", Event: "commit-msg", Command: "echo 3"},
	}
	for _, h := range hooks {
		if err := AddHook(h, git.ScopeLocal); err != nil {
			t.Fatalf("setup AddHook: %v", err)
		}
	}

	got, err := GetHooks("pre-commit", git.ScopeLocal)
	if err != nil {
		t.Fatalf("GetHooks pre-commit: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("GetHooks pre-commit: got %d hooks, want 2", len(got))
	}

	gotCmds, err := GetHooks("commit-msg", git.ScopeLocal)
	if err != nil {
		t.Fatalf("GetHooks commit-msg: %v", err)
	}
	if len(gotCmds) != 1 {
		t.Errorf("GetHooks commit-msg: got %d hooks, want 1", len(gotCmds))
	}
}

func TestGetHooks_emptyEvent(t *testing.T) {
	_, origDir := testWithRepo(t)
	defer restoreDir(origDir)

	if err := AddHook(Hook{Name: "a", Event: "pre-commit", Command: "echo a"}, git.ScopeLocal); err != nil {
		t.Fatalf("setup AddHook: %v", err)
	}
	if err := AddHook(Hook{Name: "b", Event: "commit-msg", Command: "echo b"}, git.ScopeLocal); err != nil {
		t.Fatalf("setup AddHook: %v", err)
	}

	got, err := GetHooks("", git.ScopeLocal)
	if err != nil {
		t.Fatalf("GetHooks empty event: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("GetHooks empty event: got %d hooks, want 2", len(got))
	}
}

func TestGetAllHookNames(t *testing.T) {
	_, origDir := testWithRepo(t)
	defer restoreDir(origDir)

	for _, name := range []string{"alpha", "beta", "gamma"} {
		if err := AddHook(Hook{Name: name, Event: "pre-commit", Command: "echo " + name}, git.ScopeLocal); err != nil {
			t.Fatalf("setup AddHook: %v", err)
		}
	}

	names, err := GetAllHookNames(git.ScopeLocal)
	if err != nil {
		t.Fatalf("GetAllHookNames: %v", err)
	}
	if len(names) != 3 {
		t.Errorf("GetAllHookNames: got %d names, want 3", len(names))
	}
}

// ── test helpers ──────────────────────────────────────────────────────────────

// testWithRepo creates an isolated git repo in a temp directory and changes
// the process CWD to it so that gitconfig functions (which use --local scope)
// operate on the isolated repo. Returns the repo path and the original directory.
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

// restoreDir changes back to origDir, tolerating errors so deferred restores
// don't cause secondary failures if origDir is already invalid.
func restoreDir(origDir string) {
	// Best-effort restore; ignore errors since we may already be in the temp dir
	// which is about to be cleaned up by the test runtime.
	_ = os.Chdir(origDir)
}
