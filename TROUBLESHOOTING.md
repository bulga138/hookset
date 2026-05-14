# Troubleshooting Guide

Common issues and solutions for hookset.

## Installation

### `hookset: command not found`

**Cause:** hookset is not in your PATH.

**Solution:**

```bash
# macOS/Linux
brew install hookset

# Linux/WSL
curl -fsSL https://raw.githubusercontent.com/bulga138/hookset/master/install.sh | sh

# Windows
irm https://raw.githubusercontent.com/bulga138/hookset/master/install.ps1 | iex
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

# Check version
git --version
```

---

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

This is a Windows-specific workaround for file locking issues.

### Hook blocks commit but working tree looks clean

**Cause:** Linter modified files that were unstaged.

**Explanation:** hookset re-stages the ENTIRE file for any modified file. If you had partially staged changes, the re-stage includes all modifications.

**Workaround:** Use `--no-stash` to preserve your index state, or avoid partial staging with existing work.

---

## Windows-Specific Issues

### Stash pop fails with "permission denied"

**Cause:** Antivirus or file locking prevents stash pop.

**Solution:** Use `--no-stash` flag to skip the stash cycle.

### Argument length exceeds limit

**Cause:** Too many files for Windows command line (~8000 char limit).

**Solution:** hookset automatically chunks file batches on Windows. No action needed.

---

## Migration Issues

### `hookset migrate --from husky` doesn't detect my linter

**Cause:** Custom linter names not in the detection list.

**Solution:** Manually add `--match` patterns to your `.hookset.toml`:

```toml
[[hooks]]
name = "my-linter"
event = "pre-commit"
match = ["*.ts", "*.js"]
command = "my-linter --fix"
```

### Migrated hooks run on all files, not just staged

**Cause:** Hooks without `--match` run on all files.

**Solution:** Add match patterns to your manifest:

```toml
[[hooks]]
name = "eslint"
event = "pre-commit"
match = ["*.ts", "*.js", "*.jsx"]
command = "npx eslint --fix"
```

---

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

---

## Performance

### `hookset exec` is slow with many files

**Cause:** Each hook run involves git stash/pop operations.

**Solution:** Use more specific `--match` patterns to limit scope:

```toml
# Instead of running on all files:
match = ["*"]

# Target specific file types:
match = ["*.ts", "*.js"]
```

---

## CI/CD

### CI fails with "hookset not found"

**Cause:** hookset not installed in CI environment.

**Solution:** Use the GitHub Action or install during CI:

```yaml
# GitHub Actions
- uses: hookset/install-action@v1
  with:
    version: '1.0.0'
- run: hookset init

# Or install manually
- name: Install hookset
  run: curl -fsSL https://raw.githubusercontent.com/bulga138/hookset/master/install.sh | sh
```

For air-gapped CI, set `HOOKSET_BINARY_PATH`:

```yaml
- name: Install hookset (air-gapped)
  env:
    HOOKSET_BINARY_PATH: /path/to/local/hookset
  run: curl -fsSL https://raw.githubusercontent.com/bulga138/hookset/master/install.sh | sh
```

---

## Uninstall

### Remove hookset hooks from a repo

```bash
hookset init --uninstall
```

This removes all hookset-managed hooks from `.git/config`.
