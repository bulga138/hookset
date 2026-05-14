# Worktrees

## Linked Worktrees Not Supported in v1

hookset v1 does **not support linked worktrees**. When running from a linked worktree, hooks will be skipped.

### Why?

Linked worktrees share the same `.git` directory, which can cause:
- Race conditions between multiple worktrees
- Incorrect stash/pop behavior
- Unpredictable hook execution

### How It Works

hookset detects linked worktrees by checking if `--git-dir` points to `.git/worktrees/<name>`:

```bash
git rev-parse --git-dir
# Main worktree: /path/to/repo/.git
# Linked worktree: /path/to/repo/.git/worktrees/feature
```

### Workaround

For now, run hookset from the **main worktree only**.

### Future Support

Linked worktree support is planned for a future version. Track progress in the [GitHub issues](https://github.com/bulga138/hookset/issues).
