# add

Add a hook to git config or `.hookset.toml`.

## Synopsis

```
hookset add <name> [flags] -- <command>
```

## Description

Add a named hook entry.

- **Without `--manifest`**: writes directly to `.git/config` (personal, uncommitted)
- **With `--manifest`**: appends to `.hookset.toml` only; run `hookset init` to apply
- **With `--global`**: writes to `~/.gitconfig` (applies to all repos on this machine)

## Flags

| Flag | Description |
|------|-------------|
| `--match <pattern>` | File pattern to match (repeatable) |
| `--on <event>` | Git hook event (default: `pre-commit`) |
| `--manifest` | Append to `.hookset.toml` instead of git config |
| `--global` | Write to `~/.gitconfig` |

## Examples

```bash
# Personal hook (uncommitted)
hookset add typecheck --on pre-push -- tsc --noEmit

# Team hook (add to manifest)
hookset add eslint --match "*.ts" --match "*.js" --manifest -- npx eslint --cache --fix
hookset init

# Global hook (this machine only)
hookset add secrets --global --on pre-commit -- detect-secrets scan
```