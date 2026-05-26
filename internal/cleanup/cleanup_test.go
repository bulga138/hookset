package cleanup_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bulga138/hookset/internal/cleanup"
)

func TestBackupExistingHooks_backsUpExecutables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit checks are not meaningful on Windows")
	}
	hooksDir := t.TempDir()

	// Write two hook files — one executable, one not.
	writeHook := func(name string, executable bool) {
		p := filepath.Join(hooksDir, name)
		if err := os.WriteFile(p, []byte("#!/bin/sh\necho "+name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if executable {
			if err := os.Chmod(p, 0o755); err != nil {
				t.Fatal(err)
			}
		}
	}
	writeHook("pre-commit", true)
	writeHook("pre-push", true)
	writeHook("commit-msg", false) // not executable — should be skipped

	entries, err := cleanup.BackupExistingHooks(hooksDir)
	if err != nil {
		t.Fatalf("BackupExistingHooks: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 backup entries, got %d", len(entries))
	}

	// Backup files should exist on disk.
	for _, e := range entries {
		if _, err := os.Stat(e.BackupPath); err != nil {
			t.Errorf("backup file missing: %s", e.BackupPath)
		}
		if e.SHA256 == "" {
			t.Errorf("entry %s has empty SHA256", e.Event)
		}
	}

	// MANIFEST.json should exist.
	manifestPath := filepath.Join(hooksDir, "_backup", "MANIFEST.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("MANIFEST.json missing: %v", err)
	}
}

func TestBackupExistingHooks_idempotent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit checks are not meaningful on Windows")
	}
	hooksDir := t.TempDir()

	p := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	e1, err := cleanup.BackupExistingHooks(hooksDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(e1) != 1 {
		t.Fatalf("first call: expected 1 entry, got %d", len(e1))
	}

	// Second call with identical content should produce no new entries.
	e2, err := cleanup.BackupExistingHooks(hooksDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(e2) != 0 {
		t.Fatalf("second call (idempotent): expected 0 entries, got %d", len(e2))
	}
}

func TestBackupExistingHooks_skipsUnknownFiles(t *testing.T) {
	hooksDir := t.TempDir()

	// Write a file with an unknown name (not a git hook).
	p := filepath.Join(hooksDir, "not-a-hook")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	entries, err := cleanup.BackupExistingHooks(hooksDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries for unknown file, got %d", len(entries))
	}
}

func TestRestoreFromBackup_restoresLatest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit checks are not meaningful on Windows")
	}
	hooksDir := t.TempDir()

	content := []byte("#!/bin/sh\necho hello\n")
	p := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(p, content, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := cleanup.BackupExistingHooks(hooksDir); err != nil {
		t.Fatal(err)
	}

	// Remove the original.
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}

	// Restore it.
	if err := cleanup.RestoreFromBackup(hooksDir, "pre-commit"); err != nil {
		t.Fatalf("RestoreFromBackup: %v", err)
	}

	restored, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(restored) != string(content) {
		t.Errorf("restored content mismatch: got %q, want %q", restored, content)
	}
}

func TestRestoreFromBackup_errorWhenNoBackup(t *testing.T) {
	hooksDir := t.TempDir()
	err := cleanup.RestoreFromBackup(hooksDir, "pre-commit")
	if err == nil {
		t.Fatal("expected error when no backup exists, got nil")
	}
}
