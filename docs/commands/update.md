# update

Download and install the latest version of hookset.

## Synopsis

```
hookset update [--yes]
```

## Description

Fetches the latest hookset release from GitHub Releases, downloads the
appropriate binary for the current OS and architecture, and replaces the
running executable in-place.

The update is applied atomically: the new binary is written to a temporary
file, verified, and then renamed over the existing binary.

## Flags

| Flag | Description |
|------|-------------|
| `--yes` | Skip the confirmation prompt |

## Examples

```bash
# Check what version would be installed, then confirm
hookset update

# Non-interactive (CI, dotfiles automation)
hookset update --yes
```

## See also

- [`hookset check`](check.md) — check for an update without installing
