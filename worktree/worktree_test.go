package worktree

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
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

func TestWorktreePathSandboxSingleRepoDefault(t *testing.T) {
	m := &Manager{
		Config:  Config{Sandbox: true},
		RepoDir: "/tmp/project",
	}
	got, err := m.WorktreePath("feat/my-feature")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/tmp/project", ".worktrees", "project@feat-my-feature")
	if got != want {
		t.Fatalf("WorktreePath = %q, want %q", got, want)
	}
}

func TestListWorkspaceReposBoundsConcurrencyAndPreservesOrder(t *testing.T) {
	repos := make([]workspace.Repo, 6)
	for i := range repos {
		repos[i] = workspace.Repo{Name: fmt.Sprintf("repo%d", i), Path: fmt.Sprintf("/tmp/repo%d", i)}
	}

	var active atomic.Int32
	var maximum atomic.Int32
	started := make(chan string, workspaceListWorkers)
	release := make(chan struct{})
	type result struct {
		infos []WorktreeInfo
		err   error
	}
	done := make(chan result, 1)
	go func() {
		infos, err := listWorkspaceRepos(repos, func(name, _ string) ([]WorktreeInfo, error) {
			running := active.Add(1)
			for {
				previous := maximum.Load()
				if running <= previous || maximum.CompareAndSwap(previous, running) {
					break
				}
			}
			started <- name
			<-release
			active.Add(-1)
			return []WorktreeInfo{{Repo: name}}, nil
		})
		done <- result{infos: infos, err: err}
	}()

	for range workspaceListWorkers {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("workspace repository enumeration did not start four workers")
		}
	}
	if got := maximum.Load(); got != workspaceListWorkers {
		t.Fatalf("maximum concurrent repository enumerations = %d, want %d", got, workspaceListWorkers)
	}

	close(release)
	got := <-done
	if got.err != nil {
		t.Fatal(got.err)
	}
	if len(got.infos) != len(repos) {
		t.Fatalf("listed %d worktrees, want %d", len(got.infos), len(repos))
	}
	for i, info := range got.infos {
		if want := repos[i].Name; info.Repo != want {
			t.Fatalf("result %d repo = %q, want %q", i, info.Repo, want)
		}
	}
}

func TestListWorkspaceReposReturnsEarliestErrorWithoutPartialResults(t *testing.T) {
	first := errors.New("first repository failed")
	later := errors.New("later repository failed")
	repos := []workspace.Repo{
		{Name: "repo0", Path: "/tmp/repo0"},
		{Name: "repo1", Path: "/tmp/repo1"},
		{Name: "repo2", Path: "/tmp/repo2"},
		{Name: "repo3", Path: "/tmp/repo3"},
	}

	infos, err := listWorkspaceRepos(repos, func(name, _ string) ([]WorktreeInfo, error) {
		switch name {
		case "repo1":
			return nil, first
		case "repo3":
			return nil, later
		default:
			return []WorktreeInfo{{Repo: name}}, nil
		}
	})
	if !errors.Is(err, first) {
		t.Fatalf("listWorkspaceRepos() error = %v, want earliest error %v", err, first)
	}
	if infos != nil {
		t.Fatalf("listWorkspaceRepos() infos = %#v, want no partial results", infos)
	}
}

func TestListWorkspaceReposEmpty(t *testing.T) {
	called := false
	infos, err := listWorkspaceRepos(nil, func(string, string) ([]WorktreeInfo, error) {
		called = true
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("empty workspace should not enumerate repositories")
	}
	if infos != nil {
		t.Fatalf("empty workspace infos = %#v, want nil", infos)
	}
}

func TestListSingleRepoDoesNotUseWorkspaceScheduler(t *testing.T) {
	var calls int
	mgr := &Manager{
		RepoDir:   "/tmp/single",
		Workspace: &workspace.Workspace{Mode: workspace.ModeSingleRepo},
		listRepoFn: func(name, repoPath string) ([]WorktreeInfo, error) {
			calls++
			if name != "single" || repoPath != "/tmp/single" {
				t.Fatalf("single-repo list target = (%q, %q)", name, repoPath)
			}
			return []WorktreeInfo{{Repo: name}}, nil
		},
	}

	infos, err := mgr.List()
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(infos) != 1 || infos[0].Repo != "single" {
		t.Fatalf("single-repo List() = %#v after %d calls, want one direct repository result", infos, calls)
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

func TestWorktreePathRelativeOverrideSandboxSingleRepo(t *testing.T) {
	m := &Manager{
		Config:  Config{WorktreeDir: "worktrees", Sandbox: true},
		RepoDir: "/tmp/project",
	}
	got, err := m.WorktreePath("feat/my-feature")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/tmp/project", "worktrees", "project@feat-my-feature")
	if got != want {
		t.Fatalf("WorktreePath = %q, want %q", got, want)
	}
}

func TestWorktreePathRelativeEscapeSandboxSingleRepo(t *testing.T) {
	m := &Manager{
		Config:  Config{WorktreeDir: "../outside", Sandbox: true},
		RepoDir: "/tmp/project",
	}
	_, err := m.WorktreePath("feat/my-feature")
	if err == nil {
		t.Fatal("expected error for relative worktree_dir that escapes repo root, got nil")
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
	allBranches := []string{
		"feat/merged",
		"feat/merged-stale",
		"feat/squash-merged",
		"feat/rebased",
		"feat/stale",
		"feat/local",
		"feat/partial",
	}
	patchMerged, err := runner.PatchEquivalentBranches("main", allBranches)
	if err != nil {
		t.Fatal(err)
	}
	for _, branch := range patchMerged {
		mergedSet[branch] = struct{}{}
	}

	// Precompute branch→remote and remote branch sets.
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
			name:  "squash merged branch",
			entry: git.WorktreeEntry{Branch: "feat/squash-merged"},
			want:  StatusMerged,
		},
		{
			name:  "rebased branch",
			entry: git.WorktreeEntry{Branch: "feat/rebased"},
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
		{
			name:  "partially integrated local branch",
			entry: git.WorktreeEntry{Branch: "feat/partial"},
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

func TestBaseBranchNames(t *testing.T) {
	got := baseBranchNames("origin/main")
	want := []string{"origin/main", "main"}
	if len(got) != len(want) {
		t.Fatalf("baseBranchNames(origin/main) len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("baseBranchNames(origin/main)[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	got = baseBranchNames("main")
	want = []string{"main"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("baseBranchNames(main) = %v, want %v", got, want)
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

	// Create a secondary worktree so we can verify it gets unknown status.
	wtPath := filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"@feat-local")
	mustGit(t, runner, "worktree", "add", "-b", "feat/local", wtPath, "main")

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
	if !unknownFound {
		t.Error("expected at least one worktree with unknown status")
	}
}

func TestCreateUnresolvedBaseErrorIsActionable(t *testing.T) {
	repo := t.TempDir()
	runner := &git.Runner{Dir: repo}
	mustGit(t, runner, "init", "-b", "main")
	mustGit(t, runner, "config", "user.email", "test@example.com")
	mustGit(t, runner, "config", "user.name", "Test User")
	writeStatusFile(t, repo, "README.md", "# repo\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "initial")

	mgr := &Manager{
		Git:     runner,
		Config:  Config{},
		RepoDir: repo,
	}

	_, _, err := mgr.Create("feat/no-base", CreateOpts{DryRun: true})
	if err == nil {
		t.Fatal("Create() error = nil, want unresolved base error")
	}
	msg := err.Error()
	for _, want := range []string{
		"cannot determine base branch",
		"no default_base is configured",
		"origin/HEAD could not be used",
		"heuristic fallback could not find a usable origin/main or origin/master",
		"Set default_base in .ww.toml",
		"git remote set-head origin --auto",
		"Original error:",
		"git symbolic-ref refs/remotes/origin/HEAD",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Create() diagnostic missing %q:\n%s", want, msg)
		}
	}
}

func TestCreateExistingBranchDoesNotRequireBase(t *testing.T) {
	repo := t.TempDir()
	runner := &git.Runner{Dir: repo}
	mustGit(t, runner, "init", "-b", "main")
	mustGit(t, runner, "config", "user.email", "test@example.com")
	mustGit(t, runner, "config", "user.name", "Test User")
	writeStatusFile(t, repo, "README.md", "# repo\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "initial")
	mustGit(t, runner, "branch", "feat/existing")

	mgr := &Manager{
		Git:     runner,
		Config:  Config{},
		RepoDir: repo,
	}

	info, log, err := mgr.Create("feat/existing", CreateOpts{DryRun: true})
	if err != nil {
		t.Fatalf("Create() existing branch should not require base, got: %v", err)
	}
	if info.Base != "" {
		t.Fatalf("Create() existing branch Base = %q, want empty", info.Base)
	}
	if len(log) == 0 || !strings.Contains(log[0], "existing branch: feat/existing") {
		t.Fatalf("Create() dry-run log = %#v, want existing branch message", log)
	}
}

func TestCreateGuessRemoteExistingLocalBranchTakesPrecedence(t *testing.T) {
	repo := setupGitRepo(t)
	runner := &git.Runner{Dir: repo}
	mustGit(t, runner, "branch", "feat/existing")

	mgr := &Manager{
		Git:     runner,
		Config:  Config{},
		RepoDir: repo,
	}

	info, log, err := mgr.Create("feat/existing", CreateOpts{DryRun: true, GuessRemote: true})
	if err != nil {
		t.Fatalf("Create() existing branch with GuessRemote should not fail, got: %v", err)
	}
	if info.Base != "" {
		t.Fatalf("Create() existing branch Base = %q, want empty", info.Base)
	}
	if len(log) == 0 || !strings.Contains(log[0], "existing branch: feat/existing") {
		t.Fatalf("Create() dry-run log = %#v, want existing branch message", log)
	}
}

func TestCreateGuessRemoteMissingRemoteBranchIsActionable(t *testing.T) {
	repo, _ := setupGitRepoWithRemote(t)
	runner := &git.Runner{Dir: repo}

	mgr := &Manager{
		Git:     runner,
		Config:  Config{},
		RepoDir: repo,
	}

	_, _, err := mgr.Create("feat/missing", CreateOpts{GuessRemote: true})
	if err == nil {
		t.Fatal("Create() error = nil, want guess-remote resolution failure")
	}
	msg := err.Error()
	for _, want := range []string{
		`cannot resolve remote branch "feat/missing" with --guess-remote after refreshing origin`,
		"Make sure a matching remote branch exists and can be resolved by Git",
		"Original error:",
		"invalid reference: feat/missing",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Create() diagnostic missing %q:\n%s", want, msg)
		}
	}
}

func TestCreateGuessRemoteUnsupportedGitIsActionable(t *testing.T) {
	repo, _ := setupGitRepoWithRemote(t)
	baseRunner := &git.Runner{Dir: repo}

	mustGit(t, baseRunner, "checkout", "-b", "feat/remote-only")
	writeStatusFile(t, repo, "remote-only.txt", "remote only\n")
	mustGit(t, baseRunner, "add", ".")
	mustGit(t, baseRunner, "commit", "-m", "feat: remote only")
	mustGit(t, baseRunner, "push", "-u", "origin", "feat/remote-only")
	mustGit(t, baseRunner, "checkout", "main")
	mustGit(t, baseRunner, "branch", "-D", "feat/remote-only")

	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(t.TempDir(), "fake-git.sh")
	scriptBody := "#!/bin/sh\n" +
		"if [ \"$1\" = \"worktree\" ] && [ \"$2\" = \"add\" ] && [ \"$3\" = \"--guess-remote\" ]; then\n" +
		"  echo \"error: unknown option guess-remote\" 1>&2\n" +
		"  exit 129\n" +
		"fi\n" +
		"exec \"" + realGit + "\" \"$@\"\n"
	if err := os.WriteFile(script, []byte(scriptBody), 0755); err != nil {
		t.Fatal(err)
	}

	mgr := &Manager{
		Git:     &git.Runner{Dir: repo, GitBin: script},
		Config:  Config{},
		RepoDir: repo,
	}

	_, _, err = mgr.Create("feat/remote-only", CreateOpts{GuessRemote: true})
	if err == nil {
		t.Fatal("Create() error = nil, want unsupported guess-remote diagnostic")
	}
	msg := err.Error()
	for _, want := range []string{
		"git worktree add --guess-remote is unsupported by the installed Git",
		"Upgrade Git and retry",
		"git worktree add -b 'feat/remote-only' --track",
		"'origin/feat/remote-only'",
		"Original error:",
		"unknown option guess-remote",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Create() diagnostic missing %q:\n%s", want, msg)
		}
	}
}

func TestCreateSandboxSingleRepoUsesRepoLocalWorktrees(t *testing.T) {
	repo, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runner := &git.Runner{Dir: repo}
	mustGit(t, runner, "init", "-b", "main")
	mustGit(t, runner, "config", "user.email", "test@example.com")
	mustGit(t, runner, "config", "user.name", "Test User")
	writeStatusFile(t, repo, "README.md", "# repo\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "initial")

	mgr := &Manager{
		Git:     runner,
		Config:  Config{DefaultBase: "main", Sandbox: true},
		RepoDir: repo,
	}

	info, _, err := mgr.Create("feat/sandbox", CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(repo, ".worktrees", filepath.Base(repo)+"@feat-sandbox")
	if info.Path != want {
		t.Fatalf("Create path = %q, want %q", info.Path, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("created worktree not found at %s: %v", want, err)
	}
}

func TestCreateNewBranchDoesNotTrackBaseRemoteBranch(t *testing.T) {
	repo, _ := setupGitRepoWithRemote(t)
	runner := &git.Runner{Dir: repo}

	mgr := &Manager{
		Git:     runner,
		Config:  Config{},
		RepoDir: repo,
	}
	baseInfo, err := mgr.baseRef(runner)
	if err != nil {
		t.Fatal(err)
	}
	if baseInfo.Ref != "origin/main" {
		t.Fatalf("baseRef().Ref = %q, want %q", baseInfo.Ref, "origin/main")
	}

	info, _, err := mgr.Create("feat/no-upstream", CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	tracking, err := runner.BranchTrackingConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg, ok := tracking["feat/no-upstream"]; ok && (cfg.Remote != "" || cfg.MergeRef != "") {
		t.Fatalf("tracking[feat/no-upstream] = %+v, want no upstream after creating %s", cfg, info.Path)
	}
}

func TestCreateAndRemoveRunLifecycleHooksWithContext(t *testing.T) {
	repo := setupGitRepo(t)
	runner := &git.Runner{Dir: repo}
	logPath := filepath.Join(t.TempDir(), "lifecycle-hooks.log")

	mgr := &Manager{
		Git: runner,
		Config: Config{
			DefaultBase:    "main",
			PreCreateHook:  hookAppendCommand(logPath, "pre-create", 7),
			PostCreateHook: hookAppendCommand(logPath, "post-create", 0),
			PreRemoveHook:  hookAppendCommand(logPath, "pre-remove", 7),
			PostRemoveHook: hookAppendCommand(logPath, "post-remove", 0),
		},
		RepoDir: repo,
	}

	createStderr := captureStderr(t, func() error {
		_, _, err := mgr.Create("feat/lifecycle-hooks", CreateOpts{})
		return err
	})
	if !strings.Contains(createStderr, "warning: pre-create hook failed:") {
		t.Fatalf("Create stderr missing pre-create warning:\n%s", createStderr)
	}

	info, err := mgr.WorktreePath("feat/lifecycle-hooks")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(info); err != nil {
		t.Fatalf("created worktree missing at %s: %v", info, err)
	}

	removeStderr := captureStderr(t, func() error {
		_, _, err := mgr.Remove("feat/lifecycle-hooks", RemoveOpts{})
		return err
	})
	if !strings.Contains(removeStderr, "warning: pre-remove hook failed:") {
		t.Fatalf("Remove stderr missing pre-remove warning:\n%s", removeStderr)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(logData)), "\n")
	if len(lines) != 4 {
		t.Fatalf("lifecycle log = %q, want 4 lines", string(logData))
	}

	repoName := filepath.Base(repo)
	wantPath := map[string]string{
		"pre-create":  repo,
		"post-create": info,
		"pre-remove":  info,
		"post-remove": repo,
	}
	for i, wantPhase := range []string{"pre-create", "post-create", "pre-remove", "post-remove"} {
		fields := strings.Split(lines[i], "|")
		if len(fields) != 5 {
			t.Fatalf("log line %d = %q, want 5 fields", i, lines[i])
		}
		if fields[0] != wantPhase {
			t.Fatalf("log line %d phase = %q, want %q", i, fields[0], wantPhase)
		}
		if fields[1] != wantPath[wantPhase] {
			t.Fatalf("log line %d pwd = %q, want %q", i, fields[1], wantPath[wantPhase])
		}
		if fields[2] != "feat/lifecycle-hooks" {
			t.Fatalf("log line %d branch = %q, want feat/lifecycle-hooks", i, fields[2])
		}
		if fields[3] != repoName {
			t.Fatalf("log line %d repo = %q, want %q", i, fields[3], repoName)
		}
		if fields[4] != "2" {
			t.Fatalf("log line %d index = %q, want 2", i, fields[4])
		}
	}
}

func TestCreateSandboxRelativeEscapeRejected(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	runner := &git.Runner{Dir: repo}
	mustGit(t, runner, "init", "-b", "main")
	mustGit(t, runner, "config", "user.email", "test@example.com")
	mustGit(t, runner, "config", "user.name", "Test User")
	writeStatusFile(t, repo, "README.md", "# repo\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "initial")

	mgr := &Manager{
		Git:     runner,
		Config:  Config{DefaultBase: "main", WorktreeDir: "../outside", Sandbox: true},
		RepoDir: repo,
	}

	_, _, err := mgr.Create("feat/escape", CreateOpts{})
	if err == nil {
		t.Fatal("Create error = nil, want relative worktree_dir escape error")
	}
	if !strings.Contains(err.Error(), "resolves outside the allowed area") {
		t.Fatalf("Create error = %q, want escape diagnostic", err)
	}
}

func TestUnresolvedCreateBaseErrorDistinguishesHeuristicFailure(t *testing.T) {
	originErr := errors.New("origin head failed")
	heuristicErr := errors.New("ls-remote failed")
	err := unresolvedCreateBaseError(baseDetectionError{
		originHeadErr: originErr,
		heuristicErr:  heuristicErr,
	})
	msg := err.Error()

	for _, want := range []string{
		"heuristic fallback failed before it could choose origin/main or origin/master",
		"Set default_base in .ww.toml",
		"git remote set-head origin --auto",
		"Original error:",
		"ls-remote failed",
		"origin head failed",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("diagnostic missing %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "heuristic fallback could not find a usable origin/main or origin/master") {
		t.Fatalf("diagnostic should not describe heuristic execution failure as candidate miss:\n%s", msg)
	}
	if !errors.Is(err, originErr) {
		t.Fatal("diagnostic should preserve origin/HEAD error cause")
	}
	if !errors.Is(err, heuristicErr) {
		t.Fatal("diagnostic should preserve heuristic error cause")
	}
}

func TestSubmoduleWorktreeRemoveError(t *testing.T) {
	cause := errors.New("git worktree remove --force /tmp/repo@feat: exit status 128\nfatal: working trees containing submodules cannot be moved or removed")
	err := submoduleWorktreeRemoveError("/tmp/repo@feat", "/tmp/repo", cause)
	msg := err.Error()

	for _, want := range []string{
		"Git cannot remove worktrees containing submodules",
		"Target worktree: /tmp/repo@feat",
		"Manual cleanup permanently deletes uncommitted work",
		`rm -rf -- '/tmp/repo@feat'`,
		`git -C '/tmp/repo' worktree prune`,
		"Original error:",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("diagnostic missing %q:\n%s", want, msg)
		}
	}
	if !errors.Is(err, cause) {
		t.Fatal("diagnostic should preserve the original error cause")
	}
}

func TestShellQuotePOSIX(t *testing.T) {
	got := shellQuotePOSIX(`/tmp/repo's $branch`)
	want := `'/tmp/repo'"'"'s $branch'`
	if got != want {
		t.Fatalf("shellQuotePOSIX() = %q, want %q", got, want)
	}
}

func TestBaseRefUsesHeuristicWhenOriginHeadMissing(t *testing.T) {
	repo, runner := setupStatusRepo(t)

	mgr := &Manager{
		Git:     runner,
		Config:  Config{},
		RepoDir: repo,
	}

	base, err := mgr.baseRef(runner)
	if err != nil {
		t.Fatal(err)
	}
	if base.Ref != "origin/main" {
		t.Fatalf("baseRef().Ref = %q, want %q", base.Ref, "origin/main")
	}
	if base.StatusDetail != "heuristic-base" {
		t.Fatalf("baseRef().StatusDetail = %q, want %q", base.StatusDetail, "heuristic-base")
	}
}

func TestListRepoHeuristicStatusDetail(t *testing.T) {
	repo, runner := setupStatusRepo(t)

	worktrees := map[string]string{
		"feat/merged":        filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"@feat-merged"),
		"feat/merged-stale":  filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"@feat-merged-stale"),
		"feat/squash-merged": filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"@feat-squash-merged"),
		"feat/rebased":       filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"@feat-rebased"),
		"feat/stale":         filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"@feat-stale"),
		"feat/local":         filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"@feat-local"),
		"feat/partial":       filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"@feat-partial"),
	}
	for branch, wtPath := range worktrees {
		mustGit(t, runner, "worktree", "add", wtPath, branch)
	}

	mgr := &Manager{
		Git:     runner,
		Config:  Config{},
		RepoDir: repo,
	}

	infos, err := mgr.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) == 0 {
		t.Fatal("expected worktree infos, got none")
	}

	gotStatus := make(map[string]string)
	for _, info := range infos {
		gotStatus[info.Branch] = info.Status
		if info.StatusDetail != "heuristic-base" {
			t.Fatalf("worktree %q status_detail = %q, want %q", info.Branch, info.StatusDetail, "heuristic-base")
		}
	}

	if gotStatus["main"] != StatusActive {
		t.Fatalf("main status = %q, want %q", gotStatus["main"], StatusActive)
	}
	if gotStatus["feat/merged"] != StatusMerged {
		t.Fatalf("feat/merged status = %q, want %q", gotStatus["feat/merged"], StatusMerged)
	}
	if gotStatus["feat/merged-stale"] != StatusMerged {
		t.Fatalf("feat/merged-stale status = %q, want %q", gotStatus["feat/merged-stale"], StatusMerged)
	}
	if gotStatus["feat/squash-merged"] != StatusMerged {
		t.Fatalf("feat/squash-merged status = %q, want %q", gotStatus["feat/squash-merged"], StatusMerged)
	}
	if gotStatus["feat/rebased"] != StatusMerged {
		t.Fatalf("feat/rebased status = %q, want %q", gotStatus["feat/rebased"], StatusMerged)
	}
	if gotStatus["feat/stale"] != StatusStale {
		t.Fatalf("feat/stale status = %q, want %q", gotStatus["feat/stale"], StatusStale)
	}
	if gotStatus["feat/local"] != StatusActive {
		t.Fatalf("feat/local status = %q, want %q", gotStatus["feat/local"], StatusActive)
	}
	if gotStatus["feat/partial"] != StatusActive {
		t.Fatalf("feat/partial status = %q, want %q", gotStatus["feat/partial"], StatusActive)
	}
}

func TestFindByName(t *testing.T) {
	repo, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
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
	repo, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
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
	mustGit(t, runner, "push", "origin", "main")

	mustGit(t, runner, "checkout", "-b", "feat/merged-stale")
	writeStatusFile(t, repo, "merged-stale.txt", "merged stale\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "feat: merged stale")
	mustGit(t, runner, "push", "-u", "origin", "feat/merged-stale")
	mustGit(t, runner, "checkout", "main")
	mustGit(t, runner, "merge", "--ff-only", "feat/merged-stale")
	mustGit(t, runner, "push", "origin", "main")
	mustGit(t, runner, "push", "origin", ":feat/merged-stale")

	mustGit(t, runner, "checkout", "-b", "feat/squash-merged")
	writeStatusFile(t, repo, "squash-merged-1.txt", "squash merged one\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "feat: squash merged source 1")
	writeStatusFile(t, repo, "squash-merged-2.txt", "squash merged two\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "feat: squash merged source 2")
	mustGit(t, runner, "checkout", "main")
	mustGit(t, runner, "-c", "merge.ff=true", "merge", "--squash", "feat/squash-merged")
	mustGit(t, runner, "commit", "-m", "feat: squash merged")
	mustGit(t, runner, "push", "origin", "main")

	mustGit(t, runner, "checkout", "-b", "feat/rebased")
	writeStatusFile(t, repo, "rebased.txt", "rebased\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "feat: rebased source")
	mustGit(t, runner, "checkout", "main")
	writeStatusFile(t, repo, "rebase-base.txt", "base shift\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "chore: rebase base shift")
	mustGit(t, runner, "cherry-pick", "feat/rebased")
	mustGit(t, runner, "push", "origin", "main")

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

	mustGit(t, runner, "checkout", "-b", "feat/partial")
	writeStatusFile(t, repo, "partial.txt", "partial integrated\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "feat: partial integrated")
	mustGit(t, runner, "checkout", "main")
	mustGit(t, runner, "cherry-pick", "feat/partial")
	mustGit(t, runner, "push", "origin", "main")
	mustGit(t, runner, "checkout", "feat/partial")
	writeStatusFile(t, repo, "partial-extra.txt", "partial extra\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "feat: partial extra")
	mustGit(t, runner, "checkout", "main")

	return repo, runner
}

func mustGit(t *testing.T, runner *git.Runner, args ...string) {
	t.Helper()
	if _, err := runner.Run(args...); err != nil {
		t.Fatal(err)
	}
}

func setupGitRepo(t *testing.T) string {
	t.Helper()

	repo := t.TempDir()
	runner := &git.Runner{Dir: repo}
	mustGit(t, runner, "init", "-b", "main")
	mustGit(t, runner, "config", "user.email", "test@example.com")
	mustGit(t, runner, "config", "user.name", "Test User")
	writeStatusFile(t, repo, "README.md", "# repo\n")
	mustGit(t, runner, "add", ".")
	mustGit(t, runner, "commit", "-m", "initial")
	return repo
}

func setupGitRepoWithRemote(t *testing.T) (string, string) {
	t.Helper()

	remote := filepath.Join(t.TempDir(), "remote.git")
	mustGit(t, &git.Runner{Dir: t.TempDir()}, "init", "--bare", remote)

	repo := setupGitRepo(t)
	runner := &git.Runner{Dir: repo}
	mustGit(t, runner, "remote", "add", "origin", remote)
	mustGit(t, runner, "push", "-u", "origin", "main")
	return repo, remote
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

func captureStderr(t *testing.T, fn func() error) string {
	t.Helper()

	oldStderr := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = writer

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, reader)
		close(done)
	}()

	runErr := fn()
	_ = writer.Close()
	os.Stderr = oldStderr
	<-done

	if runErr != nil {
		t.Fatalf("captured operation failed: %v\nstderr:\n%s", runErr, buf.String())
	}
	return buf.String()
}

func hookAppendCommand(logPath, phase string, exitCode int) string {
	command := fmt.Sprintf(`printf '%%s|%%s|%%s|%%s|%%s\n' %s "$PWD" "$WW_BRANCH" "$WW_REPO_NAME" "$WW_WORKTREE_INDEX" >> %s; exit %d`,
		shellQuotePOSIX(phase), shellQuotePOSIX(logPath), exitCode,
	)
	return command
}
