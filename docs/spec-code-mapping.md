# Spec-to-Code Mapping

| Spec | Code directories | Test files |
|---|---|---|
| docs/specs/cli-commands.md | cmd/ww/, worktree/, git/, workspace/ | ./integration_test.go |
| docs/specs/git-operations.md | git/ | ./git/git_test.go |
| docs/specs/configuration.md | internal/config/ | ./internal/config/config_test.go |
| docs/specs/release-versioning.md | cmd/ww/version.go, cmd/ww/sub_version.go, Makefile, .goreleaser.yaml, .github/workflows/release.yml | ./cmd/ww/version_test.go, ./integration_test.go |
| docs/specs/workspace-discovery.md | workspace/, worktree/, cmd/ww/ | ./workspace/workspace_test.go |
| docs/specs/shell-integration.md | cmd/ww/, worktree/ | ./integration_test.go |
| docs/specs/interactive-mode.md | internal/interactive/, cmd/ww/ | ./internal/interactive/interactive_test.go, ./integration_test.go |
| docs/specs/testing.md | internal/testutil/ | ./integration_test.go |
| docs/specs/agentic-review-workflows.md | .github/workflows/plan-review.md, .github/workflows/impl-review.md, .github/workflows/spec-code-sync.md, .github/workflows/plan-review.lock.yml, .github/workflows/impl-review.lock.yml, .github/workflows/spec-code-sync.lock.yml | N/A |
