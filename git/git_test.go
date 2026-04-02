package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseWorktreeList(t *testing.T) {
	input := `worktree /home/user/myrepo
HEAD abc1234def5678901234567890123456789012
branch refs/heads/main

worktree /home/user/myrepo@feat-auth
HEAD def5678abc1234901234567890123456789012
branch refs/heads/feat/auth

`
	entries := parseWorktreeList(input)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Path != "/home/user/myrepo" {
		t.Errorf("entry[0].Path = %q, want /home/user/myrepo", entries[0].Path)
	}
	if entries[0].Branch != "main" {
		t.Errorf("entry[0].Branch = %q, want main", entries[0].Branch)
	}
	if entries[0].Head != "abc1234" {
		t.Errorf("entry[0].Head = %q, want abc1234", entries[0].Head)
	}
	if !entries[0].Main {
		t.Error("entry[0].Main should be true (first entry is main worktree)")
	}

	if entries[1].Branch != "feat/auth" {
		t.Errorf("entry[1].Branch = %q, want feat/auth", entries[1].Branch)
	}
	if entries[1].Main {
		t.Error("entry[1].Main should be false")
	}
}

func TestParseWorktreeListBare(t *testing.T) {
	input := `worktree /home/user/myrepo.git
bare

`
	entries := parseWorktreeList(input)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !entries[0].Bare {
		t.Error("expected bare = true")
	}
}

func TestMergedBranches(t *testing.T) {
	repo := setupGitRepo(t)
	runner := &Runner{Dir: repo}

	if _, err := runner.Run("checkout", "-b", "feat/merged"); err != nil {
		t.Fatal(err)
	}
	writeGitFile(t, repo, "merged.txt", "merged\n")
	if _, err := runner.Run("add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("commit", "-m", "feat: merged branch"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("checkout", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("merge", "--ff-only", "feat/merged"); err != nil {
		t.Fatal(err)
	}

	branches, err := runner.MergedBranches("main")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(branches, "feat/merged") {
		t.Fatalf("MergedBranches did not include feat/merged: %v", branches)
	}
}

// TestMergedBranchesWorktreePrefix verifies that MergedBranches strips the "+"
// prefix that git uses when a branch is currently checked out in another worktree.
func TestMergedBranchesWorktreePrefix(t *testing.T) {
	repo := setupGitRepo(t)
	runner := &Runner{Dir: repo}

	// Create and merge feat/wt-merged
	if _, err := runner.Run("checkout", "-b", "feat/wt-merged"); err != nil {
		t.Fatal(err)
	}
	writeGitFile(t, repo, "wt-merged.txt", "wt-merged\n")
	if _, err := runner.Run("add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("commit", "-m", "feat: wt-merged"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("checkout", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("merge", "--ff-only", "feat/wt-merged"); err != nil {
		t.Fatal(err)
	}

	// Create a sibling worktree that checks out feat/wt-merged, causing git to
	// display it as "+ feat/wt-merged" in branch listings.
	sibling := t.TempDir()
	if _, err := runner.Run("worktree", "add", sibling, "feat/wt-merged"); err != nil {
		t.Fatal(err)
	}

	branches, err := runner.MergedBranches("main")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(branches, "feat/wt-merged") {
		t.Fatalf("MergedBranches did not include feat/wt-merged (got %v); '+' prefix may not be stripped", branches)
	}
}

func TestBranchRemote(t *testing.T) {
	repo, remote := setupGitRepoWithRemote(t)
	runner := &Runner{Dir: repo}

	if _, err := runner.Run("checkout", "-b", "feat/pushed"); err != nil {
		t.Fatal(err)
	}
	writeGitFile(t, repo, "pushed.txt", "pushed\n")
	if _, err := runner.Run("add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("commit", "-m", "feat: pushed branch"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("push", "-u", "origin", "feat/pushed"); err != nil {
		t.Fatal(err)
	}

	remoteName, err := runner.BranchRemote("feat/pushed")
	if err != nil {
		t.Fatal(err)
	}
	if remoteName != "origin" {
		t.Fatalf("BranchRemote(feat/pushed) = %q, want origin", remoteName)
	}

	exists, err := runner.RemoteBranchExists("origin", "feat/pushed")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected origin/feat/pushed to exist in %s", remote)
	}

	if _, err := runner.Run("push", "origin", ":feat/pushed"); err != nil {
		t.Fatal(err)
	}
	exists, err = runner.RemoteBranchExists("origin", "feat/pushed")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("expected origin/feat/pushed to be deleted")
	}
}

func TestBranchRemoteMissingTracking(t *testing.T) {
	repo := setupGitRepo(t)
	runner := &Runner{Dir: repo}

	remoteName, err := runner.BranchRemote("main")
	if err != nil {
		t.Fatal(err)
	}
	if remoteName != "" {
		t.Fatalf("BranchRemote(main) = %q, want empty", remoteName)
	}
}

func TestRepoPathHelpersStandaloneRepo(t *testing.T) {
	repo := setupGitRepo(t)
	runner := &Runner{Dir: repo}

	topLevel, err := runner.TopLevelDir()
	if err != nil {
		t.Fatal(err)
	}
	if topLevel != repo {
		t.Fatalf("TopLevelDir() = %q, want %q", topLevel, repo)
	}

	gitDir, err := runner.GitDir()
	if err != nil {
		t.Fatal(err)
	}
	if gitDir != filepath.Join(repo, ".git") {
		t.Fatalf("GitDir() = %q, want %q", gitDir, filepath.Join(repo, ".git"))
	}

	gitCommonDir, err := runner.GitCommonDir()
	if err != nil {
		t.Fatal(err)
	}
	if gitCommonDir != filepath.Join(repo, ".git") {
		t.Fatalf("GitCommonDir() = %q, want %q", gitCommonDir, filepath.Join(repo, ".git"))
	}
}

func TestRepoPathHelpersLinkedWorktree(t *testing.T) {
	repo := setupGitRepo(t)
	runner := &Runner{Dir: repo}

	if _, err := runner.Run("branch", "feat/worktree-paths"); err != nil {
		t.Fatal(err)
	}
	worktree := filepath.Join(t.TempDir(), "wt")
	if _, err := runner.Run("worktree", "add", worktree, "feat/worktree-paths"); err != nil {
		t.Fatal(err)
	}

	wtRunner := &Runner{Dir: worktree}
	topLevel, err := wtRunner.TopLevelDir()
	if err != nil {
		t.Fatal(err)
	}
	if topLevel != worktree {
		t.Fatalf("TopLevelDir() = %q, want %q", topLevel, worktree)
	}

	gitDir, err := wtRunner.GitDir()
	if err != nil {
		t.Fatal(err)
	}
	gitCommonDir, err := wtRunner.GitCommonDir()
	if err != nil {
		t.Fatal(err)
	}
	if gitDir == gitCommonDir {
		t.Fatalf("linked worktree should not have gitDir == gitCommonDir (got %q)", gitDir)
	}
}

func setupGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	runner := &Runner{Dir: dir}
	if _, err := runner.Run("init", "-b", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("config", "user.name", "Test User"); err != nil {
		t.Fatal(err)
	}
	writeGitFile(t, dir, "README.md", "# repo\n")
	if _, err := runner.Run("add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("commit", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
	return dir
}

func setupGitRepoWithRemote(t *testing.T) (string, string) {
	t.Helper()

	remote := filepath.Join(t.TempDir(), "remote.git")
	if _, err := (&Runner{Dir: t.TempDir()}).Run("init", "--bare", remote); err != nil {
		t.Fatal(err)
	}

	repo := setupGitRepo(t)
	runner := &Runner{Dir: repo}
	if _, err := runner.Run("remote", "add", "origin", remote); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("push", "-u", "origin", "main"); err != nil {
		t.Fatal(err)
	}
	return repo, remote
}

func writeGitFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
