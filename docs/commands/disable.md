# disable / enable

Disable or re-enable a hook without removing it.

## Synopsis

```
hookset disable <name>
hookset enable <name>
```

## Description

Toggles `hook.<name>.enabled` between `true` and `false`.

## Flags

| Flag | Description |
|------|-------------|
| `--global` | Target global config |

## Examples

```bash
# Disable a hook temporarily
hookset disable eslint

# Re-enable it
hookset enable eslint

# On global config
hookset disable global-hook --global
```