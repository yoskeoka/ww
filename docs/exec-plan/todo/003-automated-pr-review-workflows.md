# 003: Automated PR Review Workflows

## Objective

Add three gh-aw (GitHub Agentic Workflows) that automatically review PRs based on file path patterns. Each workflow posts a PR Review (Approve / Request Changes) to enforce quality gates at different stages of the AI-Centered Development workflow.

**Operating model**: Start with Request Changes enforcement. False positives are handled by rule bypass, and each false positive gets logged as a `docs/issues/` entry to drive continuous improvement of the review prompts.

## Prerequisites

- `gh aw` CLI extension installed (`gh extension install github/gh-aw`)
- Repository initialized for gh-aw (`gh aw init`)
- CLAUDE.md updated with spec-to-code mapping (see Spec Changes below)

## Workflows

### 1. Plan Review

**Trigger**: PR diff includes new or modified files in `docs/exec-plan/todo/`.

**Context provided to reviewer**:
- The PR diff (plan content)
- Corresponding `docs/issues/` file if referenced in the plan

**Review criteria**:
- Does the plan clearly state the problem and proposed solution?
- Are spec changes identified?
- Are code changes broken into concrete sub-tasks?
- Are dependencies between sub-tasks specified?
- Is the scope appropriate (not too broad, not too narrow)?

**Output**: PR Review with Approve or Request Changes.

### 2. Implementation Review

**Trigger**: PR diff includes files moved from `docs/exec-plan/todo/` to `docs/exec-plan/done/` AND code files are changed.

**Context provided to reviewer**:
- The PR diff (code + plan move + spec updates)
- The plan file (now in `docs/exec-plan/done/`)
- Spec files listed as change targets in the plan

**Review criteria**:
- Does the implementation cover all sub-tasks in the plan?
- Are there missing sub-tasks that should have been implemented?
- Is there over-scoping (code changes not described in the plan)? If so, suggest filing as separate issues.
- Do spec updates match the code changes?
- Do tests cover the spec changes?

**Output**: PR Review with Approve or Request Changes.

### 3. Spec/Code Sync Check

**Trigger**: PR diff includes changes to `docs/specs/` OR code files mapped in CLAUDE.md.

**Context provided to reviewer**:
- The PR diff
- The spec-to-code mapping from CLAUDE.md

**Review criteria**:
- If code changed: is the corresponding spec updated (or is no spec update needed)?
- If spec changed: is the corresponding code/test updated (or is this a spec-only clarification)?
- Flag mismatches where a spec describes behavior that the code doesn't implement, or vice versa.

**Output**: PR Review with Approve or Request Changes.

## Spec Changes

Update `CLAUDE.md` to add spec-to-code mapping under Project Structure:

```
## Spec-to-Code Mapping

| Spec | Code directories | Test files |
|---|---|---|
| docs/specs/cli-commands.md | cmd/ww/ | integration_test.go |
| docs/specs/git-operations.md | git/ | git/git_test.go |
| docs/specs/configuration.md | internal/config/ | internal/config/config_test.go |
```

## Code Changes

No application code changes. All changes are workflow configuration files.

### `.github/workflows/plan-review.md`

- [ ] gh-aw workflow definition for Plan Review
- [ ] Prompt includes review criteria above
- [ ] Configured to trigger on PRs with `docs/exec-plan/todo/` changes

### `.github/workflows/impl-review.md`

- [ ] gh-aw workflow definition for Implementation Review
- [ ] Prompt includes review criteria and references plan + spec files
- [ ] Configured to trigger on PRs with `docs/exec-plan/done/` + code changes

### `.github/workflows/spec-code-sync.md`

- [ ] gh-aw workflow definition for Spec/Code Sync Check
- [ ] Prompt references CLAUDE.md mapping
- [ ] Configured to trigger on PRs with `docs/specs/` or mapped code directory changes

### `CLAUDE.md`

- [ ] Add Spec-to-Code Mapping table

## Sub-tasks

1. [ ] Update `CLAUDE.md` with spec-to-code mapping table
2. [ ] Run `gh aw init` if not already initialized
3. [ ] [parallel] Create `.github/workflows/plan-review.md` and compile
4. [ ] [parallel] Create `.github/workflows/impl-review.md` and compile
5. [ ] [parallel] Create `.github/workflows/spec-code-sync.md` and compile
6. [ ] [depends on: 3, 4, 5] Test each workflow with a dry-run PR
7. [ ] [depends on: 6] Document false-positive handling process in CLAUDE.md

## Verification

- [ ] Create a test PR that adds a file to `docs/exec-plan/todo/` -> Plan Review posts a review
- [ ] Create a test PR that moves a plan to `done/` with code changes -> Implementation Review posts a review
- [ ] Create a test PR that modifies `docs/specs/` without corresponding code -> Spec/Code Sync flags it
- [ ] Create a test PR that modifies code without corresponding spec -> Spec/Code Sync flags it
- [ ] A clean PR (all aligned) receives Approve from all applicable workflows

## Design Notes

- gh-aw workflows are defined as Markdown (`.md`) and compiled to YAML lock files (`.lock.yml`). Both are committed to the repo.
- The workflows are project-agnostic in structure. To port to another project, only the CLAUDE.md mapping table needs to change.
- False positives are expected during initial rollout. Each one should become a `docs/issues/` entry with the trigger, expected behavior, and actual behavior, to iteratively improve prompts.
