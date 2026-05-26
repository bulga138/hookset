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

hookset supports all standard **client-side** git hook events:

| Event | When it fires |
|-------|---------------|
| `pre-commit` | Before commit is finalized -- ideal for linting and formatting |
| `pre-push` | Before push is sent -- ideal for tests and type-checking |
| `commit-msg` | After commit message is written -- ideal for message linting |
| `prepare-commit-msg` | Before editor opens -- for message templates |
| `pre-rebase` | Before rebase starts |
| `post-checkout` | After branch switch or file checkout |
| `post-merge` | After a successful merge |
| `post-rewrite` | After `git commit --amend` or `git rebase` |
| `applypatch-msg` | During `git am` (patch application) |

> **Server-side hooks** (`pre-receive`, `update`, `post-receive`, `post-update`)
> run on the git server and are **out of scope** for hookset. Use your server's
> branch-protection rules or a tool like gitolite for those.
>
> See [Server-side Hooks](../troubleshooting/common-issues.md#server-side-hooks)
> for more detail.

### Arg-style hooks

Some events pass positional arguments to the hook script rather than operating on
staged files. hookset handles these automatically with `passthrough = true`:

```toml
[[hooks]]
name        = "commitlint"
event       = "commit-msg"
command     = "npx commitlint --edit"
passthrough = true   # git passes the commit message file path as $1
```

When `passthrough = true` (or when the event is one of the arg-style events
above), hookset forwards git's positional arguments to the command and skips
the staging engine entirely. The `match` field is ignored for these hooks.

**Auto-detection:** if the event is `commit-msg`, `prepare-commit-msg`,
`pre-rebase`, `post-checkout`, `post-merge`, `post-rewrite`, or
`applypatch-msg`, hookset sets `passthrough` automatically -- you don't need to
set it explicitly.

## Match Patterns

Use git pathspec globs for `--match`:

```toml
match = ["*.ts", "*.js"]           # TypeScript and JavaScript
match = ["*.{ts,js}"]              # Same as above
match = ["src/**"]                 # All files in src/
match = ["*.css", "*.scss"]        # CSS and SCSS
```

## Token Expansion

Use tokens in `command` to control exactly where files or context are injected:

| Token | Expands to |
|-------|-----------|
| `{staged_files}` | Space-separated single-quoted list of matched staged files |
| `{staged_files_or_default}` | Same as above, or `.` when no files matched |
| `{event}` | The git hook event name (e.g. `pre-commit`) |
| `{branch}` | Current branch name (from `git rev-parse --abbrev-ref HEAD`) |
| `{{token}}` | Literal `{token}` -- double braces escape |

When a command contains at least one token, matched files are **not** appended
as trailing arguments -- the token controls placement.

```toml
# Files injected via token -- useful for tools that require positional args
[[hooks]]
name    = "ruff"
event   = "pre-commit"
match   = ["*.py"]
command = "ruff check --fix {staged_files}"

# Run on everything when no Python files staged (avoids empty-invocation skip)
[[hooks]]
name    = "mypy"
event   = "pre-commit"
match   = ["*.py"]
command = "mypy {staged_files_or_default}"

# Include event name in log output
[[hooks]]
name    = "logger"
event   = "pre-commit"
command = "echo '[{event}] running on branch {branch}'"
```

## How hookset installs hooks (git 2.54)

hookset requires **git ≥ 2.54** and uses the native `[hook "name"]` config syntax
introduced in that release. It writes **only into `.git/config`** -- it never creates
or modifies files under `.git/hooks/`.

Each `[[hooks]]` entry becomes one `[hook "name"]` block in `.git/config`:

```ini
# Written by hookset init
[hook "eslint"]
    event = pre-commit
    command = sh -c 'command -v hookset …; exec hookset exec --name eslint …'

[hook "test"]
    event = pre-push
    command = sh -c 'command -v hookset …; exec hookset exec --name test …'
```

Git 2.54 reads these entries and fires them automatically at the correct point in
the git lifecycle. No shell scripts are placed in `.git/hooks/`.

The `command` value is a self-checking one-liner: if `hookset` is not on `PATH` the
commit fails with a clear installation message rather than a cryptic `command not found`.

### Parallel mode (experimental)

When `experimental = ["parallel"]` is enabled in `[hookset]`, the install layout
changes to use a coordinator per event:

```ini
# Per-entry blocks: enabled=false (visible to git hook list, not fired by git)
[hook "eslint"]
    event   = pre-commit
    command = …
    enabled = false

[hook "prettier"]
    event   = pre-commit
    command = …
    enabled = false

# Coordinator: fires by git, dispatches all pre-commit hooks in parallel
[hook "hookset-pre-commit"]
    event   = pre-commit
    command = sh -c '…; exec hookset exec-event pre-commit "$@"' --
```

This means `git hook list pre-commit` shows all real hook names, while only
`hookset-pre-commit` is actually fired by git. The coordinator handles parallelism
and ordered output flushing.

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