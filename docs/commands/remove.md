# remove

Remove a hook from git config (and optionally the manifest).

## Synopsis

```
hookset remove <name> [flags]
```

## Flags

| Flag | Description |
|------|-------------|
| `--manifest` | Also remove from `.hookset.toml` |
| `--global` | Target global config |

## Examples

```bash
# Remove from local git config
hookset remove eslint

# Remove from config and manifest
hookset remove eslint --manifest

# Remove from global config
hookset remove global-hook --global
```