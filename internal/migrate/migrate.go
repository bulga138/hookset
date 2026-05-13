// Package migrate converts existing hook configurations to hookset manifest entries.
package migrate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/bulga138/hookset/internal/manifest"
)

// Source identifies the tool being migrated from.
type Source string

const (
	SourceLintStaged Source = "lint-staged"
	SourceHusky      Source = "husky"
	SourceLefthook   Source = "lefthook"
)

// Result holds the entries produced by a migration and any warnings.
type Result struct {
	Entries  []manifest.Entry
	Warnings []string
}

func (r *Result) warn(msg string) {
	r.Warnings = append(r.Warnings, msg)
}

// Migrate auto-detects and migrates from the given source in repoRoot.
func Migrate(source Source, repoRoot string) (*Result, error) {
	switch source {
	case SourceLintStaged:
		return migrateLintStaged(repoRoot)
	case SourceHusky:
		return migrateHusky(repoRoot)
	case SourceLefthook:
		return migrateLefthook(repoRoot)
	default:
		return nil, fmt.Errorf("unknown source %q", source)
	}
}

// ── lint-staged ───────────────────────────────────────────────────────────────

// lintStagedConfig is the shape of the lint-staged config object.
// Keys are glob patterns, values are command strings or arrays.
type lintStagedConfig map[string]json.RawMessage

func migrateLintStaged(repoRoot string) (*Result, error) {
	cfg, err := findLintStagedConfig(repoRoot)
	if err != nil {
		return nil, err
	}

	res := &Result{}
	idx := 0
	for glob, rawCmd := range cfg {
		idx++
		name := fmt.Sprintf("lint-staged-%d", idx)

		// Command can be a string or an array of strings.
		var cmdStr string
		var cmdArr []string

		if err := json.Unmarshal(rawCmd, &cmdStr); err == nil {
			// Single string command.
		} else if err := json.Unmarshal(rawCmd, &cmdArr); err == nil {
			if len(cmdArr) == 1 {
				cmdStr = cmdArr[0]
			} else {
				// Multiple commands for same glob — emit one entry per command.
				for i, c := range cmdArr {
					res.Entries = append(res.Entries, manifest.Entry{
						Name:    fmt.Sprintf("%s-%d", name, i+1),
						Event:   "pre-commit",
						Match:   expandGlob(glob),
						Command: c,
					})
				}
				continue
			}
		} else {
			// Function syntax or unknown — cannot migrate automatically.
			res.warn(fmt.Sprintf("glob %q uses function syntax — cannot migrate automatically; skipping", glob))
			continue
		}

		res.Entries = append(res.Entries, manifest.Entry{
			Name:    name,
			Event:   "pre-commit",
			Match:   expandGlob(glob),
			Command: cmdStr,
		})
	}
	return res, nil
}

// findLintStagedConfig searches for lint-staged config in priority order:
// .lintstagedrc.json > .lintstagedrc > package.json["lint-staged"].
func findLintStagedConfig(repoRoot string) (lintStagedConfig, error) {
	// Try standalone config files first.
	for _, name := range []string{".lintstagedrc.json", ".lintstagedrc"} {
		p := filepath.Join(repoRoot, name)
		data, err := os.ReadFile(p)
		if err == nil {
			var cfg lintStagedConfig
			if err := json.Unmarshal(data, &cfg); err != nil {
				return nil, fmt.Errorf("parsing %s: %w", name, err)
			}
			return cfg, nil
		}
	}

	// Fall back to package.json.
	pkgPath := filepath.Join(repoRoot, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("no lint-staged config found in %s", repoRoot)
	}
	var pkg struct {
		LintStaged lintStagedConfig `json:"lint-staged"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parsing package.json: %w", err)
	}
	if pkg.LintStaged == nil {
		return nil, fmt.Errorf("no lint-staged key found in package.json")
	}
	return pkg.LintStaged, nil
}

// expandGlob splits brace-expanded globs like "*.{ts,js}" into ["*.ts","*.js"].
// Handles simple single-level brace expansion only.
func expandGlob(glob string) []string {
	start := strings.Index(glob, "{")
	end := strings.Index(glob, "}")
	if start == -1 || end == -1 || end < start {
		return []string{glob}
	}
	prefix := glob[:start]
	suffix := glob[end+1:]
	alts := strings.Split(glob[start+1:end], ",")
	var out []string
	for _, a := range alts {
		out = append(out, prefix+strings.TrimSpace(a)+suffix)
	}
	return out
}

// ── husky ─────────────────────────────────────────────────────────────────────

func migrateHusky(repoRoot string) (*Result, error) {
	huskyDir := filepath.Join(repoRoot, ".husky")
	entries, err := os.ReadDir(huskyDir)
	if err != nil {
		return nil, fmt.Errorf("no .husky directory found at %s", repoRoot)
	}

	res := &Result{}
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), "_") {
			continue // skip husky internals
		}
		event := e.Name() // e.g. "pre-commit"
		scriptPath := filepath.Join(huskyDir, event)
		data, err := os.ReadFile(scriptPath)
		if err != nil {
			res.warn(fmt.Sprintf("could not read .husky/%s: %v", event, err))
			continue
		}

		commands := extractHuskyCommands(string(data))
		for i, cmd := range commands {
			name := fmt.Sprintf("husky-%s", event)
			if len(commands) > 1 {
				name = fmt.Sprintf("husky-%s-%d", event, i+1)
			}

			// Detect lint-staged delegation — fall back to lint-staged migrator.
			if isLintStagedCall(cmd) {
				sub, err := migrateLintStaged(repoRoot)
				if err != nil {
					res.warn(fmt.Sprintf(".husky/%s calls lint-staged but migration failed: %v", event, err))
					continue
				}
				// Override events from lint-staged (it assumes pre-commit).
				for j := range sub.Entries {
					sub.Entries[j].Event = event
				}
				res.Entries = append(res.Entries, sub.Entries...)
				res.Warnings = append(res.Warnings, sub.Warnings...)
				continue
			}

			// Classify: file-filtering vs non-filtering.
			// Non-filtering hooks (tsc, tests, etc.) get no --match; they run as-is.
			// File-filtering hooks are rare in husky scripts without lint-staged,
			// so we default to no match and let the user add --match manually.
			res.Entries = append(res.Entries, manifest.Entry{
				Name:    name,
				Event:   event,
				Match:   nil, // no staged-file filtering — runs as plain command
				Command: cmd,
			})
			if looksLikeFileFilter(cmd) {
				res.warn(fmt.Sprintf(
					"hook %q looks like a file-filtering command — consider adding a 'match' field to run it through `hookset exec`",
					name,
				))
			}
		}
	}
	return res, nil
}

// extractHuskyCommands returns non-boilerplate command lines from a husky script.
func extractHuskyCommands(script string) []string {
	var cmds []string
	for _, line := range strings.Split(script, "\n") {
		line = strings.TrimSpace(line)
		if line == "" ||
			strings.HasPrefix(line, "#") ||
			strings.HasPrefix(line, ". ") ||
			line == "#!/bin/sh" ||
			line == "#!/usr/bin/env sh" ||
			strings.Contains(line, "husky.sh") {
			continue
		}
		cmds = append(cmds, line)
	}
	return cmds
}

func isLintStagedCall(cmd string) bool {
	return strings.Contains(cmd, "lint-staged") ||
		strings.Contains(cmd, "npx lint-staged") ||
		strings.Contains(cmd, "yarn lint-staged")
}

func looksLikeFileFilter(cmd string) bool {
	// Heuristic: commands that include common linter/formatter names
	// but are NOT typical whole-project commands.
	filterKeywords := []string{"eslint", "prettier", "stylelint", "tslint", "biome", "oxlint"}
	nonFilterKeywords := []string{"--project", "--noEmit", "run test", "jest", "vitest", "mocha"}
	lower := strings.ToLower(cmd)
	for _, kw := range nonFilterKeywords {
		if strings.Contains(lower, kw) {
			return false
		}
	}
	for _, kw := range filterKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// ── lefthook ──────────────────────────────────────────────────────────────────

type lefthookFile struct {
	PreCommit lefthookHook `yaml:"pre-commit"`
	PrePush   lefthookHook `yaml:"pre-push"`
	CommitMsg lefthookHook `yaml:"commit-msg"`
}

type lefthookHook struct {
	Commands map[string]lefthookCommand `yaml:"commands"`
}

type lefthookCommand struct {
	Glob string `yaml:"glob"`
	Run  string `yaml:"run"`
}

func migrateLefthook(repoRoot string) (*Result, error) {
	var lhPath string
	for _, name := range []string{"lefthook.yml", "lefthook.yaml"} {
		p := filepath.Join(repoRoot, name)
		if _, err := os.Stat(p); err == nil {
			lhPath = p
			break
		}
	}
	if lhPath == "" {
		return nil, fmt.Errorf("no lefthook.yml found at %s", repoRoot)
	}

	data, err := os.ReadFile(lhPath)
	if err != nil {
		return nil, fmt.Errorf("reading lefthook config: %w", err)
	}
	var lh lefthookFile
	if err := yaml.Unmarshal(data, &lh); err != nil {
		return nil, fmt.Errorf("parsing lefthook config: %w", err)
	}

	res := &Result{}

	type eventHook struct {
		event string
		hook  lefthookHook
	}
	hooks := []eventHook{
		{"pre-commit", lh.PreCommit},
		{"pre-push", lh.PrePush},
		{"commit-msg", lh.CommitMsg},
	}

	for _, eh := range hooks {
		for name, cmd := range eh.hook.Commands {
			var match []string
			if cmd.Glob != "" {
				match = expandGlob(cmd.Glob)
			}
			res.Entries = append(res.Entries, manifest.Entry{
				Name:    fmt.Sprintf("lefthook-%s-%s", eh.event, name),
				Event:   eh.event,
				Match:   match,
				Command: cmd.Run,
			})
		}
	}

	if len(res.Entries) > 0 {
		res.warn("lefthook parallelism settings are not migrated — all hooks will run sequentially in v1")
	}

	return res, nil
}
