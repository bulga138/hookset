# exec

Run a command against staged files (called by git, not directly by users).

## Synopsis

```
hookset exec [flags] -- <command>
```

## Description

`hookset exec` is the staging engine. Git calls it via the hook command written by `hookset init`.

It:
1. Collects staged files (`git diff --cached`)
2. Filters by `--match` patterns (git ls-files)
3. Stashes unstaged changes to **tracked files** (`--keep-index`). Untracked files (caches, build artifacts, generated files) are left in the working tree during the run and are not modified by hookset.
4. Runs the command with matched files as arguments
5. Re-stages any files modified by the command
6. Pops the stash
7. Exits with the command's exit code

## Flags

| Flag | Description |
|------|-------------|
| `--match <pattern>` | File pattern (repeatable). Empty = match all staged files |
| `--allow-large` | Suppress warning for staged files >10 MB |
| `--no-stash` | Disable stash/pop cycle (escape hatch for Windows locking) |

## Examples

As written into git config by `hookset init`:

```bash
hookset exec --match "*.ts" --match "*.js" -- npx eslint --cache --fix
```

## ⚠️ Partial Staging Note

Step 5 re-stages the ENTIRE file for any file modified by the command. If you stage only part of a file (partial staging), the rest of the file will be included in the re-stage.

Use `--no-stash` only if you need to preserve unstaged changes alongside partial staging.