package worktree

import (
	"path/filepath"
	"testing"

	"github.com/yoskeoka/ww/workspace"
)

func TestSanitizeBranch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"main", "main"},
		{"feat/my-feature", "feat-my-feature"},
		{"user/name/branch", "user-name-branch"},
		{"no-slashes", "no-slashes"},
	}
	for _, tt := range tests {
		got := SanitizeBranch(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeBranch(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestWorktreePathSingleRepoDefault(t *testing.T) {
	m := &Manager{RepoDir: "/tmp/project"}
	got, err := m.WorktreePath("feat/my-feature")
	if err != nil {
		t.Fatal(err)
	}
	want := "/tmp/project@feat-my-feature"
	if got != want {
		t.Fatalf("WorktreePath = %q, want %q", got, want)
	}
}

func TestWorktreePathWorkspaceDefault(t *testing.T) {
	m := &Manager{
		RepoDir: "/tmp/workspace/repo",
		Workspace: &workspace.Workspace{
			Root: "/tmp/workspace",
			Mode: workspace.ModeWorkspace,
		},
	}
	got, err := m.WorktreePath("feat/my-feature")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/tmp/workspace", ".worktrees", "repo@feat-my-feature")
	if got != want {
		t.Fatalf("WorktreePath = %q, want %q", got, want)
	}
}

func TestWorktreePathRelativeOverrideWorkspace(t *testing.T) {
	m := &Manager{
		Config:  Config{WorktreeDir: "custom"},
		RepoDir: "/tmp/workspace/repo",
		Workspace: &workspace.Workspace{
			Root: "/tmp/workspace",
			Mode: workspace.ModeWorkspace,
		},
	}
	got, err := m.WorktreePath("feat/my-feature")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/tmp/workspace", "custom", "repo@feat-my-feature")
	if got != want {
		t.Fatalf("WorktreePath = %q, want %q", got, want)
	}
}

func TestWorktreePathAbsoluteOverride(t *testing.T) {
	m := &Manager{
		Config:  Config{WorktreeDir: "/var/tmp/worktrees"},
		RepoDir: "/tmp/workspace/repo",
		Workspace: &workspace.Workspace{
			Root: "/tmp/workspace",
			Mode: workspace.ModeWorkspace,
		},
	}
	got, err := m.WorktreePath("feat/my-feature")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/var/tmp/worktrees", "repo@feat-my-feature")
	if got != want {
		t.Fatalf("WorktreePath = %q, want %q", got, want)
	}
}
