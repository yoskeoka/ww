package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestIsWorktreeRemoveSubmoduleError(t *testing.T) {
	err := errors.New("git worktree remove --force /tmp/repo@feat: exit status 128\nfatal: working trees containing submodules cannot be moved or removed")
	if !IsWorktreeRemoveSubmoduleError(err) {
		t.Fatal("expected submodule worktree remove failure to be recognized")
	}

	otherErr := errors.New("git worktree remove /tmp/repo@feat: exit status 128\nfatal: '/tmp/repo@feat' contains modified or untracked files")
	if IsWorktreeRemoveSubmoduleError(otherErr) {
		t.Fatal("dirty worktree failure should not be recognized as a submodule failure")
	}

	if IsWorktreeRemoveSubmoduleError(nil) {
		t.Fatal("nil error should not be recognized as a submodule failure")
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

func TestPatchEquivalentBranches(t *testing.T) {
	repo := setupGitRepo(t)
	runner := &Runner{Dir: repo}

	if _, err := runner.Run("checkout", "-b", "feat/squash"); err != nil {
		t.Fatal(err)
	}
	writeGitFile(t, repo, "squash-1.txt", "squash one\n")
	if _, err := runner.Run("add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("commit", "-m", "feat: squash source 1"); err != nil {
		t.Fatal(err)
	}
	writeGitFile(t, repo, "squash-2.txt", "squash two\n")
	if _, err := runner.Run("add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("commit", "-m", "feat: squash source 2"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("checkout", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("-c", "merge.ff=true", "merge", "--squash", "feat/squash"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("commit", "-m", "feat: squash merged"); err != nil {
		t.Fatal(err)
	}

	if _, err := runner.Run("checkout", "-b", "feat/rebased"); err != nil {
		t.Fatal(err)
	}
	writeGitFile(t, repo, "rebased.txt", "rebased\n")
	if _, err := runner.Run("add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("commit", "-m", "feat: rebased source"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("checkout", "main"); err != nil {
		t.Fatal(err)
	}
	writeGitFile(t, repo, "rebase-base.txt", "base shift\n")
	if _, err := runner.Run("add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("commit", "-m", "chore: rebase base shift"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("cherry-pick", "feat/rebased"); err != nil {
		t.Fatal(err)
	}

	if _, err := runner.Run("checkout", "-b", "feat/partial"); err != nil {
		t.Fatal(err)
	}
	writeGitFile(t, repo, "partial.txt", "integrated\n")
	if _, err := runner.Run("add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("commit", "-m", "feat: partial integrated"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("checkout", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("cherry-pick", "feat/partial"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("checkout", "feat/partial"); err != nil {
		t.Fatal(err)
	}
	writeGitFile(t, repo, "partial-extra.txt", "pending\n")
	if _, err := runner.Run("add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("commit", "-m", "feat: partial pending"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("checkout", "main"); err != nil {
		t.Fatal(err)
	}

	branches, err := runner.PatchEquivalentBranches("main", []string{"feat/squash", "feat/rebased", "feat/partial"})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(branches, "feat/squash") {
		t.Fatalf("PatchEquivalentBranches did not include feat/squash: %v", branches)
	}
	if !contains(branches, "feat/rebased") {
		t.Fatalf("PatchEquivalentBranches did not include feat/rebased: %v", branches)
	}
	if contains(branches, "feat/partial") {
		t.Fatalf("PatchEquivalentBranches should exclude feat/partial with extra work: %v", branches)
	}
}

func TestPatchEquivalentBranchesCachesBasePatchIDsByMergeBase(t *testing.T) {
	repo := setupGitRepo(t)
	runner := &Runner{Dir: repo}

	commitBranch := func(branch string) {
		t.Helper()
		if _, err := runner.Run("checkout", "-b", branch); err != nil {
			t.Fatal(err)
		}
		for i := 1; i <= 2; i++ {
			file := fmt.Sprintf("%s-%d.txt", branch[5:], i)
			writeGitFile(t, repo, file, fmt.Sprintf("%s change %d\\n", branch, i))
			if _, err := runner.Run("add", file); err != nil {
				t.Fatal(err)
			}
			if _, err := runner.Run("commit", "-m", fmt.Sprintf("%s change %d", branch, i)); err != nil {
				t.Fatal(err)
			}
		}
	}
	squashMerge := func(branch string) {
		t.Helper()
		if _, err := runner.Run("checkout", "main"); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Run("-c", "merge.ff=true", "merge", "--squash", branch); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Run("commit", "-m", "squash "+branch); err != nil {
			t.Fatal(err)
		}
	}

	// The first two branches share a merge-base; the third starts after their
	// squash merges and therefore requires its own cache entry.
	commitBranch("feat/squash-one")
	if _, err := runner.Run("checkout", "main"); err != nil {
		t.Fatal(err)
	}
	commitBranch("feat/squash-two")
	squashMerge("feat/squash-one")
	squashMerge("feat/squash-two")
	commitBranch("feat/squash-three")
	squashMerge("feat/squash-three")

	logPath := filepath.Join(t.TempDir(), "git-commands.log")
	gitBin := filepath.Join(t.TempDir(), "recording-git")
	if err := os.WriteFile(gitBin, []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" >> "+logPath+"\nexec git \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner.GitBin = gitBin
	branches, err := runner.PatchEquivalentBranches("main", []string{"feat/squash-one", "feat/squash-two", "feat/squash-three"})
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 3 {
		t.Fatalf("PatchEquivalentBranches = %v, want all squash branches", branches)
	}
	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(log), "rev-list "); got != 2 {
		t.Fatalf("rev-list calls = %d, want 2 for two merge-bases\\n%s", got, log)
	}
	if got := strings.Count(string(log), "show --format= --patch "); got != 4 {
		t.Fatalf("base commit patch-id calls = %d, want 4 cached base commits\\n%s", got, log)
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

func TestBranchMergeRefMissingTracking(t *testing.T) {
	repo := setupGitRepo(t)
	runner := &Runner{Dir: repo}

	mergeRef, err := runner.BranchMergeRef("main")
	if err != nil {
		t.Fatal(err)
	}
	if mergeRef != "" {
		t.Fatalf("BranchMergeRef(main) = %q, want empty", mergeRef)
	}
}

func TestHeuristicDefaultBranchPrefersLocalMainTrackingOriginMain(t *testing.T) {
	repo, _ := setupGitRepoWithRemote(t)
	runner := &Runner{Dir: repo}

	ref, ok, err := runner.HeuristicDefaultBranch()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("HeuristicDefaultBranch() = not found, want found")
	}
	if ref != "origin/main" {
		t.Fatalf("HeuristicDefaultBranch() = %q, want %q", ref, "origin/main")
	}
}

func TestHeuristicDefaultBranchFindsAnyLocalBranchTrackingOriginMain(t *testing.T) {
	repo, remote := setupGitRepoWithRemote(t)
	runner := &Runner{Dir: repo}

	if _, err := runner.Run("checkout", "-b", "dev"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("branch", "--set-upstream-to=origin/main", "dev"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("checkout", "--detach"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("branch", "--unset-upstream", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("update-ref", "-d", "refs/remotes/origin/HEAD"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("remote", "set-url", "origin", remote); err != nil {
		t.Fatal(err)
	}

	ref, ok, err := runner.HeuristicDefaultBranch()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("HeuristicDefaultBranch() = not found, want found")
	}
	if ref != "origin/main" {
		t.Fatalf("HeuristicDefaultBranch() = %q, want %q", ref, "origin/main")
	}
}

func TestHeuristicDefaultBranchFallsBackToRemoteHeadLookup(t *testing.T) {
	repo, _ := setupGitRepoWithRemote(t)
	runner := &Runner{Dir: repo}

	if _, err := runner.Run("branch", "--unset-upstream", "main"); err != nil {
		t.Fatal(err)
	}

	ref, ok, err := runner.HeuristicDefaultBranch()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("HeuristicDefaultBranch() = not found, want found")
	}
	if ref != "origin/main" {
		t.Fatalf("HeuristicDefaultBranch() = %q, want %q", ref, "origin/main")
	}
}

func TestHeuristicDefaultBranchRequiresLocalRemoteTrackingRef(t *testing.T) {
	repo, remote := setupGitRepoWithRemote(t)
	runner := &Runner{Dir: repo}

	if _, err := runner.Run("branch", "--unset-upstream", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("update-ref", "-d", "refs/remotes/origin/main"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("update-ref", "-d", "refs/remotes/origin/HEAD"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("remote", "set-url", "origin", remote); err != nil {
		t.Fatal(err)
	}

	ref, ok, err := runner.HeuristicDefaultBranch()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("HeuristicDefaultBranch() unexpectedly found %q without local remote-tracking ref", ref)
	}
	if ref != "" {
		t.Fatalf("HeuristicDefaultBranch() ref = %q, want empty", ref)
	}
}

func TestHeuristicDefaultBranchNotFound(t *testing.T) {
	repo := setupGitRepo(t)
	runner := &Runner{Dir: repo}

	ref, ok, err := runner.HeuristicDefaultBranch()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("HeuristicDefaultBranch() unexpectedly found %q", ref)
	}
	if ref != "" {
		t.Fatalf("HeuristicDefaultBranch() ref = %q, want empty", ref)
	}
}

func TestBranchTrackingConfig(t *testing.T) {
	repo, _ := setupGitRepoWithRemote(t)
	runner := &Runner{Dir: repo}

	if _, err := runner.Run("checkout", "-b", "dev"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("branch", "--set-upstream-to=origin/main", "dev"); err != nil {
		t.Fatal(err)
	}

	tracking, err := runner.BranchTrackingConfig()
	if err != nil {
		t.Fatal(err)
	}
	if tracking["main"].Remote != "origin" {
		t.Fatalf("tracking[main].Remote = %q, want %q", tracking["main"].Remote, "origin")
	}
	if tracking["main"].MergeRef != "refs/heads/main" {
		t.Fatalf("tracking[main].MergeRef = %q, want %q", tracking["main"].MergeRef, "refs/heads/main")
	}
	if tracking["dev"].Remote != "origin" {
		t.Fatalf("tracking[dev].Remote = %q, want %q", tracking["dev"].Remote, "origin")
	}
	if tracking["dev"].MergeRef != "refs/heads/main" {
		t.Fatalf("tracking[dev].MergeRef = %q, want %q", tracking["dev"].MergeRef, "refs/heads/main")
	}
}

func TestBranchRemotesUsesOneNULDelimitedConfigQuery(t *testing.T) {
	callsPath := filepath.Join(t.TempDir(), "git-calls")
	gitBin := filepath.Join(t.TempDir(), "fake-git.sh")
	script := fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >> %q
if [ "$1" = "config" ] && [ "$2" = "--null" ] && [ "$3" = "--get-regexp" ]; then
  printf 'branch.feat/one.remote\norigin\000branch.release/two.remote\nbackup\000branch.unrequested.remote\norigin\000'
  exit 0
fi
exit 1
`, callsPath)
	if err := os.WriteFile(gitBin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	runner := &Runner{GitBin: gitBin}
	remotes, err := runner.BranchRemotes([]string{
		"feat/one",
		"release/two",
		"local-only",
		"configured-without-merge",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"feat/one": "origin", "release/two": "backup"}
	if len(remotes) != len(want) {
		t.Fatalf("BranchRemotes() = %#v, want %#v", remotes, want)
	}
	for branch, remote := range want {
		if remotes[branch] != remote {
			t.Fatalf("BranchRemotes()[%q] = %q, want %q", branch, remotes[branch], remote)
		}
	}

	calls, err := os.ReadFile(callsPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(calls)); got != "config --null --get-regexp ^branch\\..*\\.remote$" {
		t.Fatalf("git invocation = %q, want one NUL-delimited remote query", got)
	}
}

func TestBranchRemotesParsesRealGitConfigOutput(t *testing.T) {
	repo, _ := setupGitRepoWithRemote(t)
	runner := &Runner{Dir: repo}
	if _, err := runner.Run("branch", "feat/one"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("branch", "release/two"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("config", "branch.feat/one.remote", "origin"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("config", "branch.release/two.remote", "backup"); err != nil {
		t.Fatal(err)
	}

	remotes, err := runner.BranchRemotes([]string{"feat/one", "release/two", "local-only"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"feat/one": "origin", "release/two": "backup"}
	if len(remotes) != len(want) {
		t.Fatalf("BranchRemotes() = %#v, want %#v", remotes, want)
	}
	for branch, remote := range want {
		if remotes[branch] != remote {
			t.Fatalf("BranchRemotes()[%q] = %q, want %q", branch, remotes[branch], remote)
		}
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
