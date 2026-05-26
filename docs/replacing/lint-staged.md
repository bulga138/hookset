# Migrating from lint-staged

lint-staged is not a standalone hook manager — it runs inside another tool's hook
(usually husky). The migration path is: **lint-staged → hookset** replaces both
lint-staged and husky in one step.

## Automated migration

```bash
hookset migrate --from lint-staged --reset-hooks-path
```

This reads your `.lintstagedrc` / `package.json["lint-staged"]` and generates
`.hookset.toml` entries. Use `--dry-run` first to preview.

## What changes

### Before (lint-staged + husky)

```json
// package.json
{
  "devDependencies": {
    "husky": "^9.0.0",
    "lint-staged": "^15.0.0"
  },
  "scripts": { "prepare": "husky" },
  "lint-staged": {
    "*.{ts,tsx,js}": ["eslint --fix", "prettier --write"],
    "*.{css,md,json}": "prettier --write"
  }
}
```

```sh
# .husky/pre-commit
npx lint-staged
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
match   = ["*.ts", "*.tsx", "*.js", "*.css", "*.md", "*.json"]
command = "npx --no-install prettier --write {staged_files}"
```

```bash
hookset init
```

## Key differences

| lint-staged (+ husky) | hookset |
|-----------------------|---------|
| Two tools to install and configure | One tool |
| `npm install` on every clone | `hookset init` once per clone |
| Config in `package.json` or `.lintstagedrc` | Config in `.hookset.toml` |
| Only pre-commit file filtering | Any hook event with file filtering |
| No commit-msg / pre-push support natively | Full hook support |

## The `{staged_files}` token

lint-staged automatically passes matched staged files to each command. hookset
provides the same via the `{staged_files}` token:

```toml
# lint-staged equivalent
[[hooks]]
name    = "eslint"
event   = "pre-commit"
match   = ["*.ts"]
command = "npx eslint --fix {staged_files}"
#                          ^^^^^^^^^^^^^^
# expands to the space-separated list of staged .ts files
# if no .ts files are staged, this hook is skipped entirely
```

Use `{staged_files_or_default}` to fall back to `.` when no files match (useful for
tools that need to run even on non-file hooks like `tsc --noEmit`).

## Removing lint-staged and husky

```bash
# 1. Run the migration
hookset migrate --from lint-staged --reset-hooks-path

# 2. Verify
hookset list

# 3. Remove both tools
npm uninstall husky lint-staged

# 4. Delete husky config
rm -rf .husky

# 5. Remove prepare script and lint-staged config from package.json

# 6. Verify clean state
hookset doctor
```
