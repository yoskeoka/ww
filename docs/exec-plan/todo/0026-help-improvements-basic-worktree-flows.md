# 0026: Help Improvements for Basic Worktree Flows
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Make `ww` self-explanatory for first-time users and agents that only consult
command help, without requiring a separately distributed workflow skill.

This plan improves `ww --help`, `ww create --help`, and `ww cd --help` so the
basic worktree lifecycle and the recommended create-vs-cd flows are visible at
the CLI entry point, including the explicit create-and-enter path via
`ww create -q`.

## Context

- The current top-level help in `cmd/ww/main.go` lists commands and flags but
  does not show a basic usage flow or recommended examples.
- The user wants `ww` to explain itself even when it has not yet been embedded
  in a larger workflow package.
- `docs/project-plan.md` positions `ww` as agent-friendly and human-usable,
  so the built-in help should teach the common path directly.

## Expected Outcome

- `ww --help` shows:
  - a compact lifecycle-oriented flow using `ww list`, `ww create`, `ww cd`,
    and `ww clean`
  - concrete examples for create-and-enter, open-existing, and workspace-root
    `--repo` usage
- `ww create --help` explains that `ww create -q` is the preferred
  create-and-enter path
- `ww cd --help` explains that `ww cd` is for opening an existing worktree and
  points create-and-enter users back to `ww create -q`
- README/spec text stays aligned with the CLI help contract

## Scope

- Improve built-in help output for:
  - `ww --help`
  - `ww create --help`
  - `ww cd --help`
- Add durable examples and flow wording to docs/specs where needed so the help
  text is not a one-off undocumented behavior.
- Clarify the user-facing role split:
  - create and enter immediately: `ww create -q`
  - open an existing worktree: `ww cd`

## Out of Scope

- New subcommands or aliases
- Full tutorial output or interactive onboarding wizard
- Skill packaging and distribution
- Changing the core command semantics beyond what is needed to support clearer
  help text

## Code Changes

- Update `cmd/ww/main.go` top-level usage output to include:
  - a short "basic flow" section covering `list -> create -> cd -> clean`
  - concrete examples for:
    - `cd "$(ww create -q feat/my-feature)"`
    - `ww cd feat/my-feature`
    - `ww create --repo backend feat/my-feature`
- Update `cmd/ww/sub_create.go` help output to explain immediate-entry usage and
  point users to `ww create -q`.
- Update `cmd/ww/sub_cd.go` help output to explain that `ww cd` opens an
  existing worktree and to redirect create-and-enter flows to `ww create -q`.
- Introduce shared help-formatting helpers only if doing so keeps command help
  consistent without making the CLI wiring harder to read.

## Spec Changes

- Update `docs/specs/cli-commands.md` so the help-facing guidance is durable:
  - top-level help includes a basic workflow example set
  - `ww create` help highlights `-q` for create-and-enter
  - `ww cd` help highlights its existing-worktree role
- Update `docs/specs/shell-integration.md` if needed so the same create-vs-cd
  role split is stated consistently outside the CLI help output.
- Update `README.md` if needed so the examples shown there match the new
  top-level help wording and ordering.

## Design Notes

- Keep top-level help compact. It should teach the default mental model quickly,
  not become a full manual.
- The flow section should be descriptive, not normative in a way that conflicts
  with real variations such as starting from `ww create -q` directly.
- Reuse the same wording for create/cd role separation across help, specs, and
  README to avoid drift.

## Sub-tasks

- [ ] Define the exact top-level help structure: usage, command list, basic
  flow, examples, then global flags
- [ ] [parallel] Update `docs/specs/cli-commands.md` with the intended help
  guidance contract
- [ ] [parallel] Update `README.md` examples/order if the durable docs need to
  mirror the new top-level flow
- [ ] [depends on: help structure] Implement top-level help improvements in
  `cmd/ww/main.go`
- [ ] [depends on: help structure] Implement targeted guidance in
  `cmd/ww/sub_create.go` and `cmd/ww/sub_cd.go`
- [ ] [depends on: implementation] Add or update CLI help tests if the current
  suite already covers help output; otherwise add focused coverage for the new
  sections

## Verification

- `go test ./...`
- Manual output checks:
  - `ww --help`
  - `ww create --help`
  - `ww cd --help`
- Confirm the examples and flow wording still make sense for:
  - single-repo usage
  - workspace-root usage with `--repo`

## Risks

- Overly verbose help text would reduce scanability and hide the important
  examples.
- If top-level and subcommand help diverge, users will see conflicting guidance.
- Help text that over-prescribes one shell pattern could confuse users on shells
  where command substitution syntax differs; examples should stay explicit about
  what is being illustrated.
