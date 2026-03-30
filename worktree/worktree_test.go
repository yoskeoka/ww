package worktree

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestListRepoUnknown(t *testing.T) {
	entries := []git.WorktreeEntry{
		{Path: "/repo", Branch: "main", Head: "abc1234", Main: true},
		{Path: "/repo@feat-x", Branch: "feat/x", Head: "def5678"},
		{Path: "/repo@feat-y", Branch: "feat/y", Head: "111aaaa"},
	}
	infos := listRepoUnknown(entries, "repo", "base-detect-failed")

	if len(infos) != 3 {
		t.Fatalf("expected 3 infos, got %d", len(infos))
	}
	// Main worktree should be active with no detail.
	if infos[0].Status != StatusActive {
		t.Errorf("main worktree status = %q, want %q", infos[0].Status, StatusActive)
	}
	if infos[0].StatusDetail != "" {
		t.Errorf("main worktree status_detail = %q, want empty", infos[0].StatusDetail)
	}
	// Non-main worktrees should be unknown with detail.
	for _, info := range infos[1:] {
		if info.Status != StatusUnknown {
			t.Errorf("worktree %s status = %q, want %q", info.Branch, info.Status, StatusUnknown)
		}
		if info.StatusDetail != "base-detect-failed" {
			t.Errorf("worktree %s status_detail = %q, want %q", info.Branch, info.StatusDetail, "base-detect-failed")
		}
	}
}

func TestListRepoGracefulDegradation(t *testing.T) {
	// Create a repo without a remote — no origin/HEAD, no default_base.
	repo := t.TempDir()
	runner := &git.Runner{Dir: repo}
	mustGit(t, runner, "init", "-b", "main")
	mustGit(t, runner, "config", "user.email", "test@example.com")
	mustGit(t, runner, "config", "user.name", "Test User")
	writeStatusFile(t, repo, "README.md", "# repo\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "initial")

	mustGit(t, runner, "checkout", "-b", "feat/local")
	writeStatusFile(t, repo, "local.txt", "local\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "feat: local")
	mustGit(t, runner, "checkout", "main")

	mgr := &Manager{
		Git:     runner,
		Config:  Config{}, // No DefaultBase
		RepoDir: repo,
	}

	// List should succeed (not error) with unknown status.
	infos, err := mgr.List()
	if err != nil {
		t.Fatalf("List() should not fail when base is unresolvable, got: %v", err)
	}
	if len(infos) == 0 {
		t.Fatal("expected at least one worktree info")
	}

	var mainFound, unknownFound bool
	for _, info := range infos {
		if info.Main {
			mainFound = true
			if info.Status != StatusActive {
				t.Errorf("main worktree status = %q, want %q", info.Status, StatusActive)
			}
		} else {
			unknownFound = true
			if info.Status != StatusUnknown {
				t.Errorf("worktree %s status = %q, want %q", info.Branch, info.Status, StatusUnknown)
			}
			if info.StatusDetail != "base-detect-failed" {
				t.Errorf("worktree %s status_detail = %q, want %q", info.Branch, info.StatusDetail, "base-detect-failed")
			}
		}
	}
	if !mainFound {
		t.Error("main worktree not found in list output")
	}
	// feat/local has no worktree (it's just a branch), so only main should be listed.
	// This test verifies the graceful degradation path, not worktree creation.
	_ = unknownFound
}

func TestFindByName(t *testing.T) {
	repo := t.TempDir()
	runner := &git.Runner{Dir: repo}
	mustGit(t, runner, "init", "-b", "main")
	mustGit(t, runner, "config", "user.email", "test@example.com")
	mustGit(t, runner, "config", "user.name", "Test User")
	writeStatusFile(t, repo, "README.md", "# repo\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "initial")

	wtPath := filepath.Join(filepath.Dir(repo), "repo@feat-alpha")
	mustGit(t, runner, "worktree", "add", "-b", "feat/alpha", wtPath, "main")

	mgr := &Manager{
		Git:     runner,
		Config:  Config{DefaultBase: "main"},
		RepoDir: repo,
	}

	info, err := mgr.FindByName("refs/heads/feat/alpha", false)
	if err != nil {
		t.Fatal(err)
	}
	if info.Branch != "feat/alpha" {
		t.Fatalf("FindByName returned branch %q, want %q", info.Branch, "feat/alpha")
	}
	if info.Path != wtPath {
		t.Fatalf("FindByName returned path %q, want %q", info.Path, wtPath)
	}
}

func TestMostRecentUsesWorktreeAdminMtime(t *testing.T) {
	repo := t.TempDir()
	runner := &git.Runner{Dir: repo}
	mustGit(t, runner, "init", "-b", "main")
	mustGit(t, runner, "config", "user.email", "test@example.com")
	mustGit(t, runner, "config", "user.name", "Test User")
	writeStatusFile(t, repo, "README.md", "# repo\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "initial")

	alphaPath := filepath.Join(filepath.Dir(repo), "repo@feat-alpha")
	betaPath := filepath.Join(filepath.Dir(repo), "repo@feat-beta")
	mustGit(t, runner, "worktree", "add", "-b", "feat/alpha", alphaPath, "main")
	mustGit(t, runner, "worktree", "add", "-b", "feat/beta", betaPath, "main")

	adminRoot := filepath.Join(repo, ".git", "worktrees")
	setAdminMtime(t, adminRoot, alphaPath, time.Unix(100, 0))
	setAdminMtime(t, adminRoot, betaPath, time.Unix(200, 0))

	mgr := &Manager{
		Git:     runner,
		Config:  Config{DefaultBase: "main"},
		RepoDir: repo,
	}

	info, err := mgr.MostRecent(false)
	if err != nil {
		t.Fatal(err)
	}
	if info.Path != betaPath {
		t.Fatalf("MostRecent returned path %q, want %q", info.Path, betaPath)
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

func setAdminMtime(t *testing.T, adminRoot, wantWorktreePath string, modTime time.Time) {
	t.Helper()

	entries, err := os.ReadDir(adminRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		gotPath, err := worktreePathFromAdminDir(adminRoot, entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if gotPath != wantWorktreePath {
			continue
		}
		adminDir := filepath.Join(adminRoot, entry.Name())
		if err := os.Chtimes(adminDir, modTime, modTime); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Fatalf("could not find admin dir for %s", wantWorktreePath)
}
