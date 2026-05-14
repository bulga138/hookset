# init

Read `.hookset.toml` and write hook entries into local git config.

## Synopsis

```
hookset init [flags]
```

## Description

`hookset init` reads `.hookset.toml` from the repository root, resolves any `include` directives, and writes each hook as a native `[hook "name"]` entry in `.git/config`.

Each written command is a self-checking one-liner: if hookset is not installed the commit will fail with a clear installation message rather than a cryptic "command not found" error.

Running `hookset init` multiple times is safe — it removes and rewrites each hook entry (idempotent).

## Flags

| Flag | Description |
|------|-------------|
| `--strict` | Treat missing include files as fatal errors |
| `--dry-run` | Show what would be installed without modifying config |
| `--interactive` | Force TUI selection mode |
| `--no-interactive` | Skip TUI and install all hooks non-interactively |
| `--uninstall` | Remove all hookset-managed hooks from git config |

## Examples

```bash
# Standard install
hookset init

# Show what would happen
hookset init --dry-run

# Strict mode (fails on missing includes)
hookset init --strict

# Remove all hooks
hookset init --uninstall

# Non-interactive (for CI)
hookset init --no-interactive
```

## Output

```
[hookset] Processing 3 hook(es)
  ✓ Installed: eslint
  ✓ Installed: prettier
  ✓ Installed: test
[hookset] Done. Verify with: hookset list
```