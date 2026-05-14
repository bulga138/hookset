# Configuration

## `.hookset.toml` Format

The manifest file defines all hooks for a project.

```toml
# Optional: include shared hooks from a dotfiles repo
include = "~/dotfiles/hooks/standard.toml"

[[hooks]]
name = "eslint"
event = "pre-commit"
match = ["*.ts", "*.js"]
command = "npx eslint --cache --fix"

[[hooks]]
name = "prettier"
event = "pre-commit"
match = ["*.ts", "*.js", "*.css", "*.md"]
command = "npx prettier --write"

[[hooks]]
name = "test"
event = "pre-push"
command = "go test ./..."
```

## Fields

### `include` (optional)

Path to a shared hooks file. Supports `~` for home directory.

### `[[hooks]]`

Each hook block requires:

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique hook identifier |
| `event` | Yes | Git hook event (`pre-commit`, `pre-push`, `commit-msg`) |
| `command` | Yes | Command to run |
| `match` | No | File patterns to filter (uses git pathspec globs) |

## Events

- `pre-commit` — Before commit is finalized
- `pre-push` — Before push is sent
- `commit-msg` — Before commit message is saved

## Match Patterns

Use git pathspec globs for `--match`:

```toml
match = ["*.ts", "*.js"]           # TypeScript and JavaScript
match = ["*.{ts,js}"]              # Same as above
match = ["src/**"]                 # All files in src/
match = ["*.css", "*.scss"]        # CSS and SCSS
```

## Include Merging

Project hooks override included hooks by name:

```toml
# In your project
include = "~/dotfiles/hooks/shared.toml"

[[hooks]]
name = "eslint"      # Overrides shared eslint with custom settings
event = "pre-commit"
match = ["*.ts"]
command = "eslint --config custom.eslintrc"
```