package orchestrator_test

import (
	"strings"
	"testing"

	"github.com/bulga138/hookset/internal/manifest"
	"github.com/bulga138/hookset/internal/orchestrator"
)

func TestIsEnabled_experimental(t *testing.T) {
	cases := []struct {
		exp  []string
		want bool
	}{
		{[]string{"parallel"}, true},
		{[]string{"Parallel"}, true},
		{[]string{"other"}, false},
		{nil, false},
	}
	for _, tc := range cases {
		got := orchestrator.IsEnabled(tc.exp)
		if got != tc.want {
			t.Errorf("IsEnabled(%v) = %v, want %v", tc.exp, got, tc.want)
		}
	}
}

func TestValidateEntries_rejectsInteractive(t *testing.T) {
	entries := []manifest.Entry{
		{Name: "a", Event: "pre-commit", Command: "true", Interactive: true},
		{Name: "b", Event: "pre-commit", Command: "true"},
	}
	err := orchestrator.ValidateEntries(entries)
	if err == nil {
		t.Fatal("expected error for interactive hook in parallel mode, got nil")
	}
	if !strings.Contains(err.Error(), "a") {
		t.Errorf("error should mention hook 'a': %v", err)
	}
}

func TestValidateEntries_allowsNonInteractive(t *testing.T) {
	entries := []manifest.Entry{
		{Name: "lint", Event: "pre-commit", Command: "eslint ."},
		{Name: "test", Event: "pre-push", Command: "go test ./..."},
	}
	if err := orchestrator.ValidateEntries(entries); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFlushResults_returnsWorstExitCode(t *testing.T) {
	results := []orchestrator.Result{
		{Entry: manifest.Entry{Name: "a"}, ExitCode: 0, Output: []byte("ok\n")},
		{Entry: manifest.Entry{Name: "b"}, ExitCode: 2, Output: []byte("fail\n")},
		{Entry: manifest.Entry{Name: "c"}, ExitCode: 1, Output: []byte("also fail\n")},
	}
	var sb strings.Builder
	code := orchestrator.FlushResults(results, &sb)
	if code != 2 {
		t.Errorf("FlushResults: got exit code %d, want 2", code)
	}
	if !strings.Contains(sb.String(), "b:") {
		t.Errorf("output missing hook 'b': %s", sb.String())
	}
}
