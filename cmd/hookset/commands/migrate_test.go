package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bulga138/hookset/internal/manifest"
	"github.com/bulga138/hookset/internal/migrate"
)

// TestMigrateDryRunFlag tests the --dry-run flag
func TestMigrateDryRunFlag(t *testing.T) {
	migrateDryRun = true
	defer func() { migrateDryRun = false }()

	if !migrateDryRun {
		t.Error("migrateDryRun flag should be true")
	}
}

// TestMigrateYesFlag tests the --yes flag
func TestMigrateYesFlag(t *testing.T) {
	migrateYes = true
	defer func() { migrateYes = false }()

	if !migrateYes {
		t.Error("migrateYes flag should be true")
	}
}

// TestMigrateFromFlag tests the --from flag
func TestMigrateFromFlag(t *testing.T) {
	migrateFrom = "lint-staged"
	defer func() { migrateFrom = "" }()

	if migrateFrom != "lint-staged" {
		t.Errorf("migrateFrom = %q, want lint-staged", migrateFrom)
	}
}

// TestJoinQuoted tests the joinQuoted helper function
func TestJoinQuoted(t *testing.T) {
	tests := []struct {
		name string
		ss   []string
		want string
	}{
		{
			name: "single element",
			ss:   []string{"*.ts"},
			want: `"*.ts"`,
		},
		{
			name: "multiple elements",
			ss:   []string{"*.ts", "*.js"},
			want: `"*.ts", "*.js"`,
		},
		{
			name: "empty",
			ss:   []string{},
			want: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinQuoted(tt.ss)
			if got != tt.want {
				t.Errorf("joinQuoted(%v) = %q, want %q", tt.ss, got, tt.want)
			}
		})
	}
}

// TestMigrateLintStaged tests lint-staged migration
func TestMigrateLintStaged(t *testing.T) {
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

	// Create .lintstagedrc.json
	lintstaged := `{
		"*.ts": "npx tsc --noEmit",
		"*.js": "prettier --write"
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, ".lintstagedrc.json"), []byte(lintstaged), 0644); err != nil {
		t.Fatalf("failed to write lintstaged config: %v", err)
	}

	result, err := migrate.Migrate(migrate.SourceLintStaged, tmpDir)
	if err != nil {
		t.Errorf("Migrate(lint-staged) error = %v", err)
	}

	if len(result.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result.Entries))
	}

	// First entry should be for *.ts
	if result.Entries[0].Event != "pre-commit" {
		t.Errorf("expected event 'pre-commit', got %q", result.Entries[0].Event)
	}
}

// TestMigrateHusky tests husky migration
func TestMigrateHusky(t *testing.T) {
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

	// Create .husky directory with a hook
	huskyDir := filepath.Join(tmpDir, ".husky")
	if err := os.Mkdir(huskyDir, 0755); err != nil {
		t.Fatalf("failed to create .husky dir: %v", err)
	}

	// Write a pre-commit hook
	precommit := `#!/bin/sh
npx eslint
`
	if err := os.WriteFile(filepath.Join(huskyDir, "pre-commit"), []byte(precommit), 0644); err != nil {
		t.Fatalf("failed to write pre-commit hook: %v", err)
	}

	result, err := migrate.Migrate(migrate.SourceHusky, tmpDir)
	if err != nil {
		t.Errorf("Migrate(husky) error = %v", err)
	}

	if len(result.Entries) == 0 {
		t.Error("expected at least one entry from husky migration")
	}
}

// TestMigrateLefthook tests lefthook migration
func TestMigrateLefthook(t *testing.T) {
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

	// Create lefthook.yml
	lefthook := `
pre-commit:
  commands:
    lint:
      glob: "*.go"
      run: go fmt {files}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "lefthook.yml"), []byte(lefthook), 0644); err != nil {
		t.Fatalf("failed to write lefthook config: %v", err)
	}

	result, err := migrate.Migrate(migrate.SourceLefthook, tmpDir)
	if err != nil {
		t.Errorf("Migrate(lefthook) error = %v", err)
	}

	if len(result.Entries) == 0 {
		t.Error("expected at least one entry from lefthook migration")
	}

	// Should have warning about sequential execution
	if len(result.Warnings) == 0 {
		t.Error("expected warning about parallelism settings")
	}
}

// TestMigrateUnknownSource tests migration from unknown source
func TestMigrateUnknownSource(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := migrate.Migrate(migrate.Source("unknown"), tmpDir)
	if err == nil {
		t.Error("expected error for unknown source")
	}
}

// TestManifestWriteAndRead tests writing and reading manifest during migration
func TestManifestWriteAndRead(t *testing.T) {
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

	// Create a manifest file
	entries := []manifest.Entry{
		{
			Name:    "test",
			Event:   "pre-commit",
			Match:   []string{"*.go"},
			Command: "go fmt",
		},
	}

	manifestPath := filepath.Join(tmpDir, manifest.Filename)
	if err := manifest.Write(manifestPath, entries); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Read it back
	readEntries, err := manifest.Read(manifestPath, manifest.ReadOptions{})
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	if len(readEntries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(readEntries))
	}
	if readEntries[0].Name != "test" {
		t.Errorf("expected name 'test', got %q", readEntries[0].Name)
	}
}
