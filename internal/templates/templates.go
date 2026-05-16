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

var tsHooksList = []manifest.Entry{
	{Name: "eslint", Event: "pre-commit", Match: []string{"*.ts", "*.tsx", "*.js", "*.jsx"}, Command: "npx eslint --cache --fix"},
	{Name: "prettier", Event: "pre-commit", Match: []string{"*.ts", "*.tsx", "*.js", "*.jsx", "*.json", "*.css", "*.md"}, Command: "npx prettier --write"},
	{Name: "tsc", Event: "pre-commit", Match: []string{"*.ts", "*.tsx"}, Command: "npx tsc --noEmit"},
	{Name: "vitest", Event: "pre-push", Match: []string{"*.test.ts", "*.spec.ts", "*.test.js", "*.spec.js"}, Command: "npx vitest run"},
	{Name: "knip", Event: "pre-commit", Match: []string{"*.ts", "*.tsx", "*.js", "*.jsx"}, Command: "npx knip --no-exit-code"},
	{Name: "commitlint", Event: "commit-msg", Match: []string{".git/COMMIT_EDITMSG"}, Command: "npx commitlint --edit"},
	{Name: "no-secrets", Event: "pre-commit", Match: []string{"*"}, Command: "npx secretlint"},
	{Name: "no-direct-main", Event: "pre-push", Match: []string{}, Command: `bash -c '[[ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]] || (echo "Direct push to main blocked" && exit 1)'`},
}

var pythonHooksList = []manifest.Entry{
	{Name: "ruff-format", Event: "pre-commit", Match: []string{"*.py"}, Command: "ruff format ."},
	{Name: "ruff-lint", Event: "pre-commit", Match: []string{"*.py"}, Command: "ruff check --fix ."},
	{Name: "mypy", Event: "pre-commit", Match: []string{"*.py"}, Command: "mypy --strict ."},
	{Name: "pytest", Event: "pre-push", Match: []string{"*.py"}, Command: "pytest -q"},
	{Name: "bandit", Event: "pre-commit", Match: []string{"*.py"}, Command: "bandit -r . -ll"},
	{Name: "commitlint", Event: "commit-msg", Match: []string{".git/COMMIT_EDITMSG"}, Command: "npx commitlint --edit"},
	{Name: "no-secrets", Event: "pre-commit", Match: []string{"*"}, Command: "detect-secrets scan --baseline .secrets.baseline"},
	{Name: "no-direct-main", Event: "pre-push", Match: []string{}, Command: `bash -c '[[ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]] || (echo "Direct push to main blocked" && exit 1)'`},
}

var goHooksList = []manifest.Entry{
	{Name: "goimports", Event: "pre-commit", Match: []string{"*.go"}, Command: "goimports -w ."},
	{Name: "vet", Event: "pre-commit", Match: []string{"*.go"}, Command: "go vet ./..."},
	{Name: "golangci-lint", Event: "pre-commit", Match: []string{"*.go"}, Command: "golangci-lint run --fix"},
	{Name: "test", Event: "pre-push", Match: []string{"*.go"}, Command: "go test ./... -race -count=1"},
	{Name: "govulncheck", Event: "pre-push", Match: []string{"go.sum", "*.go"}, Command: "govulncheck ./..."},
	{Name: "commitlint", Event: "commit-msg", Match: []string{".git/COMMIT_EDITMSG"}, Command: "npx commitlint --edit"},
	{Name: "no-secrets", Event: "pre-commit", Match: []string{"*"}, Command: "npx secretlint"},
	{Name: "no-direct-main", Event: "pre-push", Match: []string{}, Command: `bash -c '[[ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]] || (echo "Direct push to main blocked" && exit 1)'`},
}

var rustHooksList = []manifest.Entry{
	{Name: "fmt", Event: "pre-commit", Match: []string{"*.rs"}, Command: "cargo fmt --all"},
	{Name: "clippy", Event: "pre-commit", Match: []string{"*.rs"}, Command: "cargo clippy --all-targets --all-features -- -D warnings"},
	{Name: "check", Event: "pre-commit", Match: []string{"*.rs", "Cargo.toml", "Cargo.lock"}, Command: "cargo check --all-targets"},
	{Name: "test", Event: "pre-push", Match: []string{"*.rs"}, Command: "cargo test --all"},
	{Name: "audit", Event: "pre-push", Match: []string{"Cargo.lock"}, Command: "cargo audit"},
	{Name: "commitlint", Event: "commit-msg", Match: []string{".git/COMMIT_EDITMSG"}, Command: "npx commitlint --edit"},
	{Name: "no-secrets", Event: "pre-commit", Match: []string{"*"}, Command: "npx secretlint"},
	{Name: "no-direct-main", Event: "pre-push", Match: []string{}, Command: `bash -c '[[ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]] || (echo "Direct push to main blocked" && exit 1)'`},
}

var javaHooksList = []manifest.Entry{
	{Name: "google-java-format", Event: "pre-commit", Match: []string{"*.java"}, Command: "google-java-format --replace $(git diff --cached --name-only | grep '.java$')"},
	{Name: "checkstyle", Event: "pre-commit", Match: []string{"*.java"}, Command: "mvn checkstyle:check -q"},
	{Name: "spotbugs", Event: "pre-commit", Match: []string{"*.java"}, Command: "mvn spotbugs:check -q"},
	{Name: "test", Event: "pre-push", Match: []string{"*.java"}, Command: "mvn test -q"},
	{Name: "pmd", Event: "pre-commit", Match: []string{"*.java"}, Command: "mvn pmd:check -q"},
	{Name: "commitlint", Event: "commit-msg", Match: []string{".git/COMMIT_EDITMSG"}, Command: "npx commitlint --edit"},
	{Name: "no-secrets", Event: "pre-commit", Match: []string{"*"}, Command: "npx secretlint"},
	{Name: "no-direct-main", Event: "pre-push", Match: []string{}, Command: `bash -c '[[ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]] || (echo "Direct push to main blocked" && exit 1)'`},
}

var cSharpHooksList = []manifest.Entry{
	{Name: "dotnet-format", Event: "pre-commit", Match: []string{"*.cs", "*.csproj"}, Command: "dotnet format --verify-no-changes"},
	{Name: "roslyn-analyzers", Event: "pre-commit", Match: []string{"*.cs"}, Command: "dotnet build /warnaserror"},
	{Name: "csharpier", Event: "pre-commit", Match: []string{"*.cs"}, Command: "dotnet csharpier --check ."},
	{Name: "test", Event: "pre-push", Match: []string{"*.cs"}, Command: "dotnet test --no-build -q"},
	{Name: "security-scan", Event: "pre-push", Match: []string{"*.csproj", "packages.lock.json"}, Command: "dotnet list package --vulnerable --include-transitive"},
	{Name: "commitlint", Event: "commit-msg", Match: []string{".git/COMMIT_EDITMSG"}, Command: "npx commitlint --edit"},
	{Name: "no-secrets", Event: "pre-commit", Match: []string{"*"}, Command: "npx secretlint"},
	{Name: "no-direct-main", Event: "pre-push", Match: []string{}, Command: `bash -c '[[ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]] || (echo "Direct push to main blocked" && exit 1)'`},
}

var rubyHooksList = []manifest.Entry{
	{Name: "rubocop", Event: "pre-commit", Match: []string{"*.rb", "Gemfile"}, Command: "bundle exec rubocop --autocorrect"},
	{Name: "brakeman", Event: "pre-commit", Match: []string{"*.rb"}, Command: "bundle exec brakeman -q --no-pager"},
	{Name: "rspec", Event: "pre-push", Match: []string{"*.rb", "*_spec.rb"}, Command: "bundle exec rspec --format progress"},
	{Name: "bundle-audit", Event: "pre-push", Match: []string{"Gemfile.lock"}, Command: "bundle exec bundle-audit check --update"},
	{Name: "reek", Event: "pre-commit", Match: []string{"*.rb"}, Command: "bundle exec reek ."},
	{Name: "commitlint", Event: "commit-msg", Match: []string{".git/COMMIT_EDITMSG"}, Command: "npx commitlint --edit"},
	{Name: "no-secrets", Event: "pre-commit", Match: []string{"*"}, Command: "npx secretlint"},
	{Name: "no-direct-main", Event: "pre-push", Match: []string{}, Command: `bash -c '[[ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]] || (echo "Direct push to main blocked" && exit 1)'`},
}

var monorepoHooksList = []manifest.Entry{
	{Name: "root-lint", Event: "pre-commit", Match: []string{"*.md"}, Command: "markdownlint"},
}
