# Specifications

Detailed specifications for `ww` CLI behavior.

| Spec | Description |
|------|-------------|
| [cli-commands.md](cli-commands.md) | All CLI commands, flags, and output formats |
| [configuration.md](configuration.md) | `.ww.toml` file format and config search |
| [git-operations.md](git-operations.md) | Low-level git operations used by `ww` |
| [workspace-discovery.md](workspace-discovery.md) | Multi-repo workspace detection algorithm |
| [cache.md](cache.md) | Persistent workspace-discovery cache contract |
| [shell-integration.md](shell-integration.md) | `ww cd`, `ww create -q`, and shell wrapper patterns |
| [interactive-mode.md](interactive-mode.md) | Human-oriented `ww i` flows for create, list, open, remove, and clean |
| [release-versioning.md](release-versioning.md) | SemVer tagging, build metadata, and release automation |
| [github-actions-pinning.md](github-actions-pinning.md) | Contract for managing ordinary workflow YAML `uses:` references with `pinact` |
| [testing.md](testing.md) | Testing strategy and test utilities |
| [workflow-linter.md](workflow-linter.md) | Workflow-linter lifecycle and completion metadata |
