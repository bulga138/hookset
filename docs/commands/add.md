# add

Add a hook to git config or `.hookset.toml`.

## Synopsis

```
hookset add <name> [flags] -- <command>
hookset add <name> <event> "<command>"
```

## Description

Two forms are accepted:

**Flag form** -- full control over all options:
```bash
hookset add <name> --on <event> [--match <pattern>] [--manifest] [--global] -- <command>
```

**Sugar form** -- quick one-liner for the common case:
```bash
hookset add <name> <event> "<command>"
```

The sugar form writes to `.hookset.toml` (`--manifest` is implied) and
automatically sets `passthrough = true` for arg-style events (`commit-msg`,
`prepare-commit-msg`, `pre-rebase`, `post-checkout`, `post-merge`,
`post-rewrite`, `applypatch-msg`).

Storage options:

- **Without `--manifest`** (flag form only): writes directly to `.git/config` (personal, uncommitted)
- **With `--manifest`** or sugar form: appends to `.hookset.toml`; run `hookset init` to apply
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
# Sugar form -- fast one-liner, writes to .hookset.toml
hookset add typecheck pre-push "tsc --noEmit"
hookset add commitlint commit-msg "npx commitlint --edit"

# Flag form -- personal hook (uncommitted, git config only)
hookset add typecheck --on pre-push -- tsc --noEmit

# Flag form -- team hook (add to manifest)
hookset add eslint --match "*.ts" --match "*.js" --manifest -- npx eslint --cache --fix
hookset init

# Flag form -- global hook (this machine only)
hookset add secrets --global --on pre-commit -- detect-secrets scan
```

## See also

- [`hookset list`](list.md) -- list all configured hooks
- [`hookset init`](init.md) -- apply manifest changes to git config