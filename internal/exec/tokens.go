// Package exec — token expansion for hook commands.
//
// Supported tokens:
//
//	{staged_files}          Space-separated single-quoted list of matched staged files.
//	                        Example: 'src/a.ts' 'src/b.ts'
//	{staged_files_or_default}
//	                        Same as {staged_files} when there are matched files;
//	                        expands to "." (run-on-everything fallback) when the list is empty.
//	{event}                 The git hook event name (e.g. "pre-commit").
//	{branch}                The current branch name (git rev-parse --abbrev-ref HEAD).
//
// Escaping:
//
//	{{token}}               Literal braces — e.g. {{staged_files}} → {staged_files}
//
// Append behaviour:
//
//	When a command contains at least one recognized token, matched files are NOT
//	appended to the command as trailing arguments by the staging engine. This
//	lets users write:
//
//	    command = "eslint --fix {staged_files}"
//
//	instead of the default tail-append:
//
//	    command = "eslint --fix"   # files appended automatically
package exec

import (
	"strings"
)

// expandTokens replaces recognized {token} placeholders in the command string.
// It returns the expanded string and a boolean indicating whether any token was
// found (so the caller knows not to append files as trailing arguments).
//
// Parameters:
//
//	command   — the raw command string from .hookset.toml (e.g. "eslint --fix {staged_files}")
//	files     — the matched staged files (already single-quoted by shellQuotePaths)
//	event     — the hook event name (e.g. "pre-commit"), empty string if unknown
//	branch    — the current branch name, empty string if unknown
func expandTokens(command string, files []string, event, branch string) (expanded string, hadToken bool) {
	// Fast path: no braces at all.
	if !strings.ContainsAny(command, "{}") {
		return command, false
	}

	stagedFilesVal := shellQuotePaths(files)
	stagedFilesOrDefault := stagedFilesVal
	if stagedFilesOrDefault == "" {
		stagedFilesOrDefault = "."
	}

	// Two-pass replacement so that {{…}} escapes survive the first pass intact.
	// Pass 1: replace {{token}} with a sentinel that cannot appear in user commands.
	// Pass 2: replace {token} with values.
	// Pass 3: restore sentinels to literal {token}.
	const sentinel = "\x00HS\x00"

	// Protect escaped double-brace sequences.
	s := strings.ReplaceAll(command, "{{", sentinel+"OPEN"+sentinel)
	s = strings.ReplaceAll(s, "}}", sentinel+"CLOSE"+sentinel)

	// Now perform token substitution. Track whether any token was consumed.
	type tokenPair struct {
		placeholder string
		value       string
	}
	tokens := []tokenPair{
		{"{staged_files}", stagedFilesVal},
		{"{staged_files_or_default}", stagedFilesOrDefault},
		{"{event}", event},
		{"{branch}", branch},
	}

	for _, t := range tokens {
		if strings.Contains(s, t.placeholder) {
			s = strings.ReplaceAll(s, t.placeholder, t.value)
			hadToken = true
		}
	}

	// Restore escaped braces to their literal form.
	s = strings.ReplaceAll(s, sentinel+"OPEN"+sentinel, "{")
	s = strings.ReplaceAll(s, sentinel+"CLOSE"+sentinel, "}")

	return s, hadToken
}

// shellQuotePaths returns a space-separated string of POSIX single-quoted paths.
// Each path has its embedded single-quotes escaped with the standard POSIX idiom
// (end quote, escaped quote, re-open quote: ' → '\'' ).
//
// Example:
//
//	["src/a.ts", "src/b b.ts", "it's.go"] →
//	`'src/a.ts' 'src/b b.ts' 'it'\''s.go'`
func shellQuotePaths(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	quoted := make([]string, len(paths))
	for i, p := range paths {
		quoted[i] = "'" + strings.ReplaceAll(p, "'", `'\''`) + "'"
	}
	return strings.Join(quoted, " ")
}
