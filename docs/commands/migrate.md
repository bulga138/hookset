# migrate

Convert an existing hook setup to `.hookset.toml`.

## Synopsis

```
hookset migrate --from <source> [flags]
```

## Description

Reads your current hook configuration (lint-staged, husky, or lefthook) and produces equivalent `.hookset.toml` entries.

## Flags

| Flag | Description |
|------|-------------|
| `--from <source>` | Source tool: `lint-staged`, `husky`, or `lefthook` |
| `--dry-run` | Print what would be migrated without writing files |
| `--yes` | Write `.hookset.toml` and run `hookset init` without prompting |

## Examples

```bash
# Dry run lint-staged migration
hookset migrate --from lint-staged --dry-run

# Migrate from husky
hookset migrate --from husky --yes

# Migrate from lefthook
hookset migrate --from lefthook
```

## Supported Sources

### lint-staged

Reads config from:
- `.lintstagedrc.json`
- `.lintstagedrc`
- `package.json["lint-staged"]`

### husky

Reads `.husky/<event>` scripts and converts them to hookset entries.

- File-filtering hooks (eslint, prettier) get `--match` patterns
- Non-filtering hooks (tsc, tests) run as-is

### lefthook

Reads `lefthook.yml` or `lefthook.yaml` and converts commands.

Note: Parallelism settings are not migrated (all hooks run sequentially in v1).