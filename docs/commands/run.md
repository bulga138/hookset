# run

Manually trigger all hooks for a git event.

## Synopsis

```
hookset run <event>
```

## Description

`hookset run` fires all hooks configured for the given git hook event, exactly as
git would — but without requiring a real commit, push, or rebase. It is useful for:

- Testing your hook configuration locally before committing
- Running a specific event from a CI step or Makefile
- Debugging why a hook passes or fails

Each hook is executed via the same wrapper as when git fires it (including the staging
engine, stash/restage cycle, token expansion, and kill-switch env vars).

> **Note:** `hookset run` does not set the full git hook environment
> (`GIT_AUTHOR_NAME`, `GIT_DIR`, etc.). For `commit-msg` hooks the commit message
> file path is not provided — use a real commit to test those.

## Arguments

| Argument | Description |
|----------|-------------|
| `<event>` | Git hook event name (e.g. `pre-commit`, `pre-push`) |

## Examples

```bash
# Run all pre-commit hooks
hookset run pre-commit

# Run all pre-push hooks
hookset run pre-push

# Run hooks for a less common event
hookset run post-merge
```

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | All hooks passed |
| `1` | One or more hooks failed |

## See also

- [`hookset exec`](exec.md) — run a single named hook (called by git, not users)
- [`hookset list`](list.md) — show all configured hooks
- [`hookset doctor`](doctor.md) — diagnose configuration problems
