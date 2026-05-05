# GitHub Actions Pinning

Ordinary GitHub Actions workflow files and composite actions in this repository must manage `uses:` references through `pinact`.

## Operator Contract

- When editing ordinary workflow YAML files under `.github/workflows/` excluding `*.lock.yml`, or when editing `.github/actions/**`, use `pinact` to pin new `uses:` references and to update existing pinned references.
- Do not hand-edit floating version tags such as `@v6` when the change is intended to pin or refresh an action dependency; run `pinact` instead and review the resulting diff.
- `gh aw` source workflows (`.github/workflows/*.md`), generated lock files (`.github/workflows/*.lock.yml`), and `.github/aw/actions-lock.json` are out of scope for `pinact` and must continue to follow the `gh aw` toolchain instead.
- It is acceptable to scope a rollout by passing explicit file paths to `pinact run` so only ordinary workflow YAML files are updated.
