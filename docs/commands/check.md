# check

Check for a newer version of hookset.

## Synopsis

```
hookset check
```

## Description

Queries the GitHub Releases API for the latest published hookset release and
compares it against the running binary's version. Prints a one-line result:

- **Up to date** — running version matches the latest release
- **Update available** — prints the current and latest version with an upgrade command

No flags. No side effects. Safe to run in CI.

## Examples

```bash
hookset check
```

Sample output when an update is available:

```
[hookset] Update available: v1.0.3 → v1.1.0
          Run: hookset update
```

Sample output when up to date:

```
[hookset] hookset v1.1.0 is up to date
```

## See also

- [`hookset update`](update.md) — download and install the latest release
