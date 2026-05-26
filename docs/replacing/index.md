# Replacing hook managers with hookset

hookset is a **drop-in replacement** for husky, lefthook, and lint-staged. It uses
**native git 2.54 config-based hooks** (`[hook "name"]` blocks in `.git/config`) so
projects ship only `.hookset.toml` — no per-project npm/Python/Go dependencies, no
`prepare` scripts, no bootstrap step.

---

## Why git 2.54 changed everything

Before git 2.54, git hooks lived as executable scripts in `.git/hooks/<event>`. There
was no standard way to define them in a config file or share them across a team without
a per-project tool:

| Era | Mechanism | Per-project dependency |
|-----|-----------|----------------------|
| Before tools | `.git/hooks/<event>` scripts | None, but not shareable |
| husky v4 | `package.json` `"husky"` + `.huskyrc` | `npm install husky` |
| husky v5+ | `.husky/<event>` scripts + `core.hooksPath` | `npm install husky` + `prepare` script |
| lefthook | `lefthook.yml` + `lefthook install` | `gem install lefthook` or `brew install` |
| lint-staged | `.lintstagedrc` + husky integration | `npm install lint-staged` |
| **git 2.54** | `[hook "name"]` in `.git/config` | **None** |

Git 2.54 (released April 2026) introduced `[hook "<friendly-name>"]` blocks that git
reads and fires automatically — no external tool needed to intercept the hook dispatch.
hookset writes these blocks and adds a self-checking wrapper so you get a clear install
message when `hookset` is not on `PATH`.

---

## Side-by-side comparison

| Feature | husky v9 | lefthook | lint-staged | hookset |
|---------|----------|----------|-------------|---------|
| Config file | `.husky/<event>` scripts | `lefthook.yml` | `.lintstagedrc` | `.hookset.toml` |
| Install step | `npm install husky` + `prepare` | `brew/gem install` + `lefthook install` | `npm install lint-staged` | `brew install hookset` (once per machine) |
| Per-project dependency | Yes (`package.json`) | Yes (`Gemfile` or `brew`) | Yes (`package.json`) | **No** |
| Language-agnostic | Partial (Node-first) | Yes | No (Node only) | **Yes** |
| File filtering | Via lint-staged | Via `glob:` | Native | Via `match:` |
| Parallel hooks | No | Yes | No | Alpha (`experimental = ["parallel"]`) |
| Arg-style hooks (commit-msg, etc.) | Scripts only | Yes | No | Yes (`passthrough = true`) |
| Git version required | Any | Any | Any | **≥ 2.54** |
| Mechanism | `core.hooksPath` | `core.hooksPath` | Via husky | Native `[hook]` config |

---

## Migration guides

- [Migrating from husky →](husky.md)
- [Migrating from lefthook →](lefthook.md)
- [Migrating from lint-staged →](lint-staged.md)

---

## One-line install after migration

```bash
# macOS / Linux (Homebrew — once per machine)
brew install bulga138/hookset/hookset

# Windows (Scoop — once per machine)
scoop bucket add hookset https://github.com/bulga138/scoop-hookset
scoop install hookset

# Any platform (audited install script)
curl -fsSL https://github.com/bulga138/hookset/releases/latest/download/install.sh | sh

# go install
go install github.com/bulga138/hookset/cmd/hookset@latest
```

Then, in each repo that has `.hookset.toml`:

```bash
hookset init
```

That's it. No `npm install`, no `prepare` script, no `package.json` entry.
