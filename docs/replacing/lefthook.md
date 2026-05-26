# Migrating from lefthook

## Automated migration

```bash
hookset migrate --from lefthook
```

This reads `lefthook.yml` / `lefthook.yaml` and generates `.hookset.toml` entries.
Use `--dry-run` first to preview.

## What changes

### Before (lefthook)

```yaml
# lefthook.yml
pre-commit:
  parallel: true
  commands:
    eslint:
      glob: "*.{ts,tsx,js}"
      run: npx eslint --cache --fix {staged_files}
    prettier:
      glob: "*.{ts,tsx,js,json,css,md}"
      run: npx prettier --write {staged_files}

commit-msg:
  commands:
    commitlint:
      run: npx commitlint --edit {1}
```

### After (hookset)

```toml
# .hookset.toml

[[hooks]]
name    = "eslint"
event   = "pre-commit"
match   = ["*.ts", "*.tsx", "*.js"]
command = "npx --no-install oxlint --fix {staged_files}"

[[hooks]]
name    = "prettier"
event   = "pre-commit"
match   = ["*.ts", "*.tsx", "*.js", "*.json", "*.css", "*.md"]
command = "npx --no-install prettier --write {staged_files}"

[[hooks]]
name        = "commitlint"
event       = "commit-msg"
command     = "npx --no-install commitlint --edit"
passthrough = true
```

## Key differences

| lefthook | hookset |
|----------|---------|
| `lefthook.yml` (YAML) | `.hookset.toml` (TOML) |
| `lefthook install` writes scripts to `.git/hooks/` | `hookset init` writes `[hook]` blocks to `.git/config` |
| `parallel: true` per-event | `experimental = ["parallel"]` global opt-in |
| `{staged_files}` token | `{staged_files}` and `{staged_files_or_default}` tokens |
| `glob:` for file filtering | `match = [...]` array |
| `{1}` for passthrough args | `passthrough = true` auto-forwards `$@` |

## Notes on parallelism

lefthook's `parallel: true` runs all commands for an event concurrently. hookset
supports this as an alpha feature:

```toml
[hookset]
experimental = ["parallel"]
```

See [exec-event](../commands/exec-event.md) for details and constraints.

## Removing lefthook

```bash
# 1. Run the migration
hookset migrate --from lefthook

# 2. Install the generated hooks
hookset init

# 3. Remove lefthook
brew uninstall lefthook   # or however you installed it
rm lefthook.yml

# 4. Verify
hookset doctor
```
