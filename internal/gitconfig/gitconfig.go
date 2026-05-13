// Package gitconfig reads and writes hook.* git config entries.
// All writes use the remove-then-add strategy so repeated calls are idempotent.
package gitconfig

import (
	"fmt"
	"strings"

	"github.com/bulga138/hookset/internal/git"
)

// Hook represents a single [hook "name"] config block.
type Hook struct {
	Name    string
	Event   string
	Command string
	Matches []string
	Enabled bool   // true unless hook.<name>.enabled = false
	Scope   string // "local", "global", "system" — informational only
}

// sectionKey returns the git config section name for a hook, e.g. `hook.eslint`.
func sectionKey(name string) string {
	return fmt.Sprintf("hook.%s", name)
}

func key(name, field string) string {
	return fmt.Sprintf("hook.%s.%s", name, field)
}

// AddHook writes a complete hook entry at the given scope.
// It first removes any existing hook.<name>.* entries (idempotent per ROADMAP §Phase 2),
// then writes event, command, enabled, and each match pattern as a separate line.
func AddHook(h Hook, scope git.Scope) error {
	// Step 1: Remove existing section entirely (ignore error — may not exist).
	git.Run("config", string(scope), "--remove-section", sectionKey(h.Name))

	// Step 2: Write fields.
	if err := gitSet(scope, key(h.Name, "event"), h.Event); err != nil {
		return err
	}
	if err := gitSet(scope, key(h.Name, "command"), h.Command); err != nil {
		return err
	}
	if !h.Enabled {
		if err := gitSet(scope, key(h.Name, "enabled"), "false"); err != nil {
			return err
		}
	}
	for _, m := range h.Matches {
		// --add preserves existing match lines so we can write multiple.
		r := git.Run("config", string(scope), "--add", key(h.Name, "match"), m)
		if !r.OK {
			return fmt.Errorf("git config --add match failed: %s", r.Stderr)
		}
	}
	return nil
}

// RemoveHook deletes all hook.<name>.* entries at the given scope.
func RemoveHook(name string, scope git.Scope) error {
	r := git.Run("config", string(scope), "--remove-section", sectionKey(name))
	if !r.OK && !strings.Contains(r.Stderr, "No such section") {
		return fmt.Errorf("git config --remove-section failed: %s", r.Stderr)
	}
	return nil
}

// EnableHook sets hook.<name>.enabled to true or false.
func EnableHook(name string, scope git.Scope, enabled bool) error {
	val := "true"
	if !enabled {
		val = "false"
	}
	return gitSet(scope, key(name, "enabled"), val)
}

// GetHooks returns all hooks registered for the given event at the given scope.
// Pass an empty event to return hooks for all events.
func GetHooks(event string, scope git.Scope) ([]Hook, error) {
	// List all hook.*.event keys at this scope.
	r := git.Run("config", string(scope), "--get-regexp", `^hook\..*\.event$`)
	if !r.OK {
		// No hooks configured at this scope — not an error.
		return nil, nil
	}

	var hooks []Hook
	for _, line := range strings.Split(r.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "hook.<name>.event <event-value>"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		keyParts := strings.Split(parts[0], ".") // ["hook", "<name>", "event"]
		if len(keyParts) != 3 {
			continue
		}
		hookEvent := strings.TrimSpace(parts[1])
		if event != "" && hookEvent != event {
			continue
		}
		name := keyParts[1]
		h, err := readHook(name, hookEvent, scope)
		if err != nil {
			return nil, err
		}
		hooks = append(hooks, h)
	}
	return hooks, nil
}

// GetHook returns a single named hook at the given scope, or an error if not found.
func GetHook(name string, scope git.Scope) (Hook, error) {
	eventVal, ok := gitGet(scope, key(name, "event"))
	if !ok {
		return Hook{}, fmt.Errorf("hook %q not found in %s config", name, scope)
	}
	return readHook(name, eventVal, scope)
}

// readHook reads all fields for a named hook from config.
func readHook(name, event string, scope git.Scope) (Hook, error) {
	h := Hook{
		Name:    name,
		Event:   event,
		Enabled: true,
		Scope:   scopeLabel(scope),
	}

	if cmd, ok := gitGet(scope, key(name, "command")); ok {
		h.Command = cmd
	}

	if enabledStr, ok := gitGet(scope, key(name, "enabled")); ok {
		h.Enabled = enabledStr != "false"
	}

	// Multi-value: get all match lines.
	r := git.Run("config", string(scope), "--get-all", key(name, "match"))
	if r.OK && r.Stdout != "" {
		for _, m := range strings.Split(r.Stdout, "\n") {
			if m = strings.TrimSpace(m); m != "" {
				h.Matches = append(h.Matches, m)
			}
		}
	}

	return h, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func gitSet(scope git.Scope, k, v string) error {
	r := git.Run("config", string(scope), k, v)
	if !r.OK {
		return fmt.Errorf("git config set %s failed: %s", k, r.Stderr)
	}
	return nil
}

func gitGet(scope git.Scope, k string) (string, bool) {
	r := git.Run("config", string(scope), "--get", k)
	return r.Stdout, r.OK
}

func scopeLabel(s git.Scope) string {
	switch s {
	case git.ScopeLocal:
		return "local"
	case git.ScopeGlobal:
		return "global"
	case git.ScopeSystem:
		return "system"
	default:
		return string(s)
	}
}

// GetAllHookNames returns all hook names configured at the given scope.
func GetAllHookNames(scope git.Scope) ([]string, error) {
	r := git.Run("config", string(scope), "--get-regexp", `^hook\..*\.event$`)
	if !r.OK {
		// No hooks configured — not an error.
		return nil, nil
	}

	var names []string
	seen := make(map[string]bool)
	for _, line := range strings.Split(r.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "hook.<name>.event <event-value>"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		keyParts := strings.Split(parts[0], ".")
		if len(keyParts) != 3 {
			continue
		}
		name := keyParts[1]
		if !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
	}
	return names, nil
}
