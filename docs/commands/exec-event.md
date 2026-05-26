# exec-event

Per-event orchestrator for alpha parallel mode.

## Synopsis

```
hookset exec-event <event> [-- <git-args>...]
```

> **This command is called by git, not by users directly.**
> It is written into `.git/config` by `hookset init` when parallel mode is active.

## Description

`hookset exec-event` is the per-event entry point for the alpha parallel orchestrator
(A3+A4). When `hookset init` detects `experimental = ["parallel"]` in `.hookset.toml`,
it installs a single git hook per event that calls `hookset exec-event <event>` instead
of N individual `hookset exec` calls.

On invocation it:

1. Reads `.hookset.toml` and filters entries for the given event.
2. Applies `HOOKSET_SKIP` / `HOOKSET_ONLY` env-var filters.
3. If parallel mode is active: runs all hooks concurrently, buffers their output, and
   flushes in definition order so terminal output is deterministic.
4. If parallel mode is not active: falls back to serial execution.
5. Prints the alpha banner once per invocation when parallel mode is on.

## Enabling parallel mode

Add the `experimental` key to `.hookset.toml`:

```toml
[hookset]
experimental = ["parallel"]

[[hooks]]
name    = "eslint"
event   = "pre-commit"
command = "npx eslint --cache ."

[[hooks]]
name    = "typecheck"
event   = "pre-commit"
command = "tsc --noEmit"
```

Then re-run `hookset init`. Both hooks now run concurrently on `pre-commit`.

## Alpha banner

When parallel mode is active, the following warning is printed to stderr before each run:

```
[hookset] ⚠ alpha: parallel mode enabled — behaviour may change in future versions
```

## Constraints

- `interactive = true` is **incompatible** with parallel mode on the same event.
  `hookset init` (and `exec-event` at runtime) will error if this combination is detected.
- Passthrough hooks (arg-style events like `commit-msg`, `pre-rebase`) run serially even in
  parallel mode, because git's positional arguments cannot be shared between processes.

## Environment variables

| Variable | Effect |
|----------|--------|
| `HOOKSET=0` | Disable all hooks |
| `HOOKSET_SKIP_IN_CI=1` | Skip hooks when a CI environment is detected |
| `HOOKSET_SKIP=name1,name2` | Skip specific hooks by name |
| `HOOKSET_ONLY=name1,name2` | Run only the named hooks |
| `HOOKSET_EXPERIMENTAL=parallel` | Enable parallel mode without editing `.hookset.toml` |

## What hookset init writes

When parallel mode is enabled, `hookset init` installs **one** gitconfig entry per event:

```
hookset-pre-commit → hookset exec-event pre-commit "$@"
```

instead of the normal per-entry:

```
eslint    → hookset exec --name eslint -- sh -c 'npx eslint …'
typecheck → hookset exec --name typecheck -- sh -c 'tsc …'
```

## See also

- [`hookset init`](init.md) — install hooks from `.hookset.toml`
- [`hookset run`](run.md) — manually trigger an event
- [Configuration reference](../getting-started/configuration.md)
