// Package templates provides embedded starter configurations for hookset init.
package templates

import (
	"fmt"
	"strings"

	"github.com/bulga138/hookset/internal/manifest"
)

// Template represents a hookset configuration template.
type Template struct {
	Name        string
	Description string
	Hooks       []manifest.Entry // Individual hooks in the template
}

// All returns all available templates.
func All() []Template {
	return []Template{
		{
			Name:        "typescript",
			Description: "TypeScript project with ESLint and Prettier",
			Hooks:       tsHooksList,
		},
		{
			Name:        "python",
			Description: "Python project with Ruff and Black",
			Hooks:       pythonHooksList,
		},
		{
			Name:        "go",
			Description: "Go project with go fmt and golangci-lint",
			Hooks:       goHooksList,
		},
		{
			Name:        "rust",
			Description: "Rust project with cargo check and clippy",
			Hooks:       rustHooksList,
		},
		{
			Name:        "java",
			Description: "Java project with SpotBugs and Checkstyle",
			Hooks:       javaHooksList,
		},
		{
			Name:        "csharp",
			Description: "C# project with dotnet format and analyzers",
			Hooks:       cSharpHooksList,
		},
		{
			Name:        "ruby",
			Description: "Ruby project with RuboCop and Brakeman",
			Hooks:       rubyHooksList,
		},
		{
			Name:        "monorepo",
			Description: "Monorepo with package-specific hooks",
			Hooks:       monorepoHooksList,
		},
	}
}

// Get returns a template by name (case-insensitive).
func Get(name string) *Template {
	name = strings.ToLower(name)
	for _, t := range All() {
		if strings.ToLower(t.Name) == name {
			return &t
		}
	}
	return nil
}

// GenerateEntries returns manifest entries for the given template names.
func GenerateEntries(names []string) ([]manifest.Entry, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("no templates specified")
	}

	var entries []manifest.Entry
	seen := map[string]bool{}

	for _, name := range names {
		t := Get(name)
		if t == nil {
			return nil, fmt.Errorf("unknown template: %s", name)
		}
		// Simple dedupe by checking if we already have content from this template
		if !seen[name] {
			entries = append(entries, t.Hooks...)
			seen[name] = true
		}
	}

	return entries, nil
}

// Generate TOML content for the given template names.
func Generate(names []string) (string, error) {
	entries, err := GenerateEntries(names)
	if err != nil {
		return "", err
	}

	// Convert entries to TOML string
	var sb strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&sb, `[[hooks]]
name    = %q
event   = %q
match   = %v
command = %q

`, e.Name, e.Event, e.Match, e.Command)
	}
	return strings.TrimSpace(sb.String()), nil
}

// tsHooksList — TypeScript / JavaScript project.
// Tools: oxlint (fast Rust-based linter), Prettier, tsc, Vitest.
// oxlint is 50-100× faster than ESLint for large codebases; replace with
// eslint if your project relies on custom ESLint plugins not yet in oxlint.
// Uses {staged_files} so linters only process changed files.
var tsHooksList = []manifest.Entry{
	// Fast Rust-based linter — replace with "npx eslint --cache --fix {staged_files}" if needed.
	{Name: "oxlint", Event: "pre-commit", Match: []string{"*.ts", "*.tsx", "*.js", "*.jsx", "*.mjs", "*.cjs"}, Command: "npx --no-install oxlint --fix {staged_files}"},
	// Format all staged files in one pass.
	{Name: "prettier", Event: "pre-commit", Match: []string{"*.ts", "*.tsx", "*.js", "*.jsx", "*.json", "*.css", "*.md"}, Command: "npx --no-install prettier --write {staged_files}"},
	// Full type check — runs once regardless of which files changed.
	{Name: "tsc", Event: "pre-commit", Match: []string{"*.ts", "*.tsx"}, Command: "npx --no-install tsc --noEmit"},
	// Run tests before push; scope to test files when possible.
	{Name: "vitest", Event: "pre-push", Match: []string{"*.test.ts", "*.spec.ts", "*.test.js", "*.spec.js"}, Command: "npx --no-install vitest run"},
	// Detect unused exports / dead code (optional — remove if too slow).
	{Name: "knip", Event: "pre-push", Match: []string{"*.ts", "*.tsx", "*.js", "*.jsx"}, Command: "npx --no-install knip"},
	// Lint commit messages. passthrough=true so git's $1 (msg file path) is forwarded.
	{Name: "commitlint", Event: "commit-msg", Command: "npx --no-install commitlint --edit", Passthrough: true},
	// Block direct pushes to main / master.
	{Name: "no-direct-main", Event: "pre-push", Command: `sh -c 'branch=$(git rev-parse --abbrev-ref HEAD); if [ "$branch" = "main" ] || [ "$branch" = "master" ]; then echo "Direct push to $branch is blocked." >&2; exit 1; fi'`},
}

// pythonHooksList — Python project.
// Tools: ruff (format + lint in one binary), mypy, pytest, bandit.
// ruff replaces flake8 + black + isort — one tool, much faster.
// Uses {staged_files} so ruff only checks changed .py files on commit.
var pythonHooksList = []manifest.Entry{
	// Format staged Python files; ruff format is compatible with Black.
	{Name: "ruff-format", Event: "pre-commit", Match: []string{"*.py"}, Command: "ruff format {staged_files}"},
	// Lint + auto-fix staged Python files.
	{Name: "ruff-lint", Event: "pre-commit", Match: []string{"*.py"}, Command: "ruff check --fix {staged_files}"},
	// Type checking (full project, not just staged files).
	{Name: "mypy", Event: "pre-commit", Match: []string{"*.py"}, Command: "mypy --strict ."},
	// Run tests before push.
	{Name: "pytest", Event: "pre-push", Match: []string{"*.py", "tests/"}, Command: "pytest -q"},
	// Security scan for common vulnerabilities.
	{Name: "bandit", Event: "pre-push", Match: []string{"*.py"}, Command: "bandit -r . -ll -q"},
	// Lint commit messages.
	{Name: "commitlint", Event: "commit-msg", Command: "npx --no-install commitlint --edit", Passthrough: true},
	// Block direct pushes to main / master.
	{Name: "no-direct-main", Event: "pre-push", Command: `sh -c 'branch=$(git rev-parse --abbrev-ref HEAD); if [ "$branch" = "main" ] || [ "$branch" = "master" ]; then echo "Direct push to $branch is blocked." >&2; exit 1; fi'`},
}

// goHooksList — Go project.
// Tools: gofumpt (strict superset of gofmt), golangci-lint, govulncheck.
// golangci-lint uses --new-from-rev so only new/changed code is linted on commit
// (avoids blocking on pre-existing lint debt in large repos).
var goHooksList = []manifest.Entry{
	// Format staged Go files. gofumpt is a strict superset of gofmt.
	// Install: go install mvdan.cc/gofumpt@latest
	{Name: "gofumpt", Event: "pre-commit", Match: []string{"*.go"}, Command: "gofumpt -w {staged_files}"},
	// Vet the whole project (fast, catches common bugs).
	{Name: "vet", Event: "pre-commit", Match: []string{"*.go"}, Command: "go vet ./..."},
	// Lint only new/changed code so existing debt doesn't block commits.
	{Name: "golangci-lint", Event: "pre-commit", Match: []string{"*.go"}, Command: "golangci-lint run --new-from-rev=HEAD~1"},
	// Full test suite with race detector before push.
	{Name: "test", Event: "pre-push", Match: []string{"*.go"}, Command: "go test ./... -race -count=1"},
	// Vulnerability scan when go.sum changes (dependency updates).
	{Name: "govulncheck", Event: "pre-push", Match: []string{"go.sum"}, Command: "govulncheck ./..."},
	// Lint commit messages.
	{Name: "commitlint", Event: "commit-msg", Command: "npx --no-install commitlint --edit", Passthrough: true},
	// Block direct pushes to main / master.
	{Name: "no-direct-main", Event: "pre-push", Command: `sh -c 'branch=$(git rev-parse --abbrev-ref HEAD); if [ "$branch" = "main" ] || [ "$branch" = "master" ]; then echo "Direct push to $branch is blocked." >&2; exit 1; fi'`},
}

// rustHooksList — Rust project.
// Uses --check flags on commit (no in-place edits) so staged state is preserved.
// In-place formatting on commit can cause the staged snapshot and working tree
// to diverge, which confuses `git diff --cached` and hookset's stash cycle.
var rustHooksList = []manifest.Entry{
	// Check formatting without modifying files (safe with staged-file model).
	{Name: "fmt", Event: "pre-commit", Match: []string{"*.rs"}, Command: "cargo fmt --all -- --check"},
	// Lint with all warnings treated as errors.
	{Name: "clippy", Event: "pre-commit", Match: []string{"*.rs", "Cargo.toml"}, Command: "cargo clippy --all-targets --all-features -- -D warnings"},
	// Fast compile check (no test binary, no run).
	{Name: "check", Event: "pre-commit", Match: []string{"*.rs", "Cargo.toml", "Cargo.lock"}, Command: "cargo check --all-targets"},
	// Full test suite before push.
	{Name: "test", Event: "pre-push", Match: []string{"*.rs"}, Command: "cargo test --all"},
	// Dependency vulnerability scan when Cargo.lock changes.
	{Name: "audit", Event: "pre-push", Match: []string{"Cargo.lock"}, Command: "cargo audit"},
	// Lint commit messages.
	{Name: "commitlint", Event: "commit-msg", Command: "npx --no-install commitlint --edit", Passthrough: true},
	// Block direct pushes to main / master.
	{Name: "no-direct-main", Event: "pre-push", Command: `sh -c 'branch=$(git rev-parse --abbrev-ref HEAD); if [ "$branch" = "main" ] || [ "$branch" = "master" ]; then echo "Direct push to $branch is blocked." >&2; exit 1; fi'`},
}

var javaHooksList = []manifest.Entry{
	{Name: "google-java-format", Event: "pre-commit", Match: []string{"*.java"}, Command: "google-java-format --replace $(git diff --cached --name-only | grep '.java$')"},
	{Name: "checkstyle", Event: "pre-commit", Match: []string{"*.java"}, Command: "mvn checkstyle:check -q"},
	{Name: "spotbugs", Event: "pre-commit", Match: []string{"*.java"}, Command: "mvn spotbugs:check -q"},
	{Name: "test", Event: "pre-push", Match: []string{"*.java"}, Command: "mvn test -q"},
	{Name: "pmd", Event: "pre-commit", Match: []string{"*.java"}, Command: "mvn pmd:check -q"},
	{Name: "commitlint", Event: "commit-msg", Command: "npx commitlint --edit", Passthrough: true},
	{Name: "no-secrets", Event: "pre-commit", Match: []string{"*"}, Command: "npx secretlint"},
	{Name: "no-direct-main", Event: "pre-push", Match: []string{}, Command: `bash -c '[[ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]] || (echo "Direct push to main blocked" && exit 1)'`},
}

var cSharpHooksList = []manifest.Entry{
	{Name: "dotnet-format", Event: "pre-commit", Match: []string{"*.cs", "*.csproj"}, Command: "dotnet format --verify-no-changes"},
	{Name: "roslyn-analyzers", Event: "pre-commit", Match: []string{"*.cs"}, Command: "dotnet build /warnaserror"},
	{Name: "csharpier", Event: "pre-commit", Match: []string{"*.cs"}, Command: "dotnet csharpier --check ."},
	{Name: "test", Event: "pre-push", Match: []string{"*.cs"}, Command: "dotnet test --no-build -q"},
	{Name: "security-scan", Event: "pre-push", Match: []string{"*.csproj", "packages.lock.json"}, Command: "dotnet list package --vulnerable --include-transitive"},
	{Name: "commitlint", Event: "commit-msg", Command: "npx commitlint --edit", Passthrough: true},
	{Name: "no-secrets", Event: "pre-commit", Match: []string{"*"}, Command: "npx secretlint"},
	{Name: "no-direct-main", Event: "pre-push", Match: []string{}, Command: `bash -c '[[ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]] || (echo "Direct push to main blocked" && exit 1)'`},
}

var rubyHooksList = []manifest.Entry{
	{Name: "rubocop", Event: "pre-commit", Match: []string{"*.rb", "Gemfile"}, Command: "bundle exec rubocop --autocorrect"},
	{Name: "brakeman", Event: "pre-commit", Match: []string{"*.rb"}, Command: "bundle exec brakeman -q --no-pager"},
	{Name: "rspec", Event: "pre-push", Match: []string{"*.rb", "*_spec.rb"}, Command: "bundle exec rspec --format progress"},
	{Name: "bundle-audit", Event: "pre-push", Match: []string{"Gemfile.lock"}, Command: "bundle exec bundle-audit check --update"},
	{Name: "reek", Event: "pre-commit", Match: []string{"*.rb"}, Command: "bundle exec reek ."},
	{Name: "commitlint", Event: "commit-msg", Command: "npx commitlint --edit", Passthrough: true},
	{Name: "no-secrets", Event: "pre-commit", Match: []string{"*"}, Command: "npx secretlint"},
	{Name: "no-direct-main", Event: "pre-push", Match: []string{}, Command: `bash -c '[[ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]] || (echo "Direct push to main blocked" && exit 1)'`},
}

// monorepoHooksList — Monorepo with multiple sub-packages.
// Demonstrates cwd for package-scoped hooks so tools run in the right directory,
// plus a shared pre-push hook at the root.
//
// Adapt the cwd paths and commands to match your actual monorepo layout.
var monorepoHooksList = []manifest.Entry{
	// Lint markdown in the repo root (docs, READMEs).
	{Name: "root-lint", Event: "pre-commit", Match: []string{"*.md", "docs/**/*.md"}, Command: "markdownlint {staged_files}"},
	// TypeScript package in packages/frontend — runs oxlint scoped to that directory.
	{Name: "frontend-lint", Event: "pre-commit", Match: []string{"packages/frontend/**/*.ts", "packages/frontend/**/*.tsx"}, Cwd: "packages/frontend", Command: "npx --no-install oxlint --fix {staged_files}"},
	// Python service in services/api — runs ruff scoped to that directory.
	{Name: "api-lint", Event: "pre-commit", Match: []string{"services/api/**/*.py"}, Cwd: "services/api", Command: "ruff check --fix {staged_files}"},
	// Root-level pre-push: run all test suites.
	{Name: "test-all", Event: "pre-push", Command: "sh -c 'npm test --workspaces && go test ./... 2>/dev/null || true'"},
	// Lint commit messages.
	{Name: "commitlint", Event: "commit-msg", Command: "npx --no-install commitlint --edit", Passthrough: true},
	// Block direct pushes to main / master.
	{Name: "no-direct-main", Event: "pre-push", Command: `sh -c 'branch=$(git rev-parse --abbrev-ref HEAD); if [ "$branch" = "main" ] || [ "$branch" = "master" ]; then echo "Direct push to $branch is blocked." >&2; exit 1; fi'`},
}
