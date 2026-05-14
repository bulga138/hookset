# From husky

Migrate your husky configuration to hookset.

## How it works

1. hookset reads `.husky/<event>` scripts
2. Strips husky boilerplate
3. Classifies commands as file-filtering or non-filtering
4. Generates appropriate hook entries

## File-Filtering Detection

Commands like eslint, prettier, stylelint, biome automatically get `--match` patterns.

| Tool | Applied Match Patterns |
|------|----------------------|
| eslint/prettier/oxlint | `["*.js", "*.jsx", "*.ts", "*.tsx", "*.mjs", "*.cjs"]` |
| stylelint | `["*.css", "*.scss", "*.sass", "*.less"]` |
| tslint | `["*.ts", "*.tsx"]` |
| biome | `["*.js", "*.ts", "*.jsx", "*.tsx", "*.json"]` |

## Non-Filtering Commands

Commands like `tsc --noEmit`, test suites, or build commands run without file filtering.

## Example

Before (`.husky/pre-commit`):
```bash
#!/bin/sh
npx eslint
npx prettier --write
```

After (`hookset migrate --from husky --yes`):
```toml
[[hooks]]
name = "husky-pre-commit"
event = "pre-commit"
match = ["*.js", "*.jsx", "*.ts", "*.tsx", "*.mjs", "*.cjs"]
command = "npx eslint"
```

Note: husky scripts are not modified automatically — you can remove them after migration.