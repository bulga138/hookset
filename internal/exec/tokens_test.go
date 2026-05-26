package exec

import (
	"testing"
)

func TestExpandTokens_noTokens(t *testing.T) {
	out, had := expandTokens("npx eslint --fix", []string{"a.ts", "b.ts"}, "pre-commit", "main")
	if had {
		t.Error("expected hadToken=false when no tokens present")
	}
	if out != "npx eslint --fix" {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestExpandTokens_stagedFiles(t *testing.T) {
	out, had := expandTokens("npx eslint --fix {staged_files}", []string{"src/a.ts", "src/b.ts"}, "pre-commit", "main")
	if !had {
		t.Error("expected hadToken=true")
	}
	want := "npx eslint --fix 'src/a.ts' 'src/b.ts'"
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestExpandTokens_stagedFilesOrDefault_withFiles(t *testing.T) {
	out, had := expandTokens("ruff check {staged_files_or_default}", []string{"foo.py"}, "pre-commit", "main")
	if !had {
		t.Error("expected hadToken=true")
	}
	if out != "ruff check 'foo.py'" {
		t.Errorf("got %q", out)
	}
}

func TestExpandTokens_stagedFilesOrDefault_empty(t *testing.T) {
	out, had := expandTokens("ruff check {staged_files_or_default}", []string{}, "pre-commit", "main")
	if !had {
		t.Error("expected hadToken=true")
	}
	if out != "ruff check ." {
		t.Errorf("got %q", out)
	}
}

func TestExpandTokens_event(t *testing.T) {
	out, had := expandTokens("echo {event}", nil, "pre-push", "main")
	if !had {
		t.Error("expected hadToken=true")
	}
	if out != "echo pre-push" {
		t.Errorf("got %q", out)
	}
}

func TestExpandTokens_branch(t *testing.T) {
	out, had := expandTokens("echo {branch}", nil, "", "feature/my-branch")
	if !had {
		t.Error("expected hadToken=true")
	}
	if out != "echo feature/my-branch" {
		t.Errorf("got %q", out)
	}
}

func TestExpandTokens_escaped(t *testing.T) {
	// {{staged_files}} should become the literal string {staged_files}
	out, had := expandTokens("echo {{staged_files}}", []string{"a.ts"}, "pre-commit", "main")
	if had {
		t.Error("expected hadToken=false for escaped token")
	}
	if out != "echo {staged_files}" {
		t.Errorf("got %q", out)
	}
}

func TestExpandTokens_mixedEscapedAndReal(t *testing.T) {
	// One real token, one escaped; only real one counts.
	out, had := expandTokens("echo {branch} {{event}}", []string{}, "pre-commit", "develop")
	if !had {
		t.Error("expected hadToken=true")
	}
	want := "echo develop {event}"
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestExpandTokens_pathWithSpaceAndQuote(t *testing.T) {
	out, had := expandTokens("{staged_files}", []string{"path with spaces/file.ts", "it's.go"}, "", "")
	if !had {
		t.Error("expected hadToken=true")
	}
	want := `'path with spaces/file.ts' 'it'\''s.go'`
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestShellQuotePaths_empty(t *testing.T) {
	if got := shellQuotePaths(nil); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestShellQuotePaths_simple(t *testing.T) {
	got := shellQuotePaths([]string{"a.ts", "b.ts"})
	want := "'a.ts' 'b.ts'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
