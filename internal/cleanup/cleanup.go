// Package cleanup backs up existing .git/hooks/* executables before hookset
// takes ownership of the hooks directory. It writes each backed-up file to
// .git/hooks/_backup/<event>.<unix-ts> and records all backups in a
// MANIFEST.json alongside them.
//
// BackupExistingHooks is idempotent: if a backup for an event+content
// combination already exists it skips that file rather than creating a
// duplicate.
package cleanup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/bulga138/hookset/internal/version"
)

// BackupEntry records a single backed-up hook file.
type BackupEntry struct {
	Event       string `json:"event"`
	OrigPath    string `json:"original_path"`
	BackupPath  string `json:"backup_path"`
	SHA256      string `json:"sha256"`
	Timestamp   int64  `json:"timestamp_unix"`
	HooksetVer  string `json:"hookset_version"`
}

// ManifestFile is the on-disk structure for MANIFEST.json.
type ManifestFile struct {
	Entries []BackupEntry `json:"backups"`
}

// knownHookEvents is the complete set of git hook names.
var knownHookEvents = map[string]bool{
	"applypatch-msg": true, "commit-msg": true, "fsmonitor-watchman": true,
	"post-checkout": true, "post-merge": true, "post-receive": true,
	"post-rewrite": true, "post-update": true, "pre-applypatch": true,
	"pre-commit": true, "pre-merge-commit": true, "pre-push": true,
	"pre-rebase": true, "pre-receive": true, "prepare-commit-msg": true,
	"push-to-checkout": true, "update": true,
}

// BackupExistingHooks backs up any executable files in hooksDir that match
// known git hook event names. Returns the list of entries that were backed up
// (empty if nothing needed backing up).
//
// hooksDir is typically <repo-root>/.git/hooks.
func BackupExistingHooks(hooksDir string) ([]BackupEntry, error) {
	backupDir := filepath.Join(hooksDir, "_backup")
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		return nil, fmt.Errorf("cleanup: create backup dir: %w", err)
	}

	// Load existing manifest so we can de-duplicate.
	existing, err := loadManifest(backupDir)
	if err != nil {
		return nil, err
	}
	existingByHash := make(map[string]bool, len(existing.Entries))
	for _, e := range existing.Entries {
		existingByHash[e.Event+":"+e.SHA256] = true
	}

	dirEntries, err := os.ReadDir(hooksDir)
	if err != nil {
		return nil, fmt.Errorf("cleanup: read hooks dir: %w", err)
	}

	now := time.Now().Unix()
	var newEntries []BackupEntry

	for _, d := range dirEntries {
		if d.IsDir() {
			continue
		}
		name := d.Name()
		if !knownHookEvents[name] {
			continue
		}
		info, err := d.Info()
		if err != nil || info.Mode()&0o111 == 0 {
			continue // not executable — skip
		}

		origPath := filepath.Join(hooksDir, name)
		data, err := os.ReadFile(origPath) //nolint:gosec
		if err != nil {
			return nil, fmt.Errorf("cleanup: read %s: %w", origPath, err)
		}

		sum := sha256sum(data)
		if existingByHash[name+":"+sum] {
			continue // identical backup already exists
		}

		backupName := name + "." + strconv.FormatInt(now, 10)
		backupPath := filepath.Join(backupDir, backupName)
		if err := os.WriteFile(backupPath, data, 0o640); err != nil {
			return nil, fmt.Errorf("cleanup: write backup %s: %w", backupPath, err)
		}

		entry := BackupEntry{
			Event:      name,
			OrigPath:   origPath,
			BackupPath: backupPath,
			SHA256:     sum,
			Timestamp:  now,
			HooksetVer: version.Version,
		}
		newEntries = append(newEntries, entry)
		existingByHash[name+":"+sum] = true
	}

	if len(newEntries) > 0 {
		existing.Entries = append(existing.Entries, newEntries...)
		if err := saveManifest(backupDir, existing); err != nil {
			return nil, err
		}
	}

	return newEntries, nil
}

// RestoreFromBackup restores a single event's most recent backup to origPath.
// Returns an error if no backup exists for the event.
func RestoreFromBackup(hooksDir, event string) error {
	backupDir := filepath.Join(hooksDir, "_backup")
	mf, err := loadManifest(backupDir)
	if err != nil {
		return err
	}
	// Find the latest backup for this event.
	var latest *BackupEntry
	for i := range mf.Entries {
		e := &mf.Entries[i]
		if e.Event == event {
			if latest == nil || e.Timestamp > latest.Timestamp {
				latest = e
			}
		}
	}
	if latest == nil {
		return fmt.Errorf("cleanup: no backup found for event %q", event)
	}
	data, err := os.ReadFile(latest.BackupPath) //nolint:gosec
	if err != nil {
		return fmt.Errorf("cleanup: read backup: %w", err)
	}
	return os.WriteFile(latest.OrigPath, data, 0o750) //nolint:gosec
}

// ── helpers ──────────────────────────────────────────────────────────────────

func loadManifest(backupDir string) (*ManifestFile, error) {
	p := filepath.Join(backupDir, "MANIFEST.json")
	data, err := os.ReadFile(p) //nolint:gosec
	if os.IsNotExist(err) {
		return &ManifestFile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cleanup: read MANIFEST.json: %w", err)
	}
	var mf ManifestFile
	if err := json.Unmarshal(data, &mf); err != nil {
		return nil, fmt.Errorf("cleanup: parse MANIFEST.json: %w", err)
	}
	return &mf, nil
}

func saveManifest(backupDir string, mf *ManifestFile) error {
	p := filepath.Join(backupDir, "MANIFEST.json")
	out, err := json.MarshalIndent(mf, "", "  ")
	if err != nil {
		return fmt.Errorf("cleanup: marshal MANIFEST.json: %w", err)
	}
	return os.WriteFile(p, append(out, '\n'), 0o640)
}

func sha256sum(data []byte) string {
	h := sha256.New()
	_, _ = io.Writer(h).Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
