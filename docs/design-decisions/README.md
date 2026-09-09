# Architecture Decision Records

Records are immutable after acceptance. Create the next numbered file in `adr/`
using the Michael Nygard form: title, Status, Context, Decision, Consequences.

| ID | Status | Tags | Outcome |
| --- | --- | --- | --- |
| [0001](adr/0001-config-type-layering.md) | Accepted | architecture, api | `worktree.Config` separates the public library API from internal config parsing. |
| [0002](adr/0002-workspace-detection-parent-scan.md) | Superseded | workspace, detection | The original bounded parent-scan strategy was replaced by nearest qualifying workspace detection. |
| [0003](adr/0003-worktree-path-layout.md) | Accepted | worktree, layout | Workspace worktrees are centralized under `.worktrees/`. |
| [0004](adr/0004-child-repos-never-workspace-roots.md) | Accepted | workspace, detection | Child repositories are never recursively treated as workspace roots. |
| [0005](adr/0005-worktree-status-precedence.md) | Accepted | worktree, cleanup | Merged wins over stale and untracked branches remain active. |
| [0006](adr/0006-clean-without-confirmation.md) | Accepted | cli, cleanup | `ww clean` is immediate, with preview modes for safety. |
| [0007](adr/0007-testcontainers-integration-harness.md) | Superseded | testing, integration | The original Docker testcontainers harness is recorded for historical context. |
| [0008](adr/0008-shell-integration-contract.md) | Accepted | cli, shell | `ww cd` and quiet create provide explicit path-only shell interfaces. |
| [0009](adr/0009-workspace-detection-anchor.md) | Accepted | workspace, detection | Detection chooses the nearest qualifying workspace in a bounded window, superseding 0002. |
| [0010](adr/0010-dual-version-strategy.md) | Accepted | release, versioning | Releases use SemVer while untagged builds remain commit-aware dev builds. |
| [0011](adr/0011-interactive-foundation.md) | Accepted | cli, interactive | Interactive mode is a lightweight orchestration layer with CLI parity. |
| [0012](adr/0012-host-native-integration-harness.md) | Accepted | testing, integration | Host-native integration tests replace the Docker testcontainers harness. |
| [0013](adr/0013-sandbox-constrained-mode.md) | Accepted | sandbox, workspace | Sandbox mode bounds discovery and uses in-repository default worktrees. |
| [0014](adr/0014-global-config-path-and-overrides.md) | Accepted | config, global | Global config has an explicit XDG path and per-key local replacement. |
| [0015](adr/0015-global-project-matching.md) | Accepted | config, global | Ordered project path rules match against the main worktree root. |
| [0016](adr/0016-materialization-profiles.md) | Accepted | config, materialization | Each config layer expands at most one named materialization profile. |
| [0017](adr/0017-persistent-cache-location-and-trust.md) | Accepted | cache, workspace, sandbox | Persistent discovery hints use the user cache directory, fail open, and never broaden discovery. |
