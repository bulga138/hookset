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

No per-language tooling pollution. No runtime dependencies. One binary, installed once per machine.

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

## Configuration

Hooks are defined in `.hookset.toml`:

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
```

Run `hookset init` after editing. The tool writes self-checking wrapper commands into your local `.git/config` — if `hookset` isn't installed, commits fail with a human-readable message instead of "command not found."

## How It Works

When Git runs a hook, it calls `hookset exec`, which:

1. Collects staged files (`git diff --cached`)
2. Filters by `--match` patterns (using Git-native pathspec globs)
3. Stashes unstaged changes (`--keep-index`)
4. Runs your linter/formatter with matched files as arguments
5. Re-stages any files modified by the command
6. Pops the stash
7. Propagates the command's exit code

The stash/pop cycle ensures your working tree is never polluted, even if the linter fails.

## Commands

| Command                             | Description                                                            |
| ----------------------------------- | ---------------------------------------------------------------------- |
| `hookset init`                      | Read `.hookset.toml` and write hooks to `.git/config`                  |
| `hookset init --recursive`          | Discover .hookset.toml in subdirectories and merge                     |
| `hookset init --template <lang>`    | Generate config from template (typescript, python, go, rust, monorepo) |
| `hookset add <name> -- <cmd>`       | Add a hook (personal: `--local`, team: `--manifest`)                   |
| `hookset remove <name>`             | Remove a hook from config                                              |
| `hookset disable <name>`            | Temporarily disable a hook without removing it                         |
| `hookset enable <name>`             | Re-enable a previously disabled hook                                   |
| `hookset list`                      | List configured hooks (wraps `git hook list`)                          |
| `hookset exec --match ... -- <cmd>` | Staging engine (called by Git, rarely used directly)                   |
| `hookset exec --summary`            | Print summary table after hook runs                                    |
| `hookset migrate --from <tool>`     | Convert from husky, lefthook, or lint-staged                           |
| `hookset check`                     | Validate .hookset.toml syntax and check commands                       |
| `hookset update`                    | Self-update hookset binary from GitHub Releases                        |
| `hookset version`                   | Print version, commit, and build timestamp                             |

### Environment Variables

- `HOOKSET_SKIP` - Comma-separated hook names to skip (e.g., `HOOKSET_SKIP=eslint,prettier`)

## Requirements

- **Git 2.54 or later** (native `[hook]` config sections)
- **Go 1.25** (for development builds)

## Install

```bash
# macOS
brew install bulga138/homebrew-hookset/hookset

# Linux / WSL
curl -fsSL https://raw.githubusercontent.com/bulga138/hookset/master/scripts/install.sh | sh

# Windows
irm https://raw.githubusercontent.com/bulga138/hookset/master/scripts/install.ps1 | iex

# Air-gapped environments
HOOKSET_BINARY_PATH=/path/to/hookset curl -fsSL https://raw.githubusercontent.com/bulga138/hookset/master/scripts/install.sh | sh
```

## Development

```bash
git clone https://github.com/bulga138/hookset
cd hookset

make build          # production build with version info
make build-dev      # fast development build
make test           # run all tests
make run            # build and run
```

### Project Structure

```
hookset/
├── cmd/hookset/           # CLI entry point (cobra commands)
│   ├── main.go            # version check, dispatcher
│   └── commands/          # add, exec, init, list, migrate, remove
├── internal/
│   ├── git/               # git subprocess wrappers, version check, pathspec matching
│   ├── gitconfig/         # read/write hook.* config entries (idempotent)
│   ├── exec/              # staging engine (stash/filter/restage)
│   ├── manifest/          # .hookset.toml read/write/merge
│   ├── migrate/           # husky, lefthook, lint-staged parsers
│   └── scanner/           # project detection for bootstrap
├── scripts/               # install.sh, install.ps1
├── .goreleaser.yml        # cross-compilation, Homebrew tap
├── HOMEBREW_TAP.md        # Homebrew tap documentation
└── .hookset.toml          # this repo's own hook config (self-hosting)
```

## Known Limitations (v1)

- **Partial staging:** If a file has some hunks staged and others not, the formatter will re-stage the entire file — including previously unstaged hunks. This matches lint-staged's behavior.
- **Linked worktrees:** Not supported. `hookset exec` will detect and refuse to run in a linked worktree.
- **Large files (>10 MB):** A warning is emitted; pass `--allow-large` to suppress.

## License

MIT
