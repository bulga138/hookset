# Quick Start

## 1. Install hookset

See [Installation](installation.md) for your platform.

## 2. Create a manifest

In your project root, create `.hookset.toml`:

```toml
[[hooks]]
name = "eslint"
event = "pre-commit"
match = ["*.ts", "*.js", "*.jsx"]
command = "npx eslint --cache --fix"

[[hooks]]
name = "test"
event = "pre-push"
command = "go test ./..."
```

## 3. Initialize

```bash
hookset init
```

This reads `.hookset.toml` and writes hook entries to `.git/config`.

## 4. Commit

Now git hooks run automatically:

```bash
git commit -m "feat: add new feature"
# → Runs eslint on staged TS/JS files
```

## Commands Reference

| Command | Description |
|---------|-------------|
| `hookset init` | Read `.hookset.toml` and install hooks |
| `hookset add <name> -- <cmd>` | Add a hook directly |
| `hookset list` | Show installed hooks |
| `hookset remove <name>` | Remove a hook |

See [Commands](../commands/init.md) for full reference.