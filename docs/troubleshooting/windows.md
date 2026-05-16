# Windows

## Stash pop fails with "permission denied"

**Cause:** Antivirus or file locking prevents stash pop.

**Solution:** Use `--no-stash` flag to skip the stash cycle.

## Argument length exceeds limit

**Cause:** Too many files for Windows command line (~8000 char limit).

**Solution:** hookset automatically chunks file batches on Windows. No action needed.

## Git Bash Recommended

For best compatibility, use Git Bash or WSL instead of cmd/PowerShell.

## Installation

PowerShell:

```powershell
irm https://raw.githubusercontent.com/bulga138/hookset/master/scripts/install.ps1 | iex
```

Or download manually from GitHub releases.
