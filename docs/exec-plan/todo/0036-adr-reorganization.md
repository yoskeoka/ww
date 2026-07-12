# Reorganize Architecture Decision Records

> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## Objective and completion boundary

Move `ww`'s architecture decisions from the monolithic
`docs/design-decisions/adr.md` log into the workspace-standard ADR layout: a
compact index at `docs/design-decisions/README.md` and immutable, numbered
Michael Nygard records under `docs/design-decisions/adr/`.

Complete when every accepted historical decision has one numbered record, the
supersession relationships remain visible, the index has an ID/status/tag/outcome
row for every record, and the former monolithic file and its template stub are
gone. This is documentation organization only; it must not change `ww` CLI,
configuration, or runtime behavior.

## Existing implementation references

- `docs/design-decisions/adr.md` — current 16 decision records plus a
  non-record template stub; its headings and `Context`/`Decision`/
  `Consequences` sections are the migration source of truth.
- `docs/design-decisions/core-beliefs.md` — the existing decision-making
  principles that remain alongside the new ADR index.
- Workspace reference:
  `../docs/design-decisions/README.md` and `../docs/design-decisions/adr/` —
  the canonical index-and-record layout to mirror, including the Michael
  Nygard `Status`, `Context`, `Decision`, and `Consequences` form.
- `docs/exec-plan/todo/README.md` — active plan naming and lifecycle rules.

## Code change map

- (NEW) `docs/design-decisions/README.md` — ADR-management guidance and a
  compact table indexing every migrated decision.
- (NEW) `docs/design-decisions/adr/0001-config-type-layering.md` through
  `0016-materialization-profiles.md` — one immutable historical record per
  real decision, numbered in acceptance-date order.
- (DELETE) `docs/design-decisions/adr.md` — replace the monolithic log,
  including its unused `YYYY-MM-DD` template stub.

## Spec changes

N/A — this migration changes documentation retrieval and ADR governance only;
the behavioral specifications under `docs/specs/` remain unchanged.

## Execution steps

1. Create `docs/design-decisions/README.md` using the workspace format:
   state that records are immutable after acceptance, direct authors to create
   the next numbered file in `adr/`, and provide an ID, status, tags, and
   one-line outcome table.
2. Split every real record in `adr.md` into one Markdown file in
   `docs/design-decisions/adr/`, preserving its context, decision, and
   consequences without altering the historical technical outcome. Add a
   `Status` section to each record. Assign the IDs chronologically from the
   2026-03-18 config-layering decision through the 2026-07-07 materialization
   profiles decision.
3. Mark the 2026-03-19 testcontainers decision as superseded by the 2026-04-20
   host-native harness decision, and mark the earlier parent-scan workspace
   detection decision as superseded by the 2026-03-31 bounded nearest-workspace
   decision. Keep all other records accepted unless their source says
   otherwise.
4. Delete `docs/design-decisions/adr.md`; do not migrate its placeholder
   template as an ADR.
5. Verify the index links resolve, its rows cover exactly the migrated records,
   every record follows title/Status/Context/Decision/Consequences, and no
   tracked documentation still links to the removed monolithic path.

## Dependencies and parallelism

This is a single docs migration. The index depends on the final record IDs and
statuses; record extraction can otherwise be prepared independently.

## Verification

- Run a Markdown link checker or a focused script that confirms every index
  link exists and each record has the required Michael Nygard sections.
- Search tracked files for `docs/design-decisions/adr.md` and confirm no live
  reference remains.
- Run `make lint` if the repository's Markdown/documentation lint scope covers
  these files.

## Addresses

None.
