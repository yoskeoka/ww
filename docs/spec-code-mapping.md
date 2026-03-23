# Spec-to-Code Mapping

| Spec | Code directories | Test files |
|---|---|---|
| docs/specs/cli-commands.md | cmd/ww/, worktree/, git/ | ./integration_test.go |
| docs/specs/git-operations.md | git/ | ./git/git_test.go |
| docs/specs/configuration.md | internal/config/ | ./internal/config/config_test.go |
| docs/specs/workspace-discovery.md | workspace/, worktree/, cmd/ww/ | ./workspace/workspace_test.go |
| docs/specs/testing.md | internal/testutil/ | ./integration_test.go |
