# hookset Roadmap — v2

## Guiding Principles

- **Zero per-project dependencies** — One `.hookset.toml`, one global binary.
- **Git-native, always** — Build on top of git's native `[hook]` config syntax, never fight it.
- **Opt-in complexity** — Advanced features (parallelism, CI, remote hooks) must never degrade the simple case.

---

## Risk Management

**Phase 1 scope creep.** Mitigation: workspace discovery is limited to declarative patterns only. Build-system-specific discovery (Bazel, Pants, etc.) belongs in Hookpacks. If declarative workspaces prove insufficient for a large user segment, that will be revisited in Phase 4.

**Config schema stability.** The v2.x schema is stable across patch releases within a minor version. Any breaking change requires a minor bump. The VS Code extension (Phase 4) will only ship after the schema has been in production for at least two minor releases and a `hookset schema` command exists to output it as JSON Schema.

---

## Phase 1: Monorepo & Workspace Features

**Goal**: Run commands in the packages affected by a change, not the whole tree.

### 1.1 Workspace Discovery

Detect common workspace definitions and derive `cwd` + `match` bounds per hook.
Build-system-specific tooling (e.g. `bazel query`) is explicitly out of scope — those cases belong in Hookpacks. Core support covers:

| Workspace Type  | Detection File                       | Strategy                                   |
| --------------- | ------------------------------------ | ------------------------------------------ |
| npm/pnpm/yarn   | `package.json` → `workspaces` field  | Expand globs into individual package roots |
| pnpm workspaces | `pnpm-workspace.yaml`                | Parse `packages` entries                   |
| Lerna           | `lerna.json` → `packages` field      | Expand globs                               |
| Cargo           | `Cargo.toml` → `[workspace.members]` | Expand to crate directories                |
| Go workspaces   | `go.work` → `use` directives         | List affected modules                      |
| Generic         | `members = ["path/a", "b"]`          | Explicit list in `.hookset.toml`           |

New `[workspace]` section in `.hookset.toml`:

```toml
[workspace]
type = "npm"          # auto-detect from files if omitted
# Or explicit:
members = ["packages/*"]

[[hooks]]
name = "test"
event = "pre-push"
workspace = true      # run once per affected member
command = "npm test"
```

### 1.2 Affected-Member Detection

`hookset exec --affected` computes which workspace members changed vs. the merge-base:

```
git diff --name-only HEAD...$(git merge-base HEAD main)
→ map files to workspace members via prefix matching
→ run hook command only for affected members
```

- Skip members with no changed files (fast-path exit).
- `--all` flag to override and run on every member.
- Detect rename/move boundaries via `git diff --detect-renames`.

### 1.3 `package_root` Directive

Allow any hook to declare a `package_root` relative to the repo root:

```toml
[[hooks]]
name = "backend-test"
event = "pre-push"
package_root = "services/api"
command = "go test ./..."
```

Alias to the existing `cwd` field but implies workspace-aware filtering: default `match` scoped to that subtree, and automatic re-rooting of pathspecs.

### 1.4 Recursive init — production

The existing `hookset init --recursive` walks subdirectories for `.hookset.toml` files and merges entries with path-adjusted `match` patterns. Productionise with:

- Validation: detect overlapping match patterns across subdirectory manifests and warn about ambiguity.
- Export a `hookset workspace list` command that prints the resolved member list.

> **Note**: `--watch` mode (auto re-running init on config file changes) is deferred to Phase 4 as an experimental flag. For now, `hookset check` is the recommended way to validate config changes before re-running init.

---

## Phase 2: Core Functionality & Performance

### 2.1 Parallel Hook Execution (Beta → Stable)

Current state: alpha `orchestrator` package behind `experimental = ["parallel"]`. Gate is opt-in and validated against interactive hooks.

**Path to stable:**

1. **Groups system** (like lefthook's `p`/`s`):

   ```toml
   [hookset.parallel]
   default = "parallel"   # or "serial"

   [[hooks]]
   name = "lint"
   group = "p:quick"      # parallel group "quick"
   ```

2. **Dependency graph** — declare ordering between hooks:

   ```toml
   [[hooks]]
   name = "build"
   depends_on = ["lint"]  # lint must finish before build starts
   ```

   Internally build a DAG, run independent hooks concurrently, fail fast on dependency failures, output interleaving with deterministic flush order.

3. **Timeout per hook** — kill and report hooks that exceed a deadline:

   ```toml
   [[hooks]]
   name = "e2e"
   timeout = "300s"
   ```

4. **Resource limits** — cap CPU/memory per parallel hook (Linux cgroups v2, Windows job objects).

5. **Remove alpha banner** when DAG and group systems are stable.

6. **Windows named-pipe I/O** — replace temp files for inter-process output buffering to avoid Windows filesystem locking issues during parallel writes.

### 2.2 Advanced Scripting & Conditionals

Expand `.hookset.toml` with git-aware run conditions:

```toml
[[hooks]]
name = "e2e"
event = "pre-push"
run_if  = { changed = ["e2e/**", "src/**/*.test.ts"] }
skip_if = { changed = ["docs/**", "*.md"] }
```

**`git diff` integration:**

```toml
[[hooks]]
name = "heavy-lint"
event = "pre-commit"
diff_base = "auto"    # → git merge-base HEAD upstream
```

**Conditional expressions:**

```toml
[[hooks]]
name = "deploy-check"
event = "pre-push"
if     = 'branch == "main" || branch =~ "^release/"'
unless = 'message =~ "\\[skip deploy\\]"'
```

**Built-in variables for conditionals:**

| Variable    | Source                                       |
| ----------- | -------------------------------------------- |
| `{branch}`  | `git rev-parse --abbrev-ref HEAD` (existing) |
| `{event}`   | Hook event name (existing)                   |
| `{message}` | Commit message (from file or `-m` flag)      |
| `{author}`  | `git config user.name` or commit author      |
| `{changed}` | Count of changed files matching patterns     |
| `{os}`      | `windows` / `linux` / `darwin`               |

- Use a lightweight expression evaluator (not a full embedded language).
- Expressions are pre-parsed and validated at `hookset check` time.
- `hookset check --verbose` prints the evaluated truth table for each hook against the current working tree state.

### 2.3 Git Worktrees Support

Current state: `IsLinkedWorktree()` returns true, `hookset exec` refuses to run.

**Implementation plan:**

1. **Read `.git` file** — linked worktrees have `.git` as a file, not a directory. Parse it to find the real `gitdir` and `worktrees/<name>/gitdir`.

2. **Per-worktree hook isolation** — use `git rev-parse --git-path config` to write hooks to the worktree-specific config instead of the shared one.

3. **Stash isolation** — stash operations must target the worktree's HEAD, not the main worktree's index. Verify `git stash --keep-index` behaviour across worktrees with a test matrix (main + 2 linked worktrees).

4. **Race prevention** — acquire a file lock on `.git/hookset.lock` before stash/pop to prevent concurrent worktree operations from corrupting the index.

5. **Worktree-aware `init`**:

   ```bash
   hookset init        # detects linked worktree, writes to worktree config
   hookset init --main # force write to main worktree config
   ```

6. **Test matrix**: `git worktree add` with shared tracking branch, divergent branches, detached HEAD, and bare repos (which cannot have worktrees).

> **Release checklist (v2.1.0)**: Remove "linked worktrees not supported" from the README Known Limitations section once this ships.

### 2.4 Performance Pass

- **Lazy stash** — skip stash entirely if `git diff --quiet` (no unstaged changes).
- **Batch `git add`** — use `git add --intent-to-add` + `git write-tree` for large file sets instead of per-file `git add`.
- **Parallel `git add`** during re-stage when in parallel mode.
- **Token expansion cache** — memoise `git diff --cached` results across hooks in the same event so each hook doesn't re-read the same staged file list.

---

## Phase 3: Team Collaboration & Workflow

### 3.1 Centralized Hook Management (`hookset remote`)

Pull curated hooks from a remote source, similar to `pre-commit`:

```bash
hookset remote add https://github.com/myorg/hookset-hooks
hookset remote update
hookset remote list
```

**Config in `.hookset.toml`:**

```toml
[remote]
url  = "https://github.com/myorg/hookset-hooks"
ref  = "v1.0.0"                   # optional: pin to a tag/branch
hooks = ["eslint", "prettier"]    # subset to install

[[hooks]]
name        = "custom-lint"
event       = "pre-commit"
remote_hook = "eslint"
command     = "npx eslint --config custom.js --fix"
```

**Remote hook distribution format:**

```
hookset-hooks/
├── index.toml
├── eslint.toml
├── prettier.toml
└── .hookset.toml
```

`index.toml`:

```toml
[remote]
version             = "1.0.0"
min_hookset_version = "2.0.0"

[[hooks]]
name        = "eslint"
description = "Run ESLint with auto-fix"
tags        = ["javascript", "typescript", "linter"]
```

- Hooks cached locally under `~/.config/hookset/remotes/<source>/`.
- Integrity verified via checksum or Sigstore signatures.
- `hookset check --remote-verify` validates every cached remote hook.
- Offline support: cached hooks work without network; `hookset remote update` is always explicit.

> CI commands must not rely on the daemon (Phase 5). All remote hook resolution happens at `hookset init` time and works fully offline after that.

### 3.2 Official CI/CD Integration

**`hookset ci` command:**

```bash
hookset ci                   # run hooks suitable for CI
hookset ci --event pre-push  # run a specific event's hooks
hookset ci --affected        # only run for changed files (PR context)
```

**CI environment detection:**

- Auto-detect via existing `isCI()` function.
- `HOOKSET_CI=1` overrides auto-detection.
- CI-specific config section:

```toml
[ci]
fail_fast = true
event     = "pre-push"

[[hooks]]
name = "e2e"
ci   = { skip = true }

[[hooks]]
name    = "lint"
ci      = { command = "npx eslint --max-warnings=0" }
```

**Built-in CI templates:**

```bash
hookset init --template ci-github   # generates .github/workflows/hookset.yml
hookset init --template ci-gitlab   # generates .gitlab-ci.yml hookset job
```

Generated GitHub Actions workflow:

```yaml
name: hookset CI
on: [push, pull_request]
jobs:
  hookset:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: curl -fsSL https://raw.githubusercontent.com/bulga138/hookset/master/scripts/install.sh | sh
      - run: hookset ci
```

**Annotations support**: Parse hook output for error lines and emit GitHub Workflow Command annotations (`::error file=...,line=...::...`).

### 3.3 Granular Hooks & Skipping Rules

**Commit-message skip directives:**

```
git commit -m "urgent fix [skip ci]"
git commit -m "docs update [hookset skip eslint,prettier]"
```

Parse `[skip ci]`, `[hookset skip]`, and `[hookset skip <names>]` from the commit message in `prepare-commit-msg` or `commit-msg` hooks.

**`.hookset-local.toml` — user-level overrides (gitignored, per-device):**

```toml
[[hooks]]
name  = "heavy-lint"
local = { skip = true }

[[hooks]]
name  = "test"
local = { command = "npm test -- --watch" }
```

- `--no-local` flag on `hookset init` to ignore local overrides.
- Validation: warn when a local override removes a required hook.
- `hookset list --local` shows the effective config including local overrides.

**Skip matrix:**

| Mechanism                  | Scope       | Priority    |
| -------------------------- | ----------- | ----------- |
| `HOOKSET=0`                | Global      | 1 (highest) |
| `HOOKSET_SKIP_IN_CI=1`     | Global/CI   | 2           |
| `[hookset skip <name>]`    | Per-commit  | 3           |
| `.hookset-local.toml skip` | Per-device  | 4           |
| `HOOKSET_SKIP` env var     | Per-session | 5           |
| `hookset disable <name>`   | Per-repo    | 6           |
| `HOOKSET_ONLY` env var     | Per-session | 7 (inverse) |

---

## Phase 4: Ecosystem & Community

### 4.1 Plugin Ecosystem — "Hookpacks"

A Hookpack is a versioned, distributable bundle of hook definitions. This is also the intended home for build-system-specific integrations (Bazel, Pants, etc.) that are out of scope for the core.

**Spec:**

```toml
# hookpack.toml
[hookpack]
name                = "rust-clippy"
version             = "1.0.0"
min_hookset_version = "2.3.0"
description         = "Rust toolchain hooks (fmt + clippy + test)"

[[hooks]]
name    = "rust-fmt"
event   = "pre-commit"
match   = ["*.rs"]
command = "cargo fmt --all -- --check"

[[hooks]]
name    = "rust-clippy"
event   = "pre-commit"
match   = ["*.rs"]
command = "cargo clippy --all-targets -- -D warnings"

[[hooks]]
name    = "rust-test"
event   = "pre-push"
command = "cargo test"
```

**Hookpack registry (future):** `hookset pack search`, `hookset pack publish` — a community registry akin to npm or pre-commit hooks.

**Pack file structure:**

```
rust-clippy@1.0.0.hpack    # gzipped tar: hookpack.toml + optional scripts/
```

**Commands:**

| Command                       | Description                              |
| ----------------------------- | ---------------------------------------- |
| `hookset pack install <spec>` | Install a Hookpack by name or path       |
| `hookset pack list`           | List installed Hookpacks                 |
| `hookset pack update [name]`  | Update Hookpack(s) to latest             |
| `hookset pack remove <name>`  | Uninstall a Hookpack                     |
| `hookset pack info <name>`    | Show Hookpack metadata and hooks         |
| `hookset pack init`           | Scaffold a new Hookpack directory        |
| `hookset pack verify <path>`  | Validate Hookpack structure + signatures |

### 4.2 Config Watch Mode (Experimental)

Deferred from Phase 1. Adds a `--watch` flag to `hookset init` that monitors `.hookset.toml` for changes and re-runs init automatically using inotify/kqueue/FSEvents.

Ships behind an explicit `experimental = ["watch"]` flag with documented caveats (fd leak risk on config file deletion, race conditions during concurrent edits). Not recommended for CI or scripted environments.

### 4.3 Official VS Code Extension

**Prerequisites before release:**

- The v2.x config schema has been stable across at least two minor releases (i.e., ships no earlier than v2.4.0 having followed v2.2.0 and v2.3.0 without breaking changes).
- `hookset schema` command exists and outputs the current config schema as JSON Schema.

**`hookset-vscode`:**

| Feature                  | Description                                                             |
| ------------------------ | ----------------------------------------------------------------------- |
| **Hook status**          | Decoration in Source Control sidebar showing which hooks are active     |
| **Config editor**        | GUI for `.hookset.toml` with autocomplete, validation, and schema forms |
| **Hook test**            | "Run this hook" button — executes against current staged files          |
| **On-save hooks**        | Optional `pre-save` event: runs a hook when a file is saved in editor   |
| **Regex tester**         | Interactive panel to test `match` patterns against workspace files      |
| **Inline annotations**   | Hook errors/warnings surfaced as editor diagnostics                     |
| **Output channel**       | Dedicated "hookset" panel in Output view                                |
| **Git extension integ.** | Hooks into VS Code's git extension for staged file awareness            |

The extension declares a `min_hookset_version` and refuses to activate against incompatible binaries.

### 4.4 Public Roadmap & Community Hub

- **`ROADMAP.md`** (this document) — public, versioned, linked from README.
- **GitHub Discussions** — category per phase for Q&A, feature requests, and Hookpack announcements.
- **GitHub Issue templates** — bug report, feature request, Hookpack submission.
- **Monthly releases** — tagged `v2.x.x` following semver.
- **Security policy** — `SECURITY.md` with PGP key and disclosure process.

---

## Phase 5: Future Exploration

| Feature                       | Rationale                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| ----------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **`hookset daemon`**          | Persistent agent for file-watching + hook result caching. _Why Phase 5?_ The daemon only provides meaningful value after the parallel executor (Phase 2), watch mode (Phase 4), and remote hook fetching (Phase 3) are stable. Without those, the daemon solves problems that don't yet exist. It implies a persistent socket, new IPC protocol, authentication model, and lifecycle supervision — a fundamental architectural shift. Remains a research item, not a committed feature. |
| **Bare repo support**         | Server-side hooks in mirror repos                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| **Partial clone awareness**   | Skip hooks for files not present locally                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| **Nix flake / devbox integ.** | Hook commands with hermetic toolchains                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| **Pre-commit mirror**         | Auto-convert pre-commit hooks repo → Hookpack                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| **Differential testing**      | Run new hooks on only changed lines (not files)                                                                                                                                                                                                                                                                                                                                                                                                                                         |

---

## Version Strategy

```
v2.0.0   Phase 1  — Monorepo & Workspace Features
v2.1.0   Phase 2  — Parallel (stable) + Conditionals + Worktrees
v2.2.0   Phase 3  — Remote Hooks + CI + Granular Skipping
v2.3.0   Phase 4a — Hookpacks
v2.4.0   Phase 4b — VS Code Extension (schema stable prerequisite met)
v3.0.0            — Daemon mode + Nix integration + API stability guarantees
```

Milestones are approximate and will be adjusted based on community feedback.
