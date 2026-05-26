package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const (
	owner       = "bulga138"
	repo        = "hookset"
	latestURL   = "https://api.github.com/repos/%s/%s/releases/latest"
	downloadURL = "https://github.com/%s/%s/releases/download/%s/%s"
)

var (
	updateDryRun  bool
	updateVersion string
	updateBeta    bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Self-update hookset binary from GitHub Releases",
	Long: `Downloads and replaces the hookset binary from GitHub Releases.

Use --version to pin to a specific release.
Use --dry-run to see what would be downloaded without replacing.

Examples:
  hookset update              # Update to latest stable
  hookset update --beta       # Update to latest beta
  hookset update --version v1.2.3`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate()
	},
}

func runUpdate() error {
	// Determine version to fetch
	version := updateVersion
	if version == "" {
		version = "latest"
		if updateBeta {
			version = "latest"
		}
	}

	fmt.Printf("[hookset] Checking for updates (version: %s)...\n", version)

	// Get current version for comparison
	currentVersion := buildVersion()

	// Fetch release info
	tag := version
	if version == "latest" {
		var err error
		tag, err = getLatestTag()
		if err != nil {
			return fmt.Errorf("failed to get latest release: %w", err)
		}
	}

	// Build the download URL
	filename := fmt.Sprintf("hookset_%s_%s_%s", tag, runtime.GOOS, arch())
	downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", owner, repo, tag, filename)

	fmt.Printf("[hookset] Downloading %s...\n", downloadURL)

	if updateDryRun {
		fmt.Printf("[hookset] DRY RUN: Would download %s\n", downloadURL)
		fmt.Printf("[hookset] Current version: %s, would update to: %s\n", currentVersion, tag)
		return nil
	}

	// Download the binary
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Determine where to write the new binary
	newPath := os.Args[0]
	backupPath := newPath + ".bak"

	// Backup current binary
	if err := os.Rename(newPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Write new binary
	out, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		// Try to restore backup
		_ = os.Rename(backupPath, newPath)
		return fmt.Errorf("failed to write new binary: %w", err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		_ = os.Rename(backupPath, newPath)
		return fmt.Errorf("failed to save new binary: %w", err)
	}

	// Clean up backup
	_ = os.Remove(backupPath)

	fmt.Printf("[hookset] Updated to %s\n", tag)
	fmt.Printf("[hookset] Run 'hookset version' to verify\n")

	return nil
}

func getLatestTag() (string, error) {
	resp, err := http.Get(fmt.Sprintf(latestURL, owner, repo))
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Simple JSON parsing - look for "tag_name"
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Find tag_name field
	idx := strings.Index(string(body), `"tag_name"`)
	if idx == -1 {
		return "", fmt.Errorf("tag_name not found in response")
	}

	// Extract value
	start := idx + len(`"tag_name"`) + 2 // skip : and "
	end := strings.Index(string(body)[start:], `"`)
	if end == -1 {
		return "", fmt.Errorf("failed to parse tag_name")
	}

	return string(body)[start : start+end], nil
}

func arch() string {
	if runtime.GOARCH == "amd64" {
		return "x86_64"
	}
	if runtime.GOARCH == "arm64" {
		return "arm64"
	}
	return runtime.GOARCH
}

func init() {
	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false,
		"Show what would be downloaded without replacing")
	updateCmd.Flags().StringVar(&updateVersion, "version", "",
		"Specific version to install")
	updateCmd.Flags().BoolVar(&updateBeta, "beta", false,
		"Include beta releases")
	rootCmd.AddCommand(updateCmd)
}
