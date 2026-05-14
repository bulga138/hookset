# hookset

**Git-native hook manager — no Husky, no lint-staged, no committed scripts.**

`hookset` uses Git 2.54's native `[hook]` configuration sections to define and run hooks. It includes its own staged-file filtering and stash/unstash engine, replacing `lint-staged`, `husky`, and `lefthook` with a single, globally-installed binary.

## Why hookset?

| Tool        | What it leaves in your repo           |
| ----------- | ------------------------------------- |
| husky       | `.husky/` directory, `prepare` script |
| lefthook    | `lefthook.yml`, wrapper script        |
| lint-staged | `package.json` entry                  |
| **hookset** | **One file: `.hookset.toml`**         |

## Quick Start

```bash
# Install once per machine
brew install bulga138/homebrew-hookset/hookset

# In any repo with a .hookset.toml
git clone <repo>
cd repo
hookset init        # reads .hookset.toml, writes [hook] sections to .git/config

# Now commit — hooks run automatically
git commit -m "fix: something"
```

## Requirements

- **Git 2.54 or later** (native `[hook]` config sections)
- **Go 1.24** (for development builds)

## Installation

```bash
# macOS
brew install bulga138/homebrew-hookset/hookset

# Linux / WSL
curl -fsSL https://raw.githubusercontent.com/bulga138/hookset/master/install.sh | sh

# Windows
irm https://raw.githubusercontent.com/bulga138/hookset/master/install.ps1 | iex
```
