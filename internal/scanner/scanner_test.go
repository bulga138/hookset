package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanNodeJS(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json to trigger Node.js detection
	pkgJSON := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(pkgJSON, []byte(`{"name": "test"}`), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	suggestions := Scan(tmpDir)

	// Should have eslint and prettier suggestions
	if len(suggestions) < 2 {
		t.Errorf("expected at least 2 suggestions for Node.js, got %d", len(suggestions))
	}

	// Find eslint entry
	var foundESLint, foundPrettier bool
	for _, s := range suggestions {
		if s.Name == "eslint" {
			foundESLint = true
			if s.Event != "pre-commit" {
				t.Errorf("eslint event should be pre-commit, got %q", s.Event)
			}
			if len(s.Match) == 0 {
				t.Error("eslint should have match patterns")
			}
		}
		if s.Name == "prettier" {
			foundPrettier = true
		}
	}

	if !foundESLint {
		t.Error("should find eslint suggestion")
	}
	if !foundPrettier {
		t.Error("should find prettier suggestion")
	}
}

func TestScanGo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod to trigger Go detection
	goMod := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goMod, []byte(`module test`), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	suggestions := Scan(tmpDir)

	// Should have go-fmt and go-test suggestions
	if len(suggestions) < 2 {
		t.Errorf("expected at least 2 suggestions for Go, got %d", len(suggestions))
	}

	var foundFmt, foundTest bool
	for _, s := range suggestions {
		if s.Name == "go-fmt" {
			foundFmt = true
			if s.Event != "pre-commit" {
				t.Errorf("go-fmt event should be pre-commit, got %q", s.Event)
			}
		}
		if s.Name == "go-test" {
			foundTest = true
			if s.Event != "pre-push" {
				t.Errorf("go-test event should be pre-push, got %q", s.Event)
			}
		}
	}

	if !foundFmt {
		t.Error("should find go-fmt suggestion")
	}
	if !foundTest {
		t.Error("should find go-test suggestion")
	}
}

func TestScanPython(t *testing.T) {
	tmpDir := t.TempDir()

	// Create requirements.txt to trigger Python detection
	reqTxt := filepath.Join(tmpDir, "requirements.txt")
	if err := os.WriteFile(reqTxt, []byte("pytest"), 0644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	suggestions := Scan(tmpDir)

	// Should have black suggestion
	var found bool
	for _, s := range suggestions {
		if s.Name == "black" {
			found = true
			if s.Event != "pre-commit" {
				t.Errorf("black event should be pre-commit, got %q", s.Event)
			}
		}
	}

	if !found {
		t.Error("should find black suggestion for Python")
	}
}

func TestScanPythonWithPyproject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create pyproject.toml to trigger Python detection
	pyproject := filepath.Join(tmpDir, "pyproject.toml")
	if err := os.WriteFile(pyproject, []byte("[project]"), 0644); err != nil {
		t.Fatalf("failed to write pyproject.toml: %v", err)
	}

	suggestions := Scan(tmpDir)

	var found bool
	for _, s := range suggestions {
		if s.Name == "black" {
			found = true
		}
	}

	if !found {
		t.Error("should find black suggestion for Python with pyproject.toml")
	}
}

func TestScanEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// Don't create any project files
	suggestions := Scan(tmpDir)

	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions for empty directory, got %d", len(suggestions))
	}
}

func TestScanMultiple(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files for multiple languages
	files := []string{
		"package.json",
		"go.mod",
		"requirements.txt",
	}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", f, err)
		}
	}

	suggestions := Scan(tmpDir)

	// Should have suggestions from all three
	if len(suggestions) < 5 {
		t.Errorf("expected at least 5 suggestions for multi-language project, got %d", len(suggestions))
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if !fileExists(testFile) {
		t.Error("fileExists should return true for existing file")
	}

	if fileExists(filepath.Join(tmpDir, "nonexistent.txt")) {
		t.Error("fileExists should return false for non-existent file")
	}
}

func TestScanReturnsCorrectEntryTypes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goMod, []byte(`module test`), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	suggestions := Scan(tmpDir)

	for _, s := range suggestions {
		// Verify all entries have required fields
		if s.Name == "" {
			t.Error("entry should have a name")
		}
		if s.Event == "" {
			t.Error("entry should have an event")
		}
		if s.Command == "" {
			t.Error("entry should have a command")
		}
		// Match can be nil for non-filtering hooks
	}
}

func TestScanEntryCommands(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json
	pkgJSON := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(pkgJSON, []byte(`{"name": "test"}`), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	suggestions := Scan(tmpDir)

	// Verify eslint command
	for _, s := range suggestions {
		if s.Name == "eslint" {
			if s.Command != "npx eslint --fix" {
				t.Errorf("eslint command should be 'npx eslint --fix', got %q", s.Command)
			}
		}
		if s.Name == "prettier" {
			if s.Command != "npx prettier --write" {
				t.Errorf("prettier command should be 'npx prettier --write', got %q", s.Command)
			}
		}
	}
}
