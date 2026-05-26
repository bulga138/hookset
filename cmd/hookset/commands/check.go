package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bulga138/hookset/internal/git"
	"github.com/bulga138/hookset/internal/manifest"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate .hookset.toml syntax and check commands are executable",
	Long: `Validates .hookset.toml syntax, checks that referenced commands are
executable, and warns on unused match patterns.

Useful in CI to ensure hook configs don't drift.
Returns non-zero if any hook is invalid.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := git.RepoRoot()
		if err != nil {
			return err
		}

		tomlPath := filepath.Join(root, manifest.Filename)

		// Check if file exists
		if _, err := os.Stat(tomlPath); os.IsNotExist(err) {
			fmt.Printf("[hookset] %s not found — nothing to check\n", manifest.Filename)
			return nil
		}

		// Read and parse the manifest
		entries, err := manifest.Read(tomlPath, manifest.ReadOptions{
			StrictIncludes: true,
			Warn:           func(msg string) { fmt.Fprintln(os.Stderr, "[hookset] warning:", msg) },
		})
		if err != nil {
			return fmt.Errorf("parsing %s: %w", manifest.Filename, err)
		}

		if len(entries) == 0 {
			fmt.Printf("[hookset] %s is valid but contains no hooks\n", manifest.Filename)
			return nil
		}

		fmt.Printf("[hookset] Checking %d hook(s) in %s\n", len(entries), manifest.Filename)

		hasErrors := false

		// Check each hook
		for _, e := range entries {
			// Check command exists
			if err := checkCommand(e.Command); err != nil {
				fmt.Fprintf(os.Stderr, "[hookset] error: %s: %v\n", e.Name, err)
				hasErrors = true
			} else {
				fmt.Printf("  ✓ %s: command OK\n", e.Name)
			}

			// Warn on missing match patterns for non-global hooks
			if len(e.Match) == 0 {
				fmt.Fprintf(os.Stderr, "[hookset] warning: %s has no match patterns (runs on all files)\n", e.Name)
			}
		}

		if hasErrors {
			return fmt.Errorf("check failed — one or more hooks have invalid commands")
		}

		fmt.Printf("[hookset] All checks passed\n")
		return nil
	},
}

// checkCommand verifies the command is executable.
// It extracts the binary name and checks if it's available in PATH.
func checkCommand(cmdStr string) error {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	binary := parts[0]

	// Check if it's a known built-in or hookset itself
	if binary == "hookset" {
		return nil
	}

	// Check if command exists
	_, err := exec.LookPath(binary)
	if err != nil {
		return fmt.Errorf("command %q not found in PATH", binary)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
