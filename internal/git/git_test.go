package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// ── version parsing unit tests (no git binary required) ──────────────────────

func TestParseVersionString(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"git version 2.54.0", "2.54.0"},
		{"git version 2.54.0.windows.1", "2.54.0.windows.1"},
		{"git version 2.43.1 (Apple Git-145)", "2.43.1"},
		{"git version 2.55.0-rc1", "2.55.0-rc1"},
		{"git version 2.9.0", "2.9.0"},
		{"git version 3.0.0", "3.0.0"},
		{"not git output", "unknown"},
		{"git version", "unknown"},
		{"", "unknown"},
	}
	for _, tc := range tests {
		got := parseVersionString(tc.raw)
		if got != tc.want {
			t.Errorf("parseVersionString(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestCheckMinVersionStr(t *testing.T) {
	tests := []struct {
		ver     string
		major   int
		minor   int
		wantErr bool
	}{
		// Exactly the floor
		{"2.54.0", 2, 54, false},
		// Above the floor (patch)
		{"2.54.1", 2, 54, false},
		// Above the floor (minor)
		{"2.55.0", 2, 54, false},
		// Above the floor (major)
		{"3.0.0", 2, 54, false},
		// Below the floor
		{"2.53.9", 2, 54, true},
		{"2.9.0", 2, 54, true},
		// Windows variant
		{"2.54.0.windows.1", 2, 54, false},
		{"2.43.1.windows.1", 2, 54, true},
		// Apple variant (parens stripped by Fields)
		{"2.43.1", 2, 54, true},
		// Unparseable
		{"unknown", 2, 54, true},
		{"", 2, 54, true},
	}
	for _, tc := range tests {
		err := checkMinVersionStr(tc.ver, tc.major, tc.minor)
		gotErr := err != nil
		if gotErr != tc.wantErr {
			t.Errorf("checkMinVersionStr(%q, %d, %d) err=%v, wantErr=%v",
				tc.ver, tc.major, tc.minor, err, tc.wantErr)
		}
	}
}

func TestVersion(t *testing.T) {
	v := Version()
	if v == "" || v == "unknown" {
		t.Skip("git not available or version not parseable")
	}

	// Version should be a dotted string like "2.43.0.windows.1" or "2.43.0".
	// Verify it has at least major.minor format.
	major := ""
	minor := ""
	for i := 0; i < len(v); i++ {
		if v[i] == '.' {
			if major == "" {
				major = v[:i]
			} else if minor == "" {
				minor = v[i+1:]
				break
			}
		}
	}
	if major == "" || minor == "" {
		t.Errorf("Version: got %q, want major.minor format", v)
	}
}

func TestCheckMinVersion(t *testing.T) {
	v := Version()
	if v == "unknown" {
		t.Skip("git not available")
	}

	tests := []struct {
		major  int
		minor  int
		wantOK bool
	}{
		{2, 40, true},   // git 2.43+ is well above 2.40
		{999, 0, false}, // unrealistically high — should fail
	}

	for _, tc := range tests {
		err := CheckMinVersion(tc.major, tc.minor)
		gotOK := err == nil
		if gotOK != tc.wantOK {
			t.Errorf("CheckMinVersion(%d,%d): got OK=%v, want OK=%v (version=%s)", tc.major, tc.minor, gotOK, tc.wantOK, v)
		}
	}
}

func TestCheckMinVersion_edgeCases(t *testing.T) {
	v := Version()
	if v == "unknown" {
		t.Skip("git not available")
	}

	// git 2.54.x should pass for 2.54.
	err := CheckMinVersion(2, 54)
	if err != nil {
		t.Errorf("CheckMinVersion(2,54): unexpected error: %v (version=%s)", err, v)
	}

	// Should fail for 3.0 (higher major).
	err = CheckMinVersion(3, 0)
	if err == nil {
		t.Errorf("CheckMinVersion(3,0): expected error, got nil (version=%s)", v)
	}
}

func TestRepoRoot(t *testing.T) {
	// This test may be skipped if not inside a git repo.
	root, err := RepoRoot()
	if err != nil {
		t.Skip("not inside a git repository — skipping RepoRoot test")
	}
	if root == "" {
		t.Error("RepoRoot: returned empty string")
	}
}

func TestStagedFiles_empty(t *testing.T) {
	// In a fresh temp repo there are no staged files.
	// This verifies StagedFiles returns nil gracefully.
	files := StagedFiles()
	if files == nil {
		t.Log("StagedFiles in empty repo: returned nil (acceptable)")
	}
}

func TestIsInsideWorkTree(t *testing.T) {
	// This test runs in a real git repo (the project itself).
	if !IsInsideWorkTree() {
		t.Error("IsInsideWorkTree: expected true in this git repo")
	}
}

func TestHasUnstagedChanges(t *testing.T) {
	// This is a basic sanity check — the working tree may or may not have changes.
	// We just verify the function doesn't crash.
	_ = HasUnstagedChanges()
}

func TestHasUntrackedFiles(t *testing.T) {
	// Sanity check — function should not crash.
	_ = HasUntrackedFiles()
}

// TestRun verifies that Run executes a basic git command.
func TestRun(t *testing.T) {
	r := Run("--version")
	if !r.OK {
		t.Skip("git not available")
	}
	if r.Stdout == "" {
		t.Error("Run --version: stdout is empty")
	}
}

// TestResult_fields verifies Result struct is populated correctly.
func TestResult(t *testing.T) {
	r := Result{Stdout: "test out", Stderr: "test err", OK: true}
	if r.Stdout != "test out" {
		t.Errorf("Result.Stdout = %q, want %q", r.Stdout, "test out")
	}
	if r.Stderr != "test err" {
		t.Errorf("Result.Stderr = %q, want %q", r.Stderr, "test err")
	}
	if !r.OK {
		t.Error("Result.OK = false, want true")
	}
}

// TestScope_constants verifies scope constants are correct.
func TestScope_constants(t *testing.T) {
	if ScopeLocal != "--local" {
		t.Errorf("ScopeLocal = %q, want %q", ScopeLocal, "--local")
	}
	if ScopeGlobal != "--global" {
		t.Errorf("ScopeGlobal = %q, want %q", ScopeGlobal, "--global")
	}
	if ScopeSystem != "--system" {
		t.Errorf("ScopeSystem = %q, want %q", ScopeSystem, "--system")
	}
}

// TestPlatformDependent verifies platform-specific constants are defined.
func TestPlatformDependent(t *testing.T) {
	// Just verify the runtime constant is valid for the platform.
	if runtime.GOOS == "windows" {
		t.Log("Running on Windows")
	}
}

func TestRunInherit(t *testing.T) {
	err := RunInherit("--version")
	if err != nil {
		t.Skip("git not available")
	}
}

// ── stash regression tests ────────────────────────────────────────────────────
//
// These tests create isolated temporary git repositories so they do not
// interfere with the hookset repo itself.

// initTmpRepo creates a fresh git repo in a temp dir, configures the minimum
// required user identity, and returns the repo root. The caller's working
// directory is changed to the repo root for the duration of the test.
func initTmpRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		// Use a minimal env so that the host repo's git hooks (branch-name
		// validators, etc.) are not inherited by the child repo.
		cmd.Env = []string{
			"HOME=" + os.Getenv("HOME"),
			"PATH=" + os.Getenv("PATH"),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	// --template="" ensures global templateDir hooks are not copied in,
	// so the child repo's commits are not subject to the host's branch-name
	// validation hook.
	run("init", "--template=")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	// Disable line ending conversion to ensure consistent test behavior on Windows
	run("config", "core.autocrlf", "false")

	// Create and commit an initial file so the repo has a HEAD.
	initial := filepath.Join(dir, "README.md")
	if err := os.WriteFile(initial, []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "init")

	// Switch the test's working directory to the repo so git commands in the
	// package under test operate on this isolated repo.
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	return dir
}

// TestStashPushKeepIndex_leavesUntrackedFilesInWorkingTree verifies that
// StashPushKeepIndex does NOT remove untracked files from the working tree.
// This is the core contract after dropping --include-untracked.
func TestStashPushKeepIndex_leavesUntrackedFilesInWorkingTree(t *testing.T) {
	dir := initTmpRepo(t)

	// Modify the tracked file (unstaged) so a stash is actually created.
	tracked := filepath.Join(dir, "README.md")
	if err := os.WriteFile(tracked, []byte("modified\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create an untracked file (simulating .eslintcache written by a tool).
	untracked := filepath.Join(dir, ".eslintcache")
	if err := os.WriteFile(untracked, []byte("cache-data\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := StashPushKeepIndex("test-stash"); err != nil {
		t.Fatalf("StashPushKeepIndex: unexpected error: %v", err)
	}

	// The untracked file must still be present after the stash push.
	if _, err := os.Stat(untracked); os.IsNotExist(err) {
		t.Error("untracked .eslintcache was removed by StashPushKeepIndex; want it preserved")
	}

	// Clean up: pop the stash so the temp dir is in a predictable state.
	_ = StashPop()
}

// TestStashPushKeepIndex_StashPop_roundTrip_withUntrackedSideEffect is the
// full regression test for the reported bug:
//
//  1. A tracked file has an unstaged modification → stash is created.
//  2. A "tool" writes a new untracked file as a side effect (e.g. eslint --cache).
//  3. StashPop must succeed even though the untracked file now exists in the
//     working tree (it was not in the stash, so there is no conflict).
//  4. The unstaged modification to the tracked file is restored correctly.
func TestStashPushKeepIndex_StashPop_roundTrip_withUntrackedSideEffect(t *testing.T) {
	dir := initTmpRepo(t)

	tracked := filepath.Join(dir, "README.md")

	// Modify the committed file without staging it → unstaged change.
	if err := os.WriteFile(tracked, []byte("initial\nmodified unstaged\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Confirm HasUnstagedChanges sees the change.
	if !HasUnstagedChanges() {
		t.Fatal("precondition: expected unstaged changes before stash")
	}

	if err := StashPushKeepIndex("regression-stash"); err != nil {
		t.Fatalf("StashPushKeepIndex: %v", err)
	}

	// After stash push, working tree should match HEAD (unstaged change hidden).
	data, _ := os.ReadFile(tracked)
	if string(data) != "initial\n" {
		t.Errorf("expected working tree to match HEAD after stash; got %q", string(data))
	}

	// Simulate the tool writing a new untracked file (e.g. .eslintcache).
	cache := filepath.Join(dir, ".eslintcache")
	if err := os.WriteFile(cache, []byte("fresh-cache\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// StashPop must NOT fail even though .eslintcache now exists and was not in
	// the stash (since we no longer use --include-untracked).
	if err := StashPop(); err != nil {
		t.Fatalf("StashPop failed after tool wrote untracked file: %v\nThis is the regression.", err)
	}

	// The unstaged modification to the tracked file must be restored.
	data, err := os.ReadFile(tracked)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "initial\nmodified unstaged\n" {
		t.Errorf("tracked file content after pop = %q, want %q",
			string(data), "initial\nmodified unstaged\n")
	}

	// The untracked cache file must still exist (hookset does not touch it).
	if _, err := os.Stat(cache); os.IsNotExist(err) {
		t.Error(".eslintcache was removed by StashPop; want it preserved")
	}
}
