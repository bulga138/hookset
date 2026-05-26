# Quick Start

## 1. Install hookset

See [Installation](installation.md) for your platform.

## 2. Bootstrap (recommended)

One command creates `.hookset.toml` and installs hooks:

```bash
hookset bootstrap
```

hookset auto-detects your language (Go, Python, TypeScript, Rust) and picks a
sensible template. To specify a template explicitly:

```bash
hookset bootstrap --template go
hookset bootstrap --template typescript
hookset bootstrap --template python
```

Templates include pre-commit linting/formatting, pre-push tests, and a
`commit-msg` linter. Review `.hookset.toml` after bootstrapping and remove
anything you don't need.

---

## Manual setup

### 2. Create a manifest

In your project root, create `.hookset.toml`:

```toml
[[hooks]]
name    = "eslint"
event   = "pre-commit"
match   = ["*.ts", "*.js", "*.jsx"]
command = "npx eslint --cache --fix"

[[hooks]]
name    = "test"
event   = "pre-push"
command = "go test ./..."

# commit-msg: git passes the message file path as $1 -- use passthrough
[[hooks]]
name        = "commitlint"
event       = "commit-msg"
command     = "npx commitlint --edit"
passthrough = true
```

### 3. Initialize

```bash
hookset init
```

This reads `.hookset.toml` and writes hook entries to `.git/config`.

### 4. Commit

Now git hooks run automatically:

```bash
git add src/app.ts
git commit -m "feat: add new feature"
# → Runs eslint on staged TS/JS files
```

---

## Quick add (no manifest)

Add a one-off personal hook without touching `.hookset.toml`:

```bash
# Sugar form -- no flags needed
hookset add typecheck pre-push "tsc --noEmit"

# Add to the shared manifest instead
hookset add eslint --match "*.ts" --manifest -- npx eslint --cache --fix
hookset init
```

---

## Commands Reference

| Command | Description |
|---------|-------------|
| `hookset bootstrap` | Auto-detect language and create `.hookset.toml` + install |
| `hookset init` | Read `.hookset.toml` and install hooks |
| `hookset add <name> <event> "<cmd>"` | Add a hook (sugar form) |
| `hookset run <event>` | Manually trigger all hooks for an event |
| `hookset list` | Show installed hooks (table, `--json`, `--porcelain`) |
| `hookset doctor` | Diagnose installation problems |
| `hookset remove <name>` | Remove a hook |
| `hookset disable <name>` | Disable without removing |
| `hookset migrate --from husky` | Import from husky/lefthook/lint-staged |

See [Commands](../commands/init.md) for full reference.

---

## Alpha: parallel hooks

Enable parallel execution of hooks on the same event by opting in via
`.hookset.toml`:

```toml
[hookset]
experimental = ["parallel"]
```

Then re-run `hookset init`. hookset will install a single per-event orchestrator
instead of per-entry hooks. Output is buffered and flushed in definition order.

See [`hookset exec-event`](../commands/exec-event.md) for full details and constraints.
