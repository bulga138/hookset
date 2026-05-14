# list

List configured hooks (wraps `git hook list`).

## Synopsis

```
hookset list [event]
```

## Description

Shows hooks from git's native hook list, augmented with hookset match patterns.

## Examples

```bash
# List all hooks
hookset list

# Filter by event
hookset list pre-commit

# With flag
hookset list --event pre-commit
```

## Output

```
pre-commit
  hook.eslint
  hook.prettier

hookset matches:
  eslint: [*.ts, *.js]
  prettier: [*.ts, *.js, *.css, *.md]
```