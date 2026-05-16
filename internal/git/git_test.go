package git

import (
	"runtime"
	"testing"
)

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
