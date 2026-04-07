# Architectural Decision Records (ADR)

## [YYYY-MM-DD] Title of Decision

### Context
[Describe the issue or problem.]

### Decision
[Describe the decision made.]

### Consequences
[Describe the positive and negative consequences.]

---

## [2026-03-18] Config type layering: worktree.Config vs flat fields

### Context

`worktree.Manager` depended on `internal/config.Config`, making the public `worktree` package unusable as a library. Three options were considered:

- **Option A**: Move `internal/config` to a public package. Rejected — it couples library consumers to TOML parsing and file search logic they don't need.
- **Option B**: Replace `Config` with flat fields on `Manager`. Simple, but fields will grow as features are added (Phase 2 multi-repo, Phase 4 hooks), bloating the Manager struct.
- **Option C**: Create a public `worktree.Config` type that Manager accepts. CLI layer maps `internal/config.Config` → `worktree.Config` in `newManager()`.

### Decision

Option C. Two separate config types for two separate concerns:

- `worktree.Config` — what Manager needs to operate (public, stable API for library consumers)
- `internal/config.Config` — how `.ww.toml` is parsed (CLI concern, free to change format/fields)

### Consequences

- **Positive**: Library consumers can construct `worktree.Config` without importing internal packages. Config fields stay grouped. Adding new config fields doesn't pollute Manager's top-level API.
- **Negative**: DTO mapping in `newManager()` is boilerplate. Acceptable — it's a handful of field assignments in one place, and it makes the layer boundary explicit.

---

## [2026-03-18] Workspace detection algorithm: parent-scan strategy

### Context

Phase 2 adds workspace awareness — detecting when `ww` is inside a meta-repo with multiple child git repositories. Several detection strategies were considered:

- **Config-only**: Require explicit `workspace = true` in `.ww.toml`. Simple but violates "convention over configuration" — new users must configure before workspace features work.
- **Recursive scan**: Walk up the directory tree and scan all children at each level. Correct but slow and risks false positives on deeply nested repos.
- **Parent-scan (chosen)**: Check only the immediate parent directory for git siblings and the current directory for git children. Limited to one level in each direction.

### Decision

Parent-scan strategy with a 7-step algorithm:
1. Scan CWD children for `.git` entries (parent candidate)
2. Determine current git repo root
3. Check parent directory for `.git`
4. Check parent's children for `.git` siblings
5. (Reserved for future config override)
6. Fall back to CWD as workspace root if step 0 found children
7. None → single-repo mode

### Consequences

- **Positive**: Fast (only two directory scans), predictable, zero-config for standard meta-repo layouts.
- **Negative**: Cannot detect workspaces more than one level deep. Accepted — FR-22 (recursive detection) is reserved for the future if needed.

---

## [2026-03-18] Worktree path layout: centralized `.worktrees/` in workspace mode

### Context

In single-repo mode, worktrees are created as siblings: `<repo-parent>/<repo>@<branch>`. In workspace mode with N child repos, sibling layout would scatter worktrees across child directories, making them hard to find and clean up.

### Decision

Workspace mode centralizes all worktrees under `<workspace_root>/.worktrees/<repo>@<branch>`. Single-repo mode keeps the sibling layout. An explicit `worktree_dir` in `.ww.toml` overrides either default.

### Consequences

- **Positive**: Single location for all worktrees across all repos. Easier cleanup, listing, and mental model.
- **Negative**: Worktrees are not adjacent to their source repo. Accepted — `ww list` provides the lookup and most editors/agents use absolute paths.

---

## [2026-03-18] Child repos are never workspace roots (no recursive nesting)

### Context

If a child repo could itself be treated as a workspace root (e.g., a child that has its own git children), workspace detection could recurse indefinitely or produce ambiguous results.

### Decision

Child repos are never treated as workspace roots. Detection stops at one level. This is an explicit invariant enforced in the detection algorithm.

### Consequences

- **Positive**: Deterministic detection, no ambiguity, simple mental model.
- **Negative**: Nested workspace-in-workspace layouts are not supported. Accepted — FR-22 reserves this for the future.

---

## [2026-03-18] Worktree STATUS: merged > stale, no-tracking = active

### Context

`ww list` adds a STATUS column (`active`/`merged`/`stale`). A branch could theoretically be both merged and stale (merged into base AND remote branch deleted). Branches without remote tracking could be considered stale by absence, but this would flag local-only feature branches incorrectly.

### Decision

- `merged` takes precedence over `stale` (if both conditions are met).
- Branches with no remote tracking configured are always `active`, never `stale`.
- Main worktrees are always `active`.

### Consequences

- **Positive**: Conservative — only flags worktrees as cleanable when there's strong evidence. Avoids false positives on local-only branches.
- **Negative**: A branch that was never pushed will stay `active` even if abandoned. Accepted — FR-23 (time-based stale detection) can address this later.

---

## [2026-03-18] `ww clean` has no confirmation prompt

### Context

Most CLI tools that delete data ask for confirmation (`rm -i`, `git clean -i`). However, `ww clean` targets a specific audience (developers and AI agents) and has explicit preview mechanisms.

### Decision

`ww clean` executes immediately without a confirmation prompt. Users preview with `ww list --cleanable` or `ww clean --dry-run`.

### Consequences

- **Positive**: Scriptable, agent-friendly, no interactive input required. Consistent with `ww remove` which also has no prompt.
- **Negative**: Risk of accidental deletion. Mitigated by safe defaults (`git branch -d` fails on unmerged branches) and `--force` being opt-in.

---

## [2026-03-19] Integration tests: testcontainers-go with testing.Short() split

### Context

Phase 2 workspace tests need isolation from host git config and filesystem. Two approaches were considered:

- **Standalone Docker runner**: Dockerfile.test that builds and runs tests inside a container, invoked via `make test-docker`. Requires a separate test execution model — assertion libraries (bats, shell scripts) or a compiled test binary copied into the image.
- **testcontainers-go inside `go test`**: Integration tests use testcontainers-go to spin up containers from within `go test`. Tests stay in Go's standard framework with `t.Error`/`t.Fatal` assertions.

The standalone approach was rejected because `ww` is a single-binary CLI with no service dependencies — maintaining a separate test suite and assertion framework adds complexity without proportional benefit.

### Decision

- Use **testcontainers-go** to run integration tests inside Docker containers from `go test`.
- Split tests with `testing.Short()`: `make test` runs `go test -short` (unit only, no Docker), `make test-integration` runs integration tests (Docker required).
- CI runs both as separate jobs.

### Consequences

- **Positive**: Single test framework (`go test`), no shell-based assertion tooling, developers get fast `make test` without Docker, CI has clear separation.
- **Negative**: testcontainers-go adds a dependency. Acceptable — it's test-only and well-maintained.

---

## [2026-03-29] Phase 3 shell integration contract: explicit path-only interfaces

### Context

Phase 3 adds shell-oriented navigation workflows such as `wcd` wrappers and `cd $(ww create -q feat/x)`. The binary cannot change the parent shell's current working directory directly, so the CLI contract needs to stay composable instead of pretending to mutate shell state.

Several interface shapes were considered:

- **Change default `ww create` text output to path-only**: Rejected — it would silently break existing human-oriented output and make the default UX less clear.
- **Add a path-printing subcommand plus an opt-in quiet mode**: Chosen — it preserves the current defaults while enabling shell composition.
- **Use `--path` / `--print-path` instead of `-q` / `--quiet`**: Rejected — `--path` sounds like an input location flag, and `--print-path` is too verbose for frequent interactive shell use.

### Decision

- Add `ww cd` as a read-only path resolver that prints the absolute worktree path.
- Add `-q` / `--quiet` to `ww create` so shell users can request path-only output explicitly.
- Keep path-oriented success output on `stdout`.
- Keep human-readable context and progress on `stderr` whenever shell-oriented flows are in use.
- Do not claim that the binary itself can change the parent shell's cwd; shell wrappers such as `wcd() { cd "$(ww cd "$@")"; }` are the supported pattern.

### Consequences

- **Positive**: Shell workflows stay explicit and scriptable without breaking the existing default output contract of `ww create`.
- **Positive**: The stdout/stderr split makes command substitution and agent usage predictable.
- **Negative**: There are now two related navigation entry points (`ww cd` and `ww create -q`) instead of one magical default. Accepted — the explicitness is the point.

---

## [2026-03-31] Workspace detection anchor: nearest containing workspace within a bounded window (current dir + main root/parent/grandparent)

### Context

Workspace detection originally aimed to stay local: reason from the current repository, look one level up for a containing workspace, and look one level down for child repositories. The current implementation drifted beyond that and added a grandparent guard: when the parent directory is itself a git repository, detection also inspects the grandparent and rejects the parent as a workspace root if the grandparent exposes multiple git child repositories.

That extra check causes false negatives in practical meta-repo layouts such as:

- `/parent-workspace/ww`
- `/parent-workspace/another-child-repo`
- `/some-grandparent/parent-workspace`
- `/some-grandparent/another-repo`

From `ww`, the intended workspace root is `parent-workspace`, but the grandparent guard forces single-repo mode instead.

Two approaches were considered:

- **A. Strict one-up/one-down anchor**: Always reason from the current repo's main root; if its parent is a git repo, accept that parent immediately as the workspace root. This is simple and matches the original intuition, but it leaves less room for slightly deeper layouts where the containing workspace is one more level up.
- **B. Nearest containing workspace within a bounded window**: Resolve the current repo's main root, then search upward by at most two levels for workspace-root candidates. A candidate qualifies only if it contains the current main repo root and has at least two immediate child real git repositories. Candidates are tested in order: current directory first, then parent of the main repo root, then grandparent of the main repo root. If multiple candidates qualify, pick the nearest one. If the current directory itself already qualifies as a workspace root, accept it immediately. This preserves locality while handling the practical "repo inside repo inside parent directory" cases without recursing arbitrarily.

### Decision

Choose **B**.

Workspace detection is anchored on the current repository's **main worktree root**. Detection may inspect at most:

- the current directory as an immediate workspace-root candidate
- the main repo root
- the parent of the main repo root
- the grandparent of the main repo root

Within that bounded window:

- A workspace-root candidate must contain the current main repo root.
- A workspace-root candidate must expose at least two immediate child **real git repositories**.
- If multiple candidates qualify, the nearest containing candidate wins.
- Managed git worktrees are not treated as real git repositories for workspace detection.

This choice is primarily about matching human expectation in practical 1-2-3 level layouts. When a user is operating from level 3, they may still reasonably expect either level 2 or level 1 to be treated as the active workspace, depending on which directory actually behaves as the containing workspace. Approach A only handles the nearest parent-style case and fails when level 1 is the expected workspace. Approach B preserves that expectation by allowing both containing levels to be considered within a bounded window, then selecting the nearest directory that truly qualifies as a workspace root. That "nearest qualifying workspace wins" rule is also more intuitive than a hard-coded parent-only rule when both level 1 and level 2 could plausibly be treated as workspace containers.

### Consequences

- **Positive**: Restores the intended local mental model while avoiding the current false-negative behavior for git-backed workspace roots.
- **Positive**: Detection remains bounded and deterministic; it does not recurse through arbitrary ancestors.
- **Negative**: The rules are more complex than the simple one-up/one-down model and require explicit tie-breaking.
- **Negative**: Future clone-based isolation (`FR-16`) must mark `ww`-managed clones so they are excluded from "real git repository" detection.

---

## [2026-04-04] Dual version strategy: SemVer releases plus commit-aware dev builds

### Context

Phase 3 adds first-release packaging and Homebrew distribution. The previous `ww version` contract only exposed a commit hash, which is enough for local dev builds but not for stable public releases or package managers.

Several options were considered:

- **Commit-hash only**: Keep the current model for every build. Rejected because package managers and users need stable, ordered release versions.
- **SemVer only**: Require every build to carry a release version. Rejected because local builds and untagged CI snapshots still need a useful identifier without pretending to be a release.
- **Dual strategy (chosen)**: Use SemVer tags for releases and keep commit-aware dev output for untagged builds.

GoReleaser was also considered against ad hoc shell scripts for packaging. Shell scripts were rejected because cross-platform archives, GitHub Releases, and Homebrew tap publishing would become harder to audit and maintain.

### Decision

Use a dual version strategy:

- Tagged release builds inject `Version=<SemVer tag>` and `CommitHash=<short hash>`.
- Untagged builds leave `Version` empty and inject only `CommitHash`.
- `ww version` prefers the SemVer tag when present, otherwise renders `dev+<short-hash>` or `dev`.
- `ww version --json` exposes stable `version` and `commit` fields, using `version=dev` for untagged builds.
- Use GoReleaser as the release automation entry point for:
  - cross-platform archive builds
  - GitHub Release publication
  - Homebrew tap publishing to `yoskeoka/homebrew-ww`

### Consequences

- **Positive**: Public releases get stable SemVer identifiers starting at `v0.3.0`.
- **Positive**: Local and CI dev builds stay commit-identifiable without fabricating a release version.
- **Positive**: Release metadata and distribution settings remain auditable in-repo.
- **Negative**: Build tooling now has to manage two metadata paths (`Version` and `CommitHash`) instead of one.

---

## [2026-04-08] Phase 4 interactive foundation: lightweight prompt surface with strict CLI parity

### Context

Phase 4 adds a human-oriented interactive entry point, `ww i`. The main risk is
allowing the interactive UI to drift into a separate capability surface with
behavior that cannot be expressed through the normal CLI. That would weaken
`ww`'s AI/scripting contract and create two inconsistent execution models.

The earlier Phase 3 shell integration ADR already established a strict stream
contract: machine/path-oriented output belongs on `stdout`, and human-readable
context belongs on `stderr`. The interactive mode foundation needs to preserve
that contract while introducing a prompt-based flow.

Several interaction models were considered:

- **Full-screen TUI**: Rejected for the MVP. It is heavier than needed and
  adds more UI state than the current scope requires.
- **Interactive-only behavior**: Rejected. Any action that mutates state or
  selects an externally meaningful result must remain available through the
  standard CLI.
- **Lightweight prompt flow (chosen)**: Use `ww i` as a guided entry point over
  existing CLI behavior, with fixed top-level actions and explicit stream
  routing.

For prompt tooling, the project direction is to use `huh` for the interactive
flow implementation because it fits grouped, guided prompts better than
single-prompt libraries. The foundation step keeps the session and dispatch
contracts independent from any single prompt adapter so later child plans can
wire `huh` without changing the command contract.

### Decision

- `ww i` is a lightweight prompt flow, not a separate full-screen application.
- Interactive mode is an orchestration layer over existing command/business
  logic and must obey strict non-interactive CLI parity.
- The fixed MVP top-level actions are `create`, `list`, `clean`, and `quit`.
- Interactive prompt rendering and human-readable context use `stderr`.
- `stdout` remains reserved for path-only or machine-consumable action results.
- `ww i --json` is rejected; users needing machine-readable output must use the
  standard non-interactive commands.
- The implementation direction for prompt UI is `huh`, but the foundation layer
  keeps session abstractions small so prompt-library changes do not alter the
  public CLI contract.

### Consequences

- **Positive**: Interactive mode stays additive and cannot become the only
  place where a capability exists.
- **Positive**: The Phase 3 shell contract survives intact, including future
  path-only `stdout` output for interactive `open`.
- **Positive**: The fixed action model and shared session abstractions make
  later child plans testable without requiring a real terminal in unit tests.
- **Negative**: The foundation step adds placeholder dispatch before the full
  child flows exist.
- **Negative**: Keeping parity with the non-interactive CLI may delay
  interactive UX ideas that require new command flags or commands first.
