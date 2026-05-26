# Migrating from husky

## Automated migration

```bash
hookset migrate --from husky --reset-hooks-path
```

This reads your `.husky/<event>` scripts and generates `.hookset.toml` entries
automatically. Use `--dry-run` first to preview.

## What changes

### Before (husky v9)

```
package.json
  "devDependencies": {
    "husky": "^9.0.0",
    "lint-staged": "^15.0.0"
  },
  "scripts": {
    "prepare": "husky"
  }

.husky/
├── pre-commit              ← runs `npx lint-staged`
└── commit-msg              ← runs `npx commitlint --edit "$1"`

.lintstagedrc.json
  { "*.ts": ["eslint --fix", "prettier --write"] }
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
passthrough = true   # git passes the commit message file path as $1
```

```bash
# Install hooks (once per clone, not per npm install)
hookset init
```

## Key differences

| husky | hookset |
|-------|---------|
| `npm install` triggers `prepare` which writes `.husky/` | `hookset init` writes `[hook]` entries to `.git/config` |
| Every contributor must `npm install` | Every contributor must have hookset on PATH |
| `core.hooksPath = .husky` routes all hooks to scripts | Native git config hooks — no `core.hooksPath` |
| File filtering via lint-staged (separate tool) | `match = [...]` built into hookset |
| `"$1"` forwarded manually in husky script | `passthrough = true` forwards args automatically |

## Removing husky

```bash
# 1. Run the automated migration
hookset migrate --from husky --reset-hooks-path

# 2. Verify the generated .hookset.toml
hookset list

# 3. Remove husky from package.json
npm uninstall husky lint-staged

# 4. Delete the .husky directory
rm -rf .husky

# 5. Remove the prepare script from package.json
#    (delete "prepare": "husky" from scripts)

# 6. Run hookset doctor to confirm everything is clean
hookset doctor
```

## The `core.hooksPath` problem

Husky v5+ sets `core.hooksPath = .husky` so git routes hooks to that directory.
This **overrides** git's native `[hook]` config mechanism. hookset detects this and
warns:

```
[hookset] warning: core.hooksPath = ".husky" is set.
          This overrides git config hooks used by hookset.
          Run with --reset-hooks-path to unset it automatically.
```

Pass `--reset-hooks-path` to fix it automatically:

```bash
hookset migrate --from husky --reset-hooks-path
```

Or fix it manually:

```bash
git config --local --unset core.hooksPath
```
