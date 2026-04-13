**Execution**: Use `/execute-task` to implement this plan.

# 024 Interactive Action Menu Height

## Objective

Make the `ww i` select-based menus size their visible option area correctly so
fixed action sets are fully visible and the worktree browser shows a stable
five-row viewport.

## Reproduction

1. Run `ww i` in a workspace with multiple repos.
2. Observe the top-level `Select action` prompt only showing part of the fixed
   action set unless the user scrolls.
3. Enter `list`, select a worktree, and observe the `Selected worktree` prompt
   only showing part of its action set.
4. Observe the worktree browser viewport using a height that does not match the
   intended visible-row count.

## Spec Changes

- Update `docs/specs/interactive-mode.md` to require the top-level action picker
  to render all fixed actions (`create`, `list`, `clean`, `quit`) without
  scrolling.
- Require the selected-worktree action picker to render all available actions at
  once.
- Document that the worktree browser shows up to five visible options before
  scrolling.

## Code Changes

- Update `internal/interactive/huh_ui.go` to size select fields using helpers
  that account for title and description rows.
- Add regression tests covering the helper calculations and fixed action set.

## Sub-tasks

- [ ] Add the menu-visibility requirements to the interactive mode spec.
- [ ] Adjust select height calculation for top-level actions, worktree list, and
  selected-worktree actions.
- [ ] Add regression tests for the helper functions and fixed action set.

## Design Decisions

- Keep the existing vertical `huh` UI. This change only fixes sizing so prompt
  content is visible without unnecessary scrolling.
