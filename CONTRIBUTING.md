# Contributing to hookset

Thank you for your interest in contributing to hookset!

## Project Structure

```
hookset/
├── cmd/hookset/           # CLI entry point
│   ├── main.go            # Version check, dispatcher
│   └── commands/          # Cobra command implementations
│       ├── add.go         # hookset add
│       ├── exec.go        # hookset exec
│       ├── init.go        # hookset init
│       ├── manage.go      # remove, disable, enable, list
│       └── migrate.go     # hookset migrate
├── internal/
│   ├── exec/              # Staging engine (stash/filter/restage)
│   ├── git/               # Git subprocess wrappers
│   ├── gitconfig/         # hook.* config operations
│   ├── manifest/          # .hookset.toml handling
│   ├── migrate/           # Migration parsers (husky, lefthook, lint-staged)
│   ├── scanner/           # Project detection for bootstrap
│   └── toml/              # TOML parser
├── scripts/               # Install scripts
└── .github/
    └── workflows/         # CI/CD pipelines
```

## Development Setup

### Prerequisites

- Go 1.25+
- Git 2.54+
- Make

### Quick Start

```bash
git clone https://github.com/bulga138/hookset
cd hookset

make build-dev  # Fast development build
make test       # Run all tests
make run        # Build and run
```

### Makefile Commands

| Command          | Description                        |
| ---------------- | ---------------------------------- |
| `make build`     | Production build with version info |
| `make build-dev` | Fast development build             |
| `make test`      | Run all tests with verbose output  |
| `make run`       | Build and run hookset              |
| `make lint`      | Run golangci-lint                  |

## Testing

### Writing Tests

- Place tests in `*_test.go` files alongside the code they test
- Use `t.TempDir()` for temporary test files
- For git operations, create an isolated repo with `git init` in temp dir
- Table-driven tests are preferred for complex logic

### Running Tests

```bash
# All tests
make test

# Single package
go test ./internal/exec/...

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...
```

### Integration Tests

Integration tests that require a real git repo use a helper pattern:

```go
func testWithRepo(t *testing.T) (repo, origDir string) {
    origDir, _ = os.Getwd()
    repo = t.TempDir()
    os.Chdir(repo)
    exec.Command("git", "init").Run()
    return repo, origDir
}
```

## Adding New Migration Sources

To add support for a new hook tool in `hookset migrate`:

1. Add a new `Source` constant in `internal/migrate/migrate.go`
2. Implement the migration function following the existing patterns
3. Add tests in `internal/migrate/migrate_test.go`
4. Update the help text in `cmd/hookset/commands/migrate.go`

Example structure:

```go
// Add source constant
const SourceMyTool Source = "mytool"

// Add case in Migrate()
case SourceMyTool:
    return migrateMyTool(repoRoot)

// Implement migration
func migrateMyTool(repoRoot string) (*Result, error) {
    // Parse config file
    // Convert to manifest.Entry slices
    // Return Result with Entries and any Warnings
}
```

## Code Style

- Run `make lint` before submitting
- Use meaningful variable names
- Add doc comments for exported functions
- Keep functions small and focused

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add --allow-large flag for large file handling
fix: resolve Windows argument length chunking
docs: add troubleshooting guide
test: add migration tests for lefthook
```

## Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Run lint (`make lint`)
6. Commit with clear messages
7. Push and open a PR

## Reporting Issues

Include:

- hookset version (`hookset version`)
- Git version (`git --version`)
- Operating system
- Steps to reproduce
- Expected vs actual behavior

## License

MIT - see [LICENSE](LICENSE) for details.
