# From lint-staged

Migrate your lint-staged configuration to hookset.

## How it works

1. hookset finds your lint-staged config
2. Converts each glob→command mapping to a hook entry
3. Generates `.hookset.toml` entries

## Supported Config Locations

- `.lintstagedrc.json`
- `.lintstagedrc`
- `package.json["lint-staged"]`

## Limitations

- Function syntax (`"*.ts": () => {...}`) cannot be migrated automatically
- Commands using shell features may need manual adjustment

## Example

Before (`.lintstagedrc.json`):
```json
{
  "*.ts": "npx tsc --noEmit",
  "*.js": ["prettier --write", "eslint --fix"]
}
```

After (`hookset migrate --from lint-staged --yes`):
```toml
[[hooks]]
name = "lint-staged-1"
event = "pre-commit"
match = ["*.ts"]
command = "npx tsc --noEmit"

[[hooks]]
name = "lint-staged-2"
event = "pre-commit"
match = ["*.js"]
command = "prettier --write"

[[hooks]]
name = "lint-staged-3"
event = "pre-commit"
match = ["*.js"]
command = "eslint --fix"
```