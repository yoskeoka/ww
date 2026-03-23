package worktree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yoskeoka/ww/git"
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

func TestWorktreePathRelativeEscapeWorkspace(t *testing.T) {
	m := &Manager{
		Config:  Config{WorktreeDir: "../outside"},
		RepoDir: "/tmp/workspace/repo",
		Workspace: &workspace.Workspace{
			Root: "/tmp/workspace",
			Mode: workspace.ModeWorkspace,
		},
	}
	_, err := m.WorktreePath("feat/my-feature")
	if err == nil {
		t.Fatal("expected error for relative worktree_dir that escapes workspace root, got nil")
	}
}

func TestWorktreePathRelativeEscapeSingleRepo(t *testing.T) {
	m := &Manager{
		Config:  Config{WorktreeDir: "../../outside"},
		RepoDir: "/tmp/project",
	}
	_, err := m.WorktreePath("feat/my-feature")
	if err == nil {
		t.Fatal("expected error for relative worktree_dir that escapes repo parent, got nil")
	}
}

func TestWorktreePathRelativeOverrideSingleRepo(t *testing.T) {
	m := &Manager{
		Config:  Config{WorktreeDir: "worktrees"},
		RepoDir: "/tmp/project",
	}
	got, err := m.WorktreePath("feat/my-feature")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/tmp", "worktrees", "project@feat-my-feature")
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

func TestResolveStatus(t *testing.T) {
	repo, runner := setupStatusRepo(t)
	_ = repo

	merged, err := runner.MergedBranches("main")
	if err != nil {
		t.Fatal(err)
	}
	mergedSet := make(map[string]struct{}, len(merged))
	for _, branch := range merged {
		mergedSet[branch] = struct{}{}
	}
	delete(mergedSet, "main")

	// Precompute branch→remote and remote branch sets.
	allBranches := []string{"feat/merged", "feat/merged-stale", "feat/stale", "feat/local"}
	branchRemote := make(map[string]string)
	remoteBranches := make(map[string]map[string]struct{})
	for _, branch := range allBranches {
		if _, ok := mergedSet[branch]; ok {
			continue
		}
		remote, err := runner.BranchRemote(branch)
		if err != nil {
			t.Fatal(err)
		}
		branchRemote[branch] = remote
		if remote != "" {
			if _, cached := remoteBranches[remote]; !cached {
				branches, err := runner.ListRemoteBranches(remote)
				if err != nil {
					t.Fatal(err)
				}
				remoteBranches[remote] = branches
			}
		}
	}

	tests := []struct {
		name  string
		entry git.WorktreeEntry
		want  string
	}{
		{
			name:  "main worktree",
			entry: git.WorktreeEntry{Branch: "main", Main: true},
			want:  StatusActive,
		},
		{
			name:  "merged branch",
			entry: git.WorktreeEntry{Branch: "feat/merged"},
			want:  StatusMerged,
		},
		{
			name:  "merged branch with deleted remote",
			entry: git.WorktreeEntry{Branch: "feat/merged-stale"},
			want:  StatusMerged,
		},
		{
			name:  "stale tracked branch",
			entry: git.WorktreeEntry{Branch: "feat/stale"},
			want:  StatusStale,
		},
		{
			name:  "local-only branch",
			entry: git.WorktreeEntry{Branch: "feat/local"},
			want:  StatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveStatus(tt.entry, mergedSet, branchRemote, remoteBranches)
			if got != tt.want {
				t.Fatalf("resolveStatus(%+v) = %q, want %q", tt.entry, got, tt.want)
			}
		})
	}
}

func setupStatusRepo(t *testing.T) (string, *git.Runner) {
	t.Helper()

	repo := t.TempDir()
	runner := &git.Runner{Dir: repo}
	mustGit(t, runner, "init", "-b", "main")
	mustGit(t, runner, "config", "user.email", "test@example.com")
	mustGit(t, runner, "config", "user.name", "Test User")
	writeStatusFile(t, repo, "README.md", "# repo\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "initial")

	remote := filepath.Join(t.TempDir(), "remote.git")
	mustGit(t, &git.Runner{Dir: repo}, "init", "--bare", remote)
	mustGit(t, runner, "remote", "add", "origin", remote)
	mustGit(t, runner, "push", "-u", "origin", "main")

	mustGit(t, runner, "checkout", "-b", "feat/merged")
	writeStatusFile(t, repo, "merged.txt", "merged\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "feat: merged")
	mustGit(t, runner, "checkout", "main")
	mustGit(t, runner, "merge", "--ff-only", "feat/merged")

	mustGit(t, runner, "checkout", "-b", "feat/merged-stale")
	writeStatusFile(t, repo, "merged-stale.txt", "merged stale\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "feat: merged stale")
	mustGit(t, runner, "push", "-u", "origin", "feat/merged-stale")
	mustGit(t, runner, "checkout", "main")
	mustGit(t, runner, "merge", "--ff-only", "feat/merged-stale")
	mustGit(t, runner, "push", "origin", ":feat/merged-stale")

	mustGit(t, runner, "checkout", "-b", "feat/stale")
	writeStatusFile(t, repo, "stale.txt", "stale\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "feat: stale")
	mustGit(t, runner, "push", "-u", "origin", "feat/stale")
	mustGit(t, runner, "checkout", "main")
	mustGit(t, runner, "push", "origin", ":feat/stale")

	mustGit(t, runner, "checkout", "-b", "feat/local")
	writeStatusFile(t, repo, "local.txt", "local\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "feat: local")
	mustGit(t, runner, "checkout", "main")

	return repo, runner
}

func mustGit(t *testing.T, runner *git.Runner, args ...string) {
	t.Helper()
	if _, err := runner.Run(args...); err != nil {
		t.Fatal(err)
	}
}

func writeStatusFile(t *testing.T, repo, name, content string) {
	t.Helper()
	path := filepath.Join(repo, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
