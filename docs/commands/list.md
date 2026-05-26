# list

List configured hooks.

## Synopsis

```
hookset list [--event <event>] [--json] [--porcelain]
```

## Description

Shows all hooks from `.hookset.toml` grouped by git hook event, in the
canonical event order:

`pre-commit` → `pre-push` → `commit-msg` → `prepare-commit-msg` →
`pre-rebase` → `post-checkout` → `post-merge` → `post-rewrite` → alphabetical

Events with no hooks are not shown.

## Flags

| Flag | Description |
|------|-------------|
| `--event <event>` | Filter output to a single event |
| `--json` | Output as a JSON array |
| `--porcelain` | One line per hook: `<event>\t<name>\t<command>` |

## Examples

```bash
# All hooks, grouped by event
hookset list

# Only pre-commit hooks
hookset list --event pre-commit

# Machine-readable
hookset list --json
hookset list --porcelain

# Count hooks per event
hookset list --porcelain | awk -F'\t' '{print $1}' | sort | uniq -c
```

## Output format (default)

```
pre-commit
  eslint          npx eslint --cache --fix {staged_files}   [*.ts, *.js]
  prettier        npx prettier --write {staged_files}       [*.ts, *.css, *.md]

pre-push
  typecheck       tsc --noEmit
  go-test         go test ./...
```

Columns: name, command (truncated at 50 chars), match patterns.

## JSON output

```json
[
  {
    "name": "eslint",
    "event": "pre-commit",
    "command": "npx eslint --cache --fix {staged_files}",
    "match": ["*.ts", "*.js"],
    "passthrough": false,
    "enabled": true
  }
]
```

## See also

- [`hookset add`](add.md) -- add a hook
- [`hookset doctor`](doctor.md) -- verify hooks are installed
