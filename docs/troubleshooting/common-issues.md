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

# Check version
git --version
```

---

## Hook Execution

### Stash pop failed after a hook that uses `--cache` (e.g. `eslint --cache`)

**Note:** Fixed in the current release. Upgrade hookset to get the fix.

**Root cause (older versions):** The stash was created with `--include-untracked`, which
captured untracked files such as `.eslintcache`. When ESLint regenerated that file during
the run, `git stash pop` refused to overwrite it with the stashed copy, leaving unstaged
changes stranded in the stash.

**Fix applied:** hookset no longer stashes untracked files. Only unstaged changes to
tracked files are stashed. Untracked artifacts written by the hook tool remain in the
working tree and do not cause pop conflicts.

### Hook fails with "linked worktrees are not supported"

**Cause:** You're running hookset from a linked worktree.

**Solution:** Run from the main worktree instead. Linked worktrees are not supported in v1.

### Hook fails with "stash push failed"

**Cause:** Conflicting unstaged changes or file locks.

**Solution:** Use `--no-stash` to skip the stash cycle:

```bash
hookset exec --no-stash --match "*.ts" -- npx eslint
```

### Hook blocks commit but working tree looks clean

**Cause:** Linter modified files that were unstaged.

**Explanation:** hookset re-stages the entire file for any file modified by the command.
If you had partially staged changes, the re-stage includes all modifications.

**Workaround:** Use `--no-stash` to preserve your index state, or avoid partial staging
alongside existing work.

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

**Solution:** Install during CI setup:

```yaml
# GitHub Actions -- install via script
- name: Install hookset
  run: curl -fsSL https://raw.githubusercontent.com/bulga138/hookset/master/scripts/install.sh | sh
- run: hookset init
```

For air-gapped CI, download the binary from the
[Releases page](https://github.com/bulga138/hookset/releases) and place it on `PATH`.

---

## Uninstall

### Remove hookset hooks from a repo

```bash
hookset init --uninstall
```

This removes all hookset-managed hooks from `.git/config`.

---

## Stash / Partial Staging

### Tool modified files that weren't staged -- unstaged changes lost

**Cause:** hookset stashes unstaged changes with `git stash push --keep-index`
before running the tool, then pops the stash afterwards. If the tool writes to
a file that was also unstaged, the stash pop may produce a conflict.

**Solution:**

Use `--no-stash` for that hook to skip the stash cycle:

```toml
[[hooks]]
name    = "prettier"
event   = "pre-commit"
match   = ["*.ts", "*.js"]
command = "npx prettier --write"
```

```bash
# Per-run override (env var)
HOOKSET_NO_STASH=1 git commit -m "fix"
```

Or set it permanently in the entry (planned for a future release -- see NEXT.md).

### Stash pop failed -- changes safe but not restored

hookset prints:

```
[hookset] warning: stash pop failed: …
         Your changes are safe in `git stash list` (most recent entry).
         Recover with: git stash pop
```

Run `git stash pop` to restore. If there are conflicts, resolve them with
`git checkout stash -- <file>` for each affected file, then `git stash drop`.

### Partial staging (hunks) -- re-staged as a whole file

**Cause:** hookset uses `git add <file>` to re-stage files modified by the tool.
If you staged only some hunks of a file, the entire file is re-staged.

**Workaround:** Use `--no-stash` flag, or use `git commit --no-verify` for
one-off commits where you intentionally want partial staging to survive.

---

## Server-side Hooks

hookset manages **client-side** hooks only (those that run on the developer's
machine before or after git operations). The following hooks are **not**
supported by hookset and must be configured on the git server:

| Hook | Where it runs |
|------|---------------|
| `pre-receive` | Server -- on push receive |
| `update` | Server -- per-ref on push |
| `post-receive` | Server -- after push accepted |
| `post-update` | Server -- after refs updated |

For server-side hook management consider:
- [gitolite](https://gitolite.com/) hook rules
- GitHub/GitLab/Bitbucket branch protection rules and server-side CI
- [Gerrit](https://www.gerritcodereview.com/) submit requirements
