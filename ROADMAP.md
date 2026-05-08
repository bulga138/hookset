# ROADMAP.md

## Project `hookset` — The Configuration-Native Git Hook Manager

`hookset` is a globally-installed CLI tool that lets you define, manage, and run Git hooks entirely through `git config`, leveraging the native config‑based hook mechanism introduced in Git 2.54. Every lint‑staging, formatting, or analysis step runs through `hookset exec`, a single binary that handles staged‑file filtering, stash/pop, and re‑staging — no extra tools required, no per‑repo scripts, no `package.json` pollution.

---

## Goals

- **Zero repo pollution** – no hook scripts, no `prepare` scripts, no `core.hooksPath` tricks.
- **Git 2.54 first** – use native `[hook]` config sections; provide graceful fallback for older Git.
- **Self‑contained staging engine** – `hookset exec` replaces `lint‑staged` entirely (stash, filter, run, re‑stage).
- **One binary, many installs** – pre‑compiled static Go binary, distributed via `curl|sh`, Homebrew, winget, and optional npm/pip thin wrappers.
- **Seamless migration** – `hookset migrate` can read existing husky, lefthook, or lint‑staged setups and convert them.

---

## Phase 0: Research & Design Finalization

**Goal:** Understand how existing tools handle hooks, staging, and configuration, and pin down the final architecture for `hookset` before writing code.

### Tasks

1. **Deep‑dive into Git 2.54 config‑based hooks**
   - Study `hook.<name>.event`, `hook.<name>.command`, `hook.<name>.enabled`.
   - Understand ordering: hooks run in the order they appear in config, with `$GIT_DIR/hooks` scripts running last.
   - Check multi‑value config support (e.g., repeated `--match`).
   - Determine how `git hook list` works and what `hookset list` should wrap.

2. **Analyse husky v9**
   - How does husky install? (writes `.husky/*` scripts, sets `core.hooksPath = .husky`).
   - Hook content is just user‑written shell. Most users combine with lint‑staged.
   - Extract migration strategy: read `.husky/pre-commit` to retrieve commands.

3. **Analyse lefthook**
   - Configuration in `lefthook.yml`, multiple commands per hook, globs, parallel execution.
   - Installs a wrapper script into `.git/hooks/` that invokes `lefthook run`.
   - Migration: parse YAML, map to `hookset add` options.

4. **Analyse lint‑staged**
   - Config in `package.json` (or `.lintstagedrc`), glob → command mapping.
   - Implementation: `git stash` specific index, run linters on matching files, re‑add, pop.
   - Edge cases: partial staging (it re‑stages whole files), deleted files, submodules, `--allow-empty`.
   - Migration: read config and emit `hookset add` for each tool.

5. **Decide compatibility strategy**
   - Git ≥ 2.54: write native `[hook]` sections, `hookset exec` invoked directly via `command`.
   - Git < 2.54: fallback to a `.git/hooks/<event>` shell script that calls `hookset run <event>`.
   - Auto‑detect on `hookset add` and choose the right method (or error with a clear message).

6. **Finalise config schema**
   - Hook entry fields: `event`, `command` (with `hookset exec`), optionally `match`, `enabled`.
   - Support per‑repo (`--local`) and per‑user (`--global`) scopes.
   - Allow multiple `--match` values stored as multi‑value config keys.

### Acceptance Criteria

- [ ] A written summary of native Git 2.54 hook behaviour, limitations, and multi‑value support.
- [ ] Migration mapping tables for husky, lefthook, lint‑staged.
- [ ] Decision document on fallback mechanism (e.g., fallback script template, how `hookset add` handles it).
- [ ] Final JSON/YAML schema (conceptual) for a hook entry used internally.

---

## Phase 1: Project Infrastructure

**Goal:** Set up the monorepo, build tooling, and minimal runnable skeleton.

### Tasks

1. **Initialize Go module and directory structure**
   - `cmd/hookset/main.go`, `cmd/` sub‑commands skeleton (using `cobra` or `urfave/cli`).
   - `internal/gitconfig/`, `internal/exec/`, `internal/migrate/` packages.
2. **Configure goreleaser** for cross‑compilation (linux/amd64, linux/arm64, macos/amd64, macos/arm64, windows/amd64).
3. **Set up CI** (GitHub Actions) with linting, test matrix, and release workflow.
4. **Versioning** – embed version via `-ldflags`, show with `hookset version`.
5. **Basic logging and error handling** – structured output for normal runs, `--verbose` for debugging.

### Acceptance Criteria

- [ ] `go build ./cmd/hookset` produces a binary that prints version and help.
- [ ] Cross‑compilation CI passes and attaches binaries to a GitHub Release.
- [ ] Repository structure matches the agreed layout.

---

## Phase 2: Core Git Config Operations

**Goal:** A reliable library that reads and writes `hook.*` keys using `git config`, even when the repo is missing, bare, or in a broken state.

### Tasks

1. **Implement `internal/gitconfig` package**
   - `GetHooks(event string, scope)` → list of hook entries with name, command, enabled.
   - `AddHook(name, event, command, extraKeyValues map[string]string, scope)`.
   - `RemoveHook(name, scope)`.
   - `EnableHook(name, enabled bool, scope)`.
   - All operations shell out to `git config` (safe, respects includes, handles quoting).
2. **Handle multi‑value keys**
   - `--match` values stored as multiple `hook.<name>.match` lines.
   - `hookset add --match ...` must use `git config --add` for each.
3. **Scope validation** – refuse to write `--global` if `--local` already exists? (Or allow and Git merging rules apply; document.)
4. **Fallback hook file management**
   - For pre‑2.54, `WriteFallbackHook(event)` creates `.git/hooks/<event>` with a shebang that calls `hookset run <event>`.
   - Idempotent, must not erase user’s custom scripts unless the whole file is managed by us (use a sentinel comment).
   - `RemoveFallbackHook(event)` if no more hooks exist.

### Acceptance Criteria

- [ ] Unit tests for all operations using a temporary Git repository.
- [ ] Adding a hook with three `--match` values results in three `hook.name.match` lines.
- [ ] Listing returns all hooks, including those from global/local scope, with correct ordering.
- [ ] Fallback script is created with `# hookset managed` comment; re‑running does not duplicate.

---

## Phase 3: Hook Management CLI

**Goal:** Provide the `add`, `remove`, `list`, `disable`, `enable` commands.

### Tasks

1. **`hookset add`**
   - Parses `--match` (repeatable), `--on` (event, default `pre-commit`), `--cmd` (the raw command without `hookset exec` wrapper).
   - If Git ≥ 2.54, writes `hook.<name>.event` and `hook.<name>.command = "hookset exec --match ... -- <cmd>"` directly.
   - If Git < 2.54, falls back to writing the `command` into config (for `hookset run` to read) and installs the fallback hook script.
   - Fluent examples: `hookset add eslint --match "*.ts" --on pre-commit -- npx eslint --cache --fix`.
2. **`hookset list`**
   - Wraps `git hook list` when available, else custom listing from config.
   - Shows scope, name, command, enabled status.
3. **`hookset remove <name>`** – removes all config entries for that hook.
4. **`hookset disable <name>` / `enable`** – sets `hook.<name>.enabled`.
5. **`--global` / `--local` flags** (default: `--local`).
6. **Validation**
   - Hook names must be valid config section keys.
   - Event names must be valid Git hooks.
   - Warn if `hookset exec` is not installed when adding.

### Acceptance Criteria

- [ ] Full cycle of add → list → disable → enable → remove works without errors.
- [ ] Adding a hook with `--global` places it in `~/.gitconfig`; local overrides are shown in list.
- [ ] On Git < 2.54, a `.git/hooks/pre-commit` script is created and calls `hookset run pre-commit`.
- [ ] Invalid event name is rejected with a helpful message.

---

## Phase 4: `hookset exec` — The Staging Engine

**Goal:** Implement the subcommand that Git actually calls (either directly via native 2.54 hooks or via `hookset run`). This is the core of lint‑staged‑replacement.

### Tasks

1. **Parse arguments**
   - `hookset exec --match "*.ts" --match "*.js" -- <command>`.
   - Environment variable `HOOKSET_EVENT` (set by `hookset run`) to know the event context.
   - Optionally read config for additional options (like `--only-changed`).
2. **Determine staged files**
   - `git diff --cached --name-only --diff-filter=ACMR` (new, modified).
   - Apply path matching using git’s own `pathspec` or a glob library.
   - If no files match, exit 0 immediately (skip command).
3. **Stash‑unmodified workflow**
   - Create a stash of unstaged changes: `git stash push --include-untracked --keep-index -m "hookset pre-commit"`.
   - If stash fails (e.g., no changes), continue without stashing.
   - Run the command with the list of matching files (as arguments? or via stdin? decide). For compatibility with most tools, pass filenames as arguments: `npx eslint file1.ts file2.ts`.
   - Wait for command completion, capture exit code.
   - If command modifies files, re‑add them: `git add -- <file list>`.
   - Always attempt to pop the stash: `git stash pop` (or `git stash pop --index` to restore index state? careful — we want to keep the staged changes after re‑adding. The usual lint‑staged flow: after running linter, they `git add` the files again, then stash pop drops the stash of unstaged changes. The original staged files were already in the index after the stash. So just `git add` after the tool run will update the index with the tool's modifications. Then `git stash pop` restores the working tree unstaged changes. No need for `--index` on pop because the index is already what we want. Test this.)
   - If the command fails, still pop stash, propagate exit code.
4. **Handle edge cases**
   - Deleted files: tool may delete a file; detect and `git rm -- <file>`.
   - Partial staging: if a file was only partially staged, re‑adding the whole file would stage the entire file. Document as known behaviour (like lint‑staged). Optionally add a `--respect-partial` flag for later.
   - Submodules: skip (or option to include? No, ignore for now).
   - Large repos, binary files: stash might be slow; warn if large files are detected.
5. **Verbose/debug mode** – `--verbose` logs every step, from file filtering to stash creation and command output.

### Acceptance Criteria

- [ ] A modified `.js` file that fails eslint results in commit abort and stash is cleanly popped.
- [ ] When only `.css` files are staged and no JS linter matches, the linter is skipped and exit code 0.
- [ ] Deleted staged file is properly removed from index after tool run.
- [ ] Existing unstaged changes are untouched after a successful or failed hook.
- [ ] Integration test simulates a real pre‑commit hook using both native (Git 2.54) and fallback paths.

---

## Phase 5: Compatibility Layer & `hookset run`

**Goal:** Seamlessly support Git ≥ 2.54 and older versions, and provide a single entry‑point for fallback hooks.

### Tasks

1. **Implement `hookset run <event>`**
   - Reads all configured hooks for the event from local and global config (using `internal/gitconfig`).
   - Executes them sequentially respecting `enabled` flag.
   - Sets `HOOKSET_EVENT` env variable for each execution.
2. **Auto‑detection in `hookset add`**
   - At `add` time, run `git version`, compare with 2.54.0.
   - If ≥ 2.54, write native config and (optionally) print a message that no fallback script is needed.
   - If < 2.54, write config anyway (for `hookset run` to read) and install fallback script.
3. **Upgrade path**
   - If user upgrades Git to ≥ 2.54, `hookset upgrade` can remove fallback scripts and keep native config only.
4. **Mixed environments** – a repo might be used by multiple developers with different Git versions. The native config alone works for 2.54+, but for older users the fallback script must also be present. `hookset add` could install both if `--legacy` flag is given. Document this.

### Acceptance Criteria

- [ ] On Git 2.54, adding a hook writes only config, no `.git/hooks/` files.
- [ ] On Git 2.45, a fallback `.git/hooks/pre-commit` is created and calls `hookset run pre-commit`.
- [ ] `hookset list` works correctly regardless of fallback state.
- [ ] Developer with older Git can clone a repo where `hookset add` was used on 2.54; running `hookset install` (or a dedicated command) generates the fallback scripts needed (maybe a `hookset init` command to be run after clone). This task is about designing that workflow.

---

## Phase 6: Migration Tools

**Goal:** Let users convert existing husky, lefthook, and lint‑staged configurations to `hookset` with one command.

### Tasks

1. **`hookset migrate --from lint-staged`**
   - Find `lint-staged` config in `package.json`, `.lintstagedrc`, `lint-staged.config.js` (static evaluation only).
   - For each pattern–command pair, emit equivalent `hookset add` commands (print or execute with `--yes`).
   - Handle array of commands per pattern, `--match` for multiple extensions.
   - Warning if `function` config is used (can't automatically migrate).
2. **`hookset migrate --from husky`**
   - Read `.husky/pre-commit` and other hook files.
   - Parse out the commands (often just `npx lint-staged` or a single runner).
   - Output `hookset` commands that, if lint‑staged was the only thing, directly calls the linters with `hookset exec`.
3. **`hookset migrate --from lefthook`**
   - Parse `lefthook.yml`.
   - Map each hook’s `commands` (with glob, run, etc.) to `hookset add`.
4. **`--dry-run`** – print what would be added without changing anything.
5. **Idempotency** – can run migration multiple times without duplicate hooks (check existing names).

### Acceptance Criteria

- [ ] `hookset migrate --from lint-staged --dry-run` on a sample `package.json` prints correct `hookset add` commands.
- [ ] Running with `--yes` actually writes the hooks and produces a working pre‑commit setup identical in effect.
- [ ] Migration from husky that had only `npx lint-staged` results in direct linter commands (no lint‑staged left).
- [ ] Existing `.husky/` directory is untouched (user removes it manually or via hookset’s cleanup prompt).

---

## Phase 7: Distribution & Installation

**Goal:** Make `hookset` trivially installable on any developer machine.

### Tasks

1. **Build the `curl | sh` / `irm | iex` installer**
   - Script detects OS/arch, downloads the latest binary from GitHub Releases, places it in `~/.local/bin` or `Program Files`.
   - Add to PATH if possible, or print instructions.
2. **Homebrew formula** – create `hookset.rb`, submit to homebrew-core or maintain a tap.
3. **winget / scoop manifests** for Windows.
4. **NPM thin wrapper** – `packages/npm/` with `postinstall` that downloads the binary for that platform. Expose a `hookset` bin that directly delegates. Useful for CI.
5. **Test install on clean VMs** for each OS.
6. **Goreleaser** update to attach `.tar.gz` and `.zip` to releases with correct naming.

### Acceptance Criteria

- [ ] One‑liner install works on fresh macOS, Ubuntu, Windows (via PowerShell).
- [ ] `brew install hookset` works and puts the binary in PATH.
- [ ] `npx hookset` downloads the binary on first run and works.

---

## Phase 8: Testing & CI Matrix

**Goal:** Ensure reliability across platforms and Git versions.

### Tasks

1. **Unit tests** for `internal/gitconfig`, `internal/exec` stash logic, migration parsers.
2. **Integration tests** – use `go test` with real `git` commands, covering Git 2.54 native mode and fallback mode (test against two different Git versions in CI).
3. **End‑to‑end tests** in Docker containers simulating a developer workflow.
4. **Windows‑specific tests** – file locking, path separators, PowerShell installer.
5. **Performance** – ensure stash/pop on a repository with 10k files is still under a second.
6. **Backward compatibility** – hooks created with future `hookset` versions remain readable.

### Acceptance Criteria

- [ ] CI matrix includes: ubuntu (latest Git), ubuntu (Git 2.45), macOS, Windows.
- [ ] All tests pass; no flaky tests.
- [ ] Benchmark test shows exec time for a no‑op hook is under 50 ms.

---

## Phase 9: Documentation & Community

**Goal:** Excellent first‑run experience and discoverability.

### Tasks

1. **README.md** with quickstart, comparison table, install methods.
2. **`hookset.dev` website** with interactive demo, migration guide.
3. **Man page** generated from CLI help (`hookset help man` or cobra/doc).
4. **Shell completions** (bash, zsh, fish, PowerShell).
5. **Troubleshooting guide** – common stash/pop failures, Windows Defender, CI setup.
6. **Contributing guide** – how to build, test, add new migration sources.

### Acceptance Criteria

- [ ] A new user can go from zero to running hooks in <5 minutes using only the README.
- [ ] Website includes a “Playground” that simulates `hookset add`.

---

## Phase 10: Post‑MVP Enhancements

Once the core is solid, expand functionality.

### Candidate features (prioritised by community demand)

1. **Parallel execution** – `--jobs N` flag in `hookset exec` to run multiple commands in parallel (like lefthook’s `parallel: true`).
2. **Hook templates** – `hookset init --template` creates a starter config with common linters.
3. **`hookset watch`** – file‑watcher mode that runs hooks on save (like `lefthook run --watch`).
4. **Extended migration** – support for `pre-push` hooks, commit‑msg hooks, etc.
5. **Plugin system** – allow community‑contributed migration parsers for other tools (e.g., `cargo-husky`, `overcommit`).
6. **CI‑friendly summary** – `hookset check` to verify hooks are up‑to‑date without running them (for CI linting).
7. **Graphical installer** for less terminal‑comfortable developers (a small macOS/Win app that installs the binary and adds to PATH).

---

## Timeline Estimation (rough)

- Phase 0–2: **3 weeks** (research, infrastructure, config)
- Phase 3–4: **4 weeks** (CLI management & exec engine)
- Phase 5: **2 weeks** (compatibility & fallback)
- Phase 6: **2 weeks** (migration)
- Phase 7–8: **3 weeks** (distribution, testing)
- Phase 9: **2 weeks** (docs, website)
- Total: ~16 weeks to a polished 1.0 release by a single full‑time developer.

---

## Risk Register

| Risk                                         | Mitigation                                                                                                  |
| -------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| Git 2.54 adoption is slow                    | Ensure flawless fallback; proactively document that pre‑2.54 users still get the same experience.           |
| Stash conflicts on dirty working trees       | Fail gracefully, guide user to commit or stash manually.                                                    |
| Windows AV blocking Go binary                | Code‑sign the binary; provide checksums; use known install paths.                                           |
| Migration breaks complex lint‑staged configs | `--dry-run` and manual review; fallback to running lint‑staged itself inside `hookset exec` as last resort. |

---

_This roadmap is a living document. Final decisions on fallback UX and edge‑case handling will be made during Phase 0._
