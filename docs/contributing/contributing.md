# Contributing

Thank you for your interest in contributing to hookset!

## Development Setup

### Prerequisites

- Go 1.24+
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

## Testing

```bash
# All tests
make test

# Single package
go test ./internal/exec/...

# With coverage
go test -cover ./...
```

## Project Structure

```
hookset/
├── cmd/hookset/           # CLI entry point
│   ├── main.go            # Version check, dispatcher
│   └── commands/          # Cobra command implementations
├── internal/
│   ├── exec/              # Staging engine
│   ├── git/               # Git subprocess wrappers
│   ├── gitconfig/         # hook.* config operations
│   ├── manifest/          # .hookset.toml handling
│   ├── migrate/           # Migration parsers
│   ├── scanner/           # Project detection
│   └── toml/              # TOML parser
└── scripts/               # Install scripts
```

## Code Style

- Run `make lint` before submitting
- Use meaningful variable names
- Add doc comments for exported functions

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add --allow-large flag for large file handling
fix: resolve Windows argument length chunking
docs: add troubleshooting guide
```

## Pull Requests

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests (`make test`) and lint (`make lint`)
5. Commit with clear messages
6. Push and open a PR