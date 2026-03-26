# Phase 3: Polish Scope, Naming Freeze, and Release Readiness

> **Execution**: Use `/plan-execution` to split this plan into child plans before implementation.

**Objective:** Define the actual implementation scope for `ww` Phase 3, freeze the product name before first release work, and define the child-plan split for Phase 3 execution across shell integration, installation/distribution, and end-user documentation.

## Context

Phase 2 is functionally complete: workspace detection, cross-repo listing, clean, and `--repo` are already implemented. The remaining roadmap entry for Phase 3 is currently too broad:

- shell integration (`ww cd`, `cd $(ww create feat/x)`)
- Homebrew formula
- documentation

That wording leaves two open questions that must be resolved before implementation starts:

1. What is the concrete shell contract for navigation-oriented workflows?
2. Is the tool name frozen as `ww`, or should Phase 3 absorb a rename before the first public release?

Interactive mode already has its own separate Phase 4 plan in `docs/exec-plan/todo/interactive-mode-mvp.md` and should stay out of Phase 3.

## Reviewed Decisions / Constraints

Past decisions reviewed before planning:

- `docs/design-decisions/core-beliefs.md`
  - AI-first and composability still matter more than decorative human UX.
  - Do not refactor already-stable behavior without a concrete payoff.
- `docs/design-decisions/adr.md`
  - Workspace behavior is intentionally predictable and centralized.
  - Non-interactive deletion commands intentionally avoid confirmation prompts; Phase 3 shell integration should preserve scriptability rather than introduce magical shell state changes.
- `docs/exec-plan/done/005-silent-copy-symlink-failures.md`
  - Warnings already go to `stderr` to avoid interfering with machine-readable or shell-oriented `stdout`. This is a useful precedent for any path-printing behavior added in Phase 3.

## Direction

### Naming

Freeze the Phase 3 / first-release name as **`ww`**.

Rationale:

- `ww` is already embedded in the repo name, Go module path, binary name, docs, examples, and config filename (`.ww.toml`).
- `wwm` is slightly clearer in isolation, but it does not add enough semantic value to justify a repo/module/binary/doc rename before the first release.
- `grov` is novel, but it is semantically opaque in a git/worktree context and weakens immediate discoverability.

If a real naming blocker appears later (package-manager collision, trademark concern, or severe discoverability issue), handle it as a dedicated docs-first plan before tagging `v1.0.0`. Phase 3 should not spend its main effort on a speculative rename.

### Phase 3 Scope

Phase 3 should focus on **shipping polish**, not on adding a second interaction model. Concretely, that means:

1. **Shell integration primitives**
   - Add an explicit path-oriented navigation contract.
   - Avoid changing shell state directly from the binary.
   - Preserve existing text output unless there is a strong reason to make a breaking change.
2. **Release and installation readiness**
   - Make `go install` and Homebrew installation concrete and repeatable.
   - Add the minimal release automation or versioning support needed to keep the Homebrew path maintainable.
3. **User-facing documentation**
   - Add first-release quality docs for installation, common workflows, shell setup, and workspace-mode behavior.

### Shell Integration Design Choice

Phase 3 should prefer **explicit path-only interfaces** over changing the default output of existing commands.

Planned direction:

- Add `ww cd` as a path-resolver command for navigation use cases.
- Add a path-only flag to `ww create` so command substitution is explicit and non-breaking.
- Keep human-readable status/progress on `stderr` whenever Phase 3 introduces shell-oriented `stdout` output.

This keeps the CLI composable for shells and agents without silently changing the default contract of `ww create`.

## Child Plan Strategy

This file is a **parent phase plan**, not the final implementation unit.

Before `/plan-execution`, Phase 3 should be split into child plans so review stays focused and specs remain readable. The expected split is:

- `phase3-shell-integration`
- `phase3-install-and-versioning`
- `phase3-user-docs`

If documentation stays tightly coupled to installation work, the latter two may be merged into one child plan. Shell integration should stay separate.

## Planned Deliverables

### Deliverable 1: Shell Integration

Specify and implement a minimal shell-navigation surface:

- `ww cd` resolves a destination and prints the absolute path only
- `ww create` gains an explicit path-only mode for command substitution
- docs include shell wrapper examples rather than promising impossible in-process `cd` behavior from the binary itself

### Deliverable 2: Installation / Distribution

Make installation paths concrete:

- verify and document `go install`
- add Homebrew formula support
- add or document the release/versioning workflow needed to keep Homebrew metadata current

### Version Strategy

Current behavior is **commit-hash only**:

- `Makefile` injects `main.CommitHash` from `git rev-parse --short HEAD`
- `ww version` prints `ww version <hash>` or `ww version dev`

That is acceptable for local development, but it is not sufficient as the primary public release strategy because:

- Homebrew expects stable, ordered release versions
- users need a human-meaningful version to report bugs against
- docs and release notes need a stable identifier independent of a particular clone state

Phase 3 should adopt a **dual strategy**:

- **Release artifacts**: SemVer tags, starting in the `v0.x.y` range until the CLI surface is considered stable
- **Dev / untagged builds**: retain commit-hash provenance

Planned output direction:

- tagged release build: `ww version v0.x.y`
- dev build: `ww version dev+<short-hash>` or equivalent commit-identifiable form

If desired, tagged builds may also include commit metadata in a secondary field or verbose form, but the primary public version identifier should be SemVer rather than a raw hash.

### Deliverable 3: Documentation

Publish end-user docs that match actual behavior:

- install instructions
- shell integration usage
- workspace examples
- cleanup / safety guidance

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/cli-commands.md` | Add `ww cd` and the chosen path-only create behavior |
| `docs/specs/shell-integration.md` | New spec for path-printing contracts, stdout/stderr rules, and shell wrapper examples |
| `docs/specs/release-versioning.md` | New spec for release tags, `ww version` output, and dev-vs-release build metadata |

## Docs Changes

| File | Change |
|------|--------|
| `docs/project-plan.md` | Freeze naming direction as `ww` and clarify the concrete contents of Phase 3 |

## Design Decision Changes

| File | Change |
|------|--------|
| `docs/design-decisions/adr.md` | Record the Phase 3 shell integration contract: explicit path-only interfaces, no hidden shell-state mutation, and `stderr` for human context |

## Code Changes

| File | Change |
|------|--------|
| `cmd/ww/main.go` | Register `cd` subcommand and any new create flags |
| `cmd/ww/version.go` and `cmd/ww/main.go` | Extend version metadata from commit-hash-only to release-aware output |
| `cmd/ww/sub_cd.go` | New subcommand for path resolution |
| `cmd/ww/sub_create.go` | Add explicit path-only output mode |
| `cmd/ww/helpers.go` | Share repo/workspace/path resolution logic as needed |
| `worktree/` | Expose or refine path-resolution helpers needed by shell integration |
| `.github/` and release metadata files | Add the minimal automation/files required for Homebrew distribution |

Exact packaging file names may depend on the chosen release approach, but the implementation must keep version injection and install metadata auditable in-repo.

## Docs Changes

| File | Change |
|------|--------|
| `README.md` | Add install, quick start, and shell integration examples |
| `docs/spec-code-mapping.md` | Map new shell-integration and release-versioning specs to implementation |
| `docs/specs/README.md` | Link the new shell-integration spec if needed |

## Sub-tasks

- [ ] [parallel] Update `docs/project-plan.md` to freeze `ww` as the release name and clarify the concrete scope of Phase 3
- [ ] [parallel] Add `docs/specs/shell-integration.md` and update `docs/specs/cli-commands.md` with the final shell contract
- [ ] [parallel] Add `docs/specs/release-versioning.md` describing SemVer releases and commit-aware dev builds
- [ ] [parallel] Append an ADR entry for explicit path-only shell integration
- [ ] [depends on: shell specs, ADR] Implement `ww cd` path resolution
- [ ] [depends on: shell specs] Add explicit path-only output mode to `ww create`
- [ ] [depends on: naming freeze, release-versioning spec] Add installation/release files, Homebrew support, and release-aware version metadata
- [ ] [depends on: shell specs] Write user-facing docs for shell setup, install, and common workflows
- [ ] [depends on: implementation] Add or update tests covering path-only output, stdout/stderr separation, and release metadata validation

## Parallelism

Independent planning and execution streams:

- naming/roadmap clarification
- shell integration spec + implementation
- install/versioning work
- user documentation work

Phase 3 execution should be split into child plans before implementation. The default split is:

- `phase3-shell-integration`
- `phase3-install-and-versioning`
- `phase3-user-docs`

## Verification

- `ww` remains the documented product/binary name across plan, specs, and user docs
- `ww cd` prints only the resolved absolute path on success
- `ww create` supports explicit command-substitution-friendly output without breaking existing default text usage
- human-readable context for shell-oriented flows goes to `stderr`
- tagged releases use SemVer, while untagged builds remain commit-identifiable
- installation works via `go install` and the documented Homebrew path
- docs show an accurate end-to-end setup for single-repo and workspace modes
