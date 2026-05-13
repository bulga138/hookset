# ROADMAP.md — `hookset`

## Project `hookset` — The Configuration‑Native Git Hook Manager

`hookset` is a globally‑installed, language‑agnostic CLI tool that lets you define, run, and share Git hooks
entirely through Git’s native `[hook]` configuration sections (introduced in Git 2.54). It implements its own
staging‑file filtering and stash‑unmodified‑pop engine, so repositories never depend on `lint‑staged`, husky,
lefthook, or any package‑manager‑specific hook runner.

**`hookset` requires Git 2.54 or later.** No separate fallback scripts are ever committed — hooks are pure configuration.

---

## Core Principles

1. **No per‑language tooling pollution in the repository.**  
   No `package.json` entries, no `prepare` scripts, no `.huskyrc`, no `lefthook.yml`.  
   _The only intentional committed file is a declarative, language‑agnostic `.hookset.toml`._

2. **Workstation‑level install, not a project dependency.**  
   `hookset` is installed once per machine (brew, curl, winget) — just like Git.  
   It is _not_ a `devDependency`. A contributor is expected to have it before cloning.

3. **One binary, one staging engine.**  
   `hookset exec` replaces `lint‑staged` entirely. No wrapper scripts, no shell‑out to other tools
   for the core stash‑filter‑restage dance.

4. **Native configuration only.**  
   Hooks are defined directly in Git config using `[hook "name"]` sections.  
   `hookset init` reads `.hookset.toml` and writes the equivalent local config; `hookset add` writes
   directly to the chosen scope. Git 2.54 or later is a hard requirement.

5. **Shared config via include.**  
   A project’s `.hookset.toml` can include a shared file (e.g., from a dotfiles repo) to avoid
   repeating hook definitions across repositories.

6. **No committed hook scripts.**  
   The hook command stored in Git config is a self‑contained one‑liner that checks for the
   `hookset` binary itself, so missing‑binary errors are human‑readable even without the tool.

---

## Detailed Phases

### Phase 0 — Research, Design, and Spike (3 weeks)

**Goal:** Validate the staging engine approach, pin down the bootstrapping story, and understand
every tool we intend to replace or migrate.

#### Tasks

1. **Git 2.54 native hooks deep‑dive**
   - Experiment with `[hook "name"] event = pre-commit; command = ...`
   - Understand config scoping, multi‑value keys, ordering relative to `.git/hooks/` scripts.
   - Verify how `git hook list` reports hooks from multiple scopes.

2. **Analyse husky, lefthook, lint‑staged**
   - How does husky v9 install? (`.husky/` directory, `core.hooksPath`)
   - How does lefthook install? (wrapper script, `lefthook.yml` config)
   - How does lint‑staged implement its stash → run → restage loop?
   - Document migration mapping: from each tool’s config to equivalent `hookset add` commands.

3. **Bootstrapping design**
   - Define the format of `.hookset.toml` (the single committed file). Support an `include` key
     pointing to a local path for shared standard hooks (e.g., from a dotfiles repo).

     ```toml
     include = "~/dotfiles/hooks/standard.toml"

     [[hooks]]
     name = "eslint"
     event = "pre-commit"
     match = ["*.ts", "*.js"]
     command = "npx eslint --cache --fix"
     ```

   - **Canonical flow:** maintainers edit `.hookset.toml` (or use `hookset add --manifest` to append to it).  
     Contributors run `hookset init`, which reads the TOML and writes local `[hook]` config.  
     _There is no direct path from `hookset add` to local git config that bypasses the TOML_ — when a maintainer
     wants to add a hook both locally and to the manifest, they run `hookset init` after updating the manifest,
     never the reverse. The `--manifest` flag exists only to update the committed file; `hookset add` without
     `--manifest` is for personal, uncommitted hooks.
   - Design the self‑checking wrapper command: `hookset init` writes a `command` that starts with a presence
     check, e.g.:
     ```
     sh -c 'command -v hookset >/dev/null 2>&1 || { echo "hookset is not installed. Install with: brew install hookset   (or visit https://hookset.dev)" >&2; exit 1; }; exec hookset exec --match ... -- ...'
     ```
     This ensures the commit fails with a human‑readable message if `hookset` is missing.

4. **Staging engine spike**
   - Write a separate, throw‑away Go program that takes a command and a list of file patterns,
     performs the `git stash --keep-index`, runs the command, re‑stages modified files, and pops
     the stash.
   - Read the `nano-staged` source as a compact reference for the core loop.
   - Run it against a curated set of test repositories covering:
     - Only unstaged changes unrelated to the linted files.
     - Partially staged files (some hunks staged, some not).
     - A linter that deletes a file.
     - A linter that converts CRLF ↔ LF.
     - Staged binary files.
     - Submodules in the tree (skipped).
     - Two active worktrees for the same repo (commit in one while the other is dirty).
   - Document every failure mode and mitigation strategy.

#### Acceptance Criteria

- [ ] Internal design note on Git 2.54 hook semantics and edge cases.
- [ ] Migration mapping tables for husky, lefthook, lint-staged.
- [ ] Finalised `.hookset.toml` specification (fields, types, allowed events, `include` semantics).
- [ ] Spike results: list of staging engine risks and a clear “go / no‑go” decision.

---

### Phase 1 — Infra & Skeleton (2 weeks)

**Goal:** Set up the repository, build system, and a minimal runnable CLI.

#### Tasks

1. **Go module & directory layout**

```
hookset/
├── cmd/hookset/        # main
├── internal/
│   ├── gitconfig/      # read/write git config
│   ├── exec/           # staging engine
│   ├── migrate/        # husky/lefthook/lint-staged parsers
│   └── toml/           # .hookset.toml handling
├── scripts/
│   ├── install.sh      # curl | sh installer
│   └── install.ps1     # irm | iex installer
└── .goreleaser.yml
```

2. **Goreleaser & CI**
   - Cross‑compile for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64.
   - GitHub Actions release workflow that attaches binaries to tags.

3. **CLI skeleton**
   - `hookset version` prints build info.
   - Help text for planned subcommands: `add`, `remove`, `list`, `disable`, `enable`, `exec`, `init`, `migrate`.
   - Verbose flag (`--verbose`) wired into all commands.
   - Check for Git ≥ 2.54 at startup; refuse to run otherwise with a clear message.

#### Acceptance Criteria

- [ ] `go build ./cmd/hookset` produces a binary.
- [ ] CI passes: lint, build, attach to release.
- [ ] `hookset --help` shows a clean command tree.
- [ ] Running `hookset` on Git < 2.54 prints a specific error directing the user to upgrade.

---

### Phase 2 — Git Config & `.hookset.toml` Operations (2 weeks)

**Goal:** A reliable library that reads/writes `hook.*` keys and the committed manifest.

#### Tasks

1. **`internal/gitconfig` package**
   - `GetHooks(event string, scope)` — list hooks with name, command, enabled status, match patterns.
   - `AddHook(name, event, command, matches []string, scope)` — writes via `git config`.
   - `RemoveHook(name, scope)`.
   - `EnableHook(name, scope, enabled bool)`.
   - Multi‑value match support: each `--match` becomes a separate `hook.<name>.match` line.
   - **Idempotent addition:** when adding a hook that may already exist (e.g., on re‑run of `hookset init`), `AddHook` will first delete all existing `hook.<name>.*` entries in the target scope, then write the new values. This prevents duplicate multi‑value lines from accumulating. The same remove‑then‑add strategy is used by `hookset init` for each hook it processes.

2. **Scope merging test suite**
   - Verify that when hooks with the same event exist in both global and local scope,
     they are returned in the order Git would execute them.
   - Ensure that disabling a hook at the local level does not affect the global entry.
   - Test that `hookset list` mirrors `git config --get-regexp` ordering.

3. **`.hookset.toml` handling**
   - `ReadManifest(path)` — validates and returns hooks; resolves local `include` paths and
     merges included hooks (project overrides by name if duplicate).
     **Missing included files:** emit a **warning** by default, skip the missing hooks.
     A `--strict` flag on `hookset init` turns this into a fatal error.
   - `AddToManifest(path, hook)` — idempotent update (used by `hookset add --manifest`).
   - Mapping between TOML fields and git‑config multi‑value keys.

#### Acceptance Criteria

- [ ] Unit tests for all get/add/remove/enable operations on a temporary repo.
- [ ] A hook with three `match` patterns results in three `hook.name.match` lines.
- [ ] Scope ordering tests pass and are documented as the canonical behaviour.
- [ ] An existing `.hookset.toml` can be parsed and round‑tripped without data loss.
- [ ] Including a shared TOML file merges hooks correctly, with project overrides respected.
- [ ] Missing included file produces a warning; with `--strict` it produces a non‑zero exit.

---

### Phase 3 — CLI: Hook Management (2 weeks)

**Goal:** `hookset add`, `remove`, `list`, `disable`, `enable`, and `hookset init`.

#### Canonical flow (enforced)

- **Maintainer** edits `.hookset.toml` (or uses `hookset add --manifest` to append to it), then runs `hookset init` to apply the changes locally.
- **Contributor** clones and runs `hookset init`.  
  _There is no `hookset add` that writes local git config directly for a team hook; `hookset add` without `--manifest` is reserved for personal, uncommitted hooks._

#### Tasks

1. **`hookset add`**
   - Flags: `--match` (repeatable), `--on` (default `pre-commit`). The command is specified after a `--` separator (idiomatic for subprocess invocations), e.g., `hookset add eslint --match "*.ts" --on pre-commit -- npx eslint --cache --fix`.
   - With `--manifest`: appends the hook to `.hookset.toml` **only**. Does not write git config.
   - Without `--manifest`: writes a local `[hook "name"]` section directly to `.git/config` (for personal, per‑repo usage that is not meant to be shared).
   - With `--global`: writes to `~/.gitconfig` (always direct config; never touches a manifest).

2. **`hookset remove <name>`** — removes all related `hook.<name>.*` entries from the chosen scope (and optionally from the manifest with `--manifest`).

3. **`hookset disable/enable <name>`** — toggles `hook.<name>.enabled`.

4. **`hookset list`** — uses `git hook list` and presents a clean table.

5. **`hookset init`**
   - Reads `.hookset.toml` (and any included files) from the repo root. Missing includes warn by default, `--strict` makes them an error.
   - For each hook entry, writes the self‑checking wrapper command into the local `[hook]` config:
     ```
     sh -c 'command -v hookset >/dev/null 2>&1 || { echo "hookset not found. Install: brew install hookset" >&2; exit 1; }; exec hookset exec --match ... -- ...'
     ```
     (actual install instructions tailored to OS at generation time).
   - Idempotent: can be run multiple times without duplicating entries.

#### Acceptance Criteria

- [ ] Full lifecycle: `add` (personal) → `list` → `disable` → `enable` → `remove` works.
- [ ] `add --manifest` updates `.hookset.toml` but does not touch git config.
- [ ] `hookset init` on a repo with a valid `.hookset.toml` writes the self‑checking wrapper commands;
      a subsequent commit attempt without `hookset` installed prints the install message and fails cleanly.
- [ ] `hookset init` on a repo without `.hookset.toml` prints a helpful message and exits 0.
- [ ] `hookset init` resolves `include` paths, merges hooks, and respects the `--strict` flag.

### Phase 3.1 – Windows Wrapper Format (incorporated into Phase 3)

**Additional Task:**

- **Design the self‑checking wrapper for Windows.**  
  The hook command must be a one‑liner that works on any Windows shell (cmd, PowerShell, Git Bash).
  - Use a small, committed `.hookset-wrapper.cmd` script that does the presence check and then calls `hookset exec`? (complex, violates “no committed files” principle)
  - Alternative: use a Git‑portable `sh` invocation if Git for Windows provides `sh` (it does). But `command -v` may be inconsistent.
  - **Chosen approach:** `hookset init` writes a **per‑OS command** into the config, i.e., the self‑check logic is embedded using `sh` on Unix and a `cmd /c` call on Windows. The hook command is generated **once** at `init` time and tailored to the machine that runs `init`. Document that cross‑OS cloning requires re‑running `hookset init` to regenerate the platform‑appropriate wrapper.
  - **Phase 6** must also ensure that the `install-action` and installers produce a wrapper that works on the target OS.
- Test on Windows with PowerShell, cmd, and Git Bash that the missing‑binary message appears correctly.

---

### Phase 4 — Staging Engine: `hookset exec` (6 weeks)

**Goal:** The reusable component that Git calls directly via the hook command. It does
the stash‑filter‑run‑restage‑pop dance. This is the hardest piece.

#### Tasks

1. **Core implementation (`internal/exec`)**
   - Parse arguments: `hookset exec --match "*.ts" --match "*.js" -- <command>`.
   - Determine staged files: `git diff --cached --name-only --diff-filter=ACMR`.
   - Apply path matching (delegate to `git ls-files --cached -- <patterns>` for exact Git‑native matching).
   - If no files match, exit 0 immediately.
   - Stash unmodified changes: `git stash push --include-untracked --keep-index -m "hookset pre-commit"`.
   - If stash fails (conflicts, etc.), abort with a clear error and no index change.
   - Run the command, passing matching file paths as arguments (fallback: chunk invocations or stdin if argument list would exceed the Windows safe limit of ~8000 characters).
   - Capture exit code.
   - Re‑stage files modified by the command: `git add -- <files>`.
   - If a file was deleted by the formatter, `git rm --cached <file>`.
   - Pop the stash: `git stash pop`.
   - Always propagate the command’s exit code.

2. **Edge‑case handling (documented where incomplete)**
   - **Partial staging:** If a file has some hunks staged and some not, the tool will modify the file,
     and re‑staging the entire file will **silently include the previously unstaged hunks in the commit**.
     This is identical to lint‑staged’s default behaviour, and is called out as a **known limitation** in v1.
   - **Submodules:** skip files inside submodules; do not attempt to stash/unstash across module boundaries.
   - **Large binary files:** issue a warning if a file >10 MB is about to be stashed; provide an `--allow-large` flag.
   - **File mode changes:** preserve mode after re‑adding.
   - **CRLF conversions:** handle the case where `core.autocrlf` or `.gitattributes` cause the working tree to differ from the index after `git add`.
   - **Worktrees:** detect when the repo belongs to a worktree set; if another worktree might be affected, abort with a clear message that worktrees are not supported in v1.

3. **Verbose/debug mode**
   - `--verbose` prints: staged files found, files matched, files excluded, stash SHA, command invocation, files added after command, final stash pop status.

4. **Integration test suite**
   - Set up a sandbox repo for each edge‑case from the Phase 0 spike.
   - Parameterised tests that run against Git 2.54 and 2.55+.
   - Explicit test: “existing unstaged changes remain completely untouched after a successful lint run.”
   - Explicit test: “a linter that fails leaves the working tree exactly as before (stash properly popped).”

#### Acceptance Criteria

- [ ] All spike scenario tests pass.
- [ ] A commit with only CSS files and a JS linter hook triggers no lint run and exits 0.
- [ ] A successful lint run that modifies a file results in the modified file being re‑staged and the working tree clean for the commit.
- [ ] A failed lint run pops the stash and restores the index and working tree exactly as before `hookset exec` was called.
- [ ] Verbose mode output is clear enough to debug a real‑world problem.
- [ ] When argument length would exceed the Windows limit, files are chunked or passed via stdin without error.

---

### Phase 5 — Migration Tools (2 weeks)

**Goal:** `hookset migrate` reads a project’s existing hook setup and emits working `hookset add --manifest`
commands (or a `.hookset.toml`).

#### Tasks

1. **`--from lint-staged`**
   - Find config in `package.json`, `.lintstagedrc`, etc.
   - For each glob→command pair, output equivalent `hookset add --manifest` with `--match`.
   - Warn if the config uses function syntax (cannot migrate automatically).

2. **`--from husky`**
   - Parse `.husky/<event>` shell scripts.
   - For file‑filtering tasks (e.g., linters on staged files), wrap the command with `hookset exec --match ...`.
   - For non‑filtering tasks (e.g., `tsc --noEmit`, test suite, `pre-push` checks), output a plain `hookset add` entry with no `--match`; the command runs as‑is without the staging engine.
   - If the script is simply `npx lint-staged`, fall back to the lint-staged migrator

3. **`--from lefthook`**
   - Parse `lefthook.yml`.
   - Map `commands` with `glob`, `run`, etc. to `hookset add --manifest`.
   - Note: parallelism settings are ignored (can be added post‑MVP).

4. **`--dry-run`** — prints what would be added without changing anything.
5. **`--yes`** — actually writes the manifest (and runs `hookset init` to install).

#### Acceptance Criteria

- [ ] Migrating a standard `lint-staged` setup produces a working `.hookset.toml` + local hooks
      that match the original linter coverage exactly.
- [ ] Husky migration that had `lint-staged` at the centre results in direct linter invocations
      (no leftover `lint-staged` dependency).
- [ ] Existing husky/lefthook files are not modified; the user can manually remove them afterward.

---

### Phase 6 — Distribution: Installers & CI Action (2 weeks)

**Goal:** Make `hookset` trivially installable on any developer machine and in CI pipelines.

#### Tasks

1. **`curl | sh` / `irm | iex` installer**
   - Detect OS/arch, download the latest release binary from GitHub, place in `~/.local/bin` (Unix) or `ProgramFiles` (Windows), add to user’s `PATH`.
   - Respect `HOOKSET_BINARY_PATH` environment variable: if set, copy from that local path instead of downloading (for air‑gapped environments).

2. **Homebrew formula**
   - `hookset.rb` for a custom tap (or later homebrew-core).

3. **winget and scoop manifests** for Windows.

4. **`hookset/install-action`**
   - A GitHub composite action that downloads the binary and adds it to `$PATH`.
   - Inputs: `version` (default: latest).
   - Also respects `HOOKSET_BINARY_PATH` for offline/air‑gapped CI runners.
   - Used in CI as:
     ```yaml
     - uses: hookset/install-action@v1
       with:
         version: '1.0.0'
     - uses: actions/checkout@v4
     - run: hookset init
     ```

5. **Optional NPM thin wrapper (`@hookset/cli`)**
   - Post‑MVP convenience: a package that downloads the binary on `postinstall` and exposes it as `npx hookset`.
   - Documented as secondary, not the recommended CI path, and not part of the 1.0 critical path.

#### Acceptance Criteria

- [ ] One‑command install works on fresh macOS, Ubuntu, Windows (PowerShell).
- [ ] Setting `HOOKSET_BINARY_PATH` skips the download and uses the local binary.
- [ ] `hookset/install-action` successfully runs `hookset version` in a GitHub Actions job.
- [ ] Homebrew install command places `hookset` in the user’s PATH.

---

### Phase 7 — Testing & CI Matrix (2 weeks)

**Goal:** Catch regressions across platforms and Git versions.

#### Tasks

1. **CI matrix**
   - OS: `ubuntu-latest`, `macos-latest`, `windows-latest`.
   - Git versions: `2.54`, `latest`.
   - Each job runs unit + integration tests.

2. **End‑to‑end tests**
   - Dockerised scenarios: “clone a repo with `.hookset.toml`, run `hookset init`, commit a file that fails a linter — hook must block.”

3. **Performance baseline**
   - `hookset exec` with a no‑op command on a repo of 5,000 staged files should finish in <1s.

4. **Backward compatibility**
   - Hooks added with an older version of `hookset` remain readable and functional after an upgrade.

#### Acceptance Criteria

- [ ] CI matrix green for all OS/Git combinations.
- [ ] End‑to‑end test shows the complete workflow from clone to blocked commit.
- [ ] No regression on common lint‑staged/lefthook user patterns.

---

### Phase 8 — Documentation & Community (2 weeks)

**Goal:** Excellent first‑run experience.

#### Tasks

1. **README** — install, quickstart, comparison table with husky/lefthook. Clearly state Git 2.54+ requirement.
2. **`hookset.dev` static site** — interactive demo, migration guide.
3. **Manpage & shell completions** (bash, zsh, fish, pwsh).
4. **Troubleshooting guide** — stash failures, file‑mode changes, Windows permissions, worktree limitations, partial staging caveats.
5. **Contributing guide** for new migration parsers or enhancements.

#### Acceptance Criteria

- [ ] A new user can go from nothing to running hooks in <5 minutes using the README alone.
- [ ] The website’s “Quick setup” snippet is copy‑paste runnable.

---

### Phase 9 — Post‑MVP Enhancements

After 1.0, high‑priority improvements (in order of community demand):

1. **Parallel execution** — `--jobs N` in `hookset exec` to run multiple tools simultaneously.
   _Note:_ Git 2.55 may introduce native `jobs = N` on hook config; evaluate before building a custom scheduler.
2. **Hook templates** — `hookset init --template` for a quick starter config.
3. **Extended migration** — commit‑msg, pre‑push hooks, etc.
4. **Plugin system for migration sources** — allow community‑supplied parsers.
5. **`hookset check`** — CI linting of hook config correctness without running hooks.
6. **Graphical installer** — a small Electron/Tauri app for absolute beginners (very low priority).

---

## Timeline (full‑time solo developer)

| Phase | Duration | Cumulative |
| ----- | -------- | ---------- |
| 0     | 3 weeks  | 3 w        |
| 1     | 2 weeks  | 5 w        |
| 2     | 2 weeks  | 7 w        |
| 3     | 2 weeks  | 9 w        |
| 4     | 6 weeks  | 15 w       |
| 5     | 2 weeks  | 17 w       |
| 6     | 2 weeks  | 19 w       |
| 7     | 2 weeks  | 21 w       |
| 8     | 2 weeks  | 23 w       |

**~23 weeks to a polished 1.0 release.**

---

## Risk Register

| Risk                                                                                                 | Likelihood | Impact   | Mitigation                                                                                                                                                                                                        |
| ---------------------------------------------------------------------------------------------------- | ---------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Staging engine edge cases cause data loss or corrupted index                                         | Medium     | Critical | Phase 0 spike, 6‑week implementation, public “known limitations” v1 doc. Fallback to embedded, stripped‑down lint‑staged only as last resort.                                                                     |
| Git 2.54 not available on a user’s system                                                            | Medium     | High     | Hard requirement documented at install and startup. Installers prompt upgrade; CI action installs correct Git version if needed.                                                                                  |
| Config scope merging produces incorrect hook ordering                                                | Medium     | High     | Dedicated test suite in Phase 2. Behaviour documented; any deviation from Git’s own ordering is a P1 bug.                                                                                                         |
| Windows file‑locking / antivirus blocks stash pop                                                    | Medium     | Medium   | CI tests on real Windows environment with Defender. Provide `--no-stash` escape hatch for affected users.                                                                                                         |
| **`hookset` not installed but hooks are configured** — commit fails with generic “command not found” | **High**   | **High** | Hook command generated by `hookset init` includes a self‑check that prints explicit install instructions before failing.                                                                                          |
| Bootstrapping confusion (new contributors forget `hookset init` after clone)                         | **Medium** | Medium   | `hookset init` is idempotent; the self‑checking wrapper will tell them `hookset` is missing if they skipped the install entirely. Clone instructions prominently documented; `hookset/install-action` handles CI. |
