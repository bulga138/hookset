# Common Issues

## Installation

### `hookset: command not found`

**Cause:** hookset is not in your PATH.

**Solution:**

```bash
# macOS
brew install bulga138/homebrew-hookset/hookset

# Linux/WSL
curl -fsSL https://raw.githubusercontent.com/bulga138/hookset/master/scripts/install.sh | sh

# Windows
irm https://raw.githubusercontent.com/bulga138/hookset/master/scripts/install.ps1 | iex
```

### Git 2.54+ required error

**Cause:** Your Git version is older than 2.54.

**Solution:**

```bash
# macOS
brew upgrade git

# Ubuntu
sudo add-apt-repository ppa:git-core/ppa
sudo apt-get update && sudo apt-get install git
```

## Hook Execution

### Hook fails with "linked worktrees are not supported"

**Cause:** You're running hookset from a linked worktree.

**Solution:** Run from the main worktree instead. Linked worktrees are not supported in v1.

### Hook fails with "stash push failed"

**Cause:** Conflicting unstaged changes or file locks.

**Solution:** Use `--no-stash` to skip the stash cycle:

```bash
hookset exec --no-stash --match "*.ts" -- npx eslint
```

## Configuration

### `hookset init` does nothing

**Cause:** No `.hookset.toml` exists.

**Solution:** Create one or run the bootstrap wizard:

```bash
hookset init  # triggers TUI wizard if no manifest
```

### Include file not found warning

**Cause:** Referenced include file doesn't exist.

**Solution:** Either create the file or remove the `include` line from `.hookset.toml`. Use `--strict` to make missing includes fatal:

```bash
hookset init --strict
```
