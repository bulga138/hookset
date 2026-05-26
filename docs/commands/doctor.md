# doctor

Diagnose your hookset installation.

## Synopsis

```
hookset doctor [flags]
```

## Description

Runs a series of checks and reports any problems with your hookset setup.

| Check | What it verifies |
|-------|-----------------|
| `git-version` | git ≥ 2.9 (required for `hook.*` config key support) |
| `binary-on-path` | `hookset` is on `PATH` and resolves to this executable |
| `manifest-exists` | `.hookset.toml` is present in the repo root |
| `manifest-valid` | `.hookset.toml` is valid TOML and parses cleanly |
| `hooks-installed` | Every entry in the manifest is registered in `git config` |
| `core-hooks-path` | `core.hooksPath` is not set (it bypasses `hook.*` config) |
| `stale-hook-files` | No executable files in `.git/hooks/` left by other tools |
| `sibling-tools` | No competing hook tools in `package.json` |

## Flags

| Flag | Description |
|------|-------------|
| `--strict` | Exit 1 on any warning (useful in CI) |
| `--json` | Output results as a JSON array |

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | All checks passed (or warnings only without `--strict`) |
| `1` | One or more errors found, or warnings with `--strict` |

## Examples

```bash
# Interactive diagnosis
hookset doctor

# Fail CI on warnings too
hookset doctor --strict

# Machine-readable output
hookset doctor --json

# Filter to non-passing checks
hookset doctor --json | jq '.[] | select(.status != "ok")'
```

## Sample output

```
  ✓  git-version              git 2.47.0
  ✓  binary-on-path           /usr/local/bin/hookset
  ✓  manifest-exists          /home/user/project/.hookset.toml
  ✓  manifest-valid           3 hook(s) defined
  ⚠  hooks-installed          manifest hooks not in git config: eslint — run: hookset init
  ✓  core-hooks-path          core.hooksPath not set
  ✓  stale-hook-files         .git/hooks/ has no conflicting executables
  ✓  sibling-tools            no competing hook tools found in package.json
```

## See also

- [`hookset init`](init.md) — install hooks from manifest
- [`hookset migrate`](migrate.md) — migrate from husky/lefthook/lint-staged
