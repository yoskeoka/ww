package integration_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yoskeoka/ww/internal/testutil"
)

// globalEnv is the shared host environment for all integration tests.
// It is nil when running in short mode (go test -short).
var globalEnv *testutil.HostEnv

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		var err error
		globalEnv, err = testutil.NewHostEnv(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: setup host env: %v\n", err)
			os.Exit(1)
		}
	}
	code := m.Run()
	if globalEnv != nil {
		globalEnv.Terminate()
	}
	os.Exit(code)
}

func setupRepo(t *testing.T) string {
	t.Helper()
	return testutil.SetupRepo(t, globalEnv, testutil.RepoOpts{})
}

func writeConfig(t *testing.T, repo, content string) {
	t.Helper()
	if err := globalEnv.WriteFile(path.Join(repo, ".ww.toml"), content); err != nil {
		t.Fatal(err)
	}
}

func runWW(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	return globalEnv.RunWW(dir, args...)
}

func runWWSplit(t *testing.T, dir string, args ...string) (string, string, error) {
	t.Helper()
	return globalEnv.RunWWSplit(dir, args...)
}

func runWWWithEnv(t *testing.T, dir string, extraEnv []string, args ...string) (string, error) {
	t.Helper()
	return globalEnv.RunWWWithEnv(dir, extraEnv, args...)
}

func runWWSplitWithEnv(t *testing.T, dir string, extraEnv []string, args ...string) (string, string, error) {
	t.Helper()
	return globalEnv.RunWWSplitWithEnv(dir, extraEnv, args...)
}

func TestVersionCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	dir, err := globalEnv.MkdirTemp("ww-ver")
	if err != nil {
		t.Fatal(err)
	}
	out, err := runWW(t, dir, "version")
	if err != nil {
		t.Fatalf("ww version: %v\n%s", err, out)
	}
	if !strings.HasPrefix(out, "ww version") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestVersionCommandJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	dir, err := globalEnv.MkdirTemp("ww-ver-json")
	if err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, dir, "version", "--json")
	if err != nil {
		t.Fatalf("ww version --json: %v\n%s", err, out)
	}

	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if got["version"] != "dev" {
		t.Fatalf("version = %q, want dev", got["version"])
	}
	if got["commit"] == "" {
		t.Fatal("commit should not be empty")
	}
}

func TestHelpIncludesInteractiveCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)

	out, err := runWW(t, repo, "--help")
	if err != nil {
		t.Fatalf("ww --help: %v\n%s", err, out)
	}
	if !strings.Contains(out, "i             Start interactive mode") {
		t.Fatalf("help output should list interactive command: %s", out)
	}
}

func TestInteractiveCommandRejectsJSONBeforeTTYCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)

	stdout, stderr, err := runWWSplit(t, repo, "i", "--json")
	if err == nil {
		t.Fatal("expected ww i --json to fail")
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("ww i --json should not write stdout, got: %s", stdout)
	}
	if !strings.Contains(stderr, "interactive mode does not support --json") {
		t.Fatalf("ww i --json stderr should mention JSON rejection: %s", stderr)
	}
	if strings.Contains(stderr, "requires a TTY") {
		t.Fatalf("ww i --json should reject before TTY validation: %s", stderr)
	}
}

func TestInteractiveHelpHidesJSONFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)

	out, err := runWW(t, repo, "i", "--help")
	if err != nil {
		t.Fatalf("ww i --help: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Start interactive mode") {
		t.Fatalf("ww i --help should describe the command: %s", out)
	}
	if !strings.Contains(out, "Usage:\n  ww i") {
		t.Fatalf("ww i --help should include simple usage: %s", out)
	}
	if !strings.Contains(out, "requires a TTY on stdin and stderr") {
		t.Fatalf("ww i --help should mention TTY requirement: %s", out)
	}
	if !strings.Contains(out, "--json is not supported; use standard ww commands for machine-readable output") {
		t.Fatalf("ww i --help should mention json guidance: %s", out)
	}
	if strings.Contains(out, "Usage of ww i:") {
		t.Fatalf("ww i --help should not use default pflag usage output: %s", out)
	}
}

func TestInteractiveCommandFailsWithoutTTY(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)

	stdout, stderr, err := runWWSplit(t, repo, "i")
	if err == nil {
		t.Fatal("expected ww i to fail without TTY")
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("ww i should not write stdout, got: %s", stdout)
	}
	if !strings.Contains(stderr, "interactive mode requires a TTY on stdin and stderr") {
		t.Fatalf("ww i stderr should mention TTY requirement: %s", stderr)
	}
}

func TestCreateAndList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	out, err := runWW(t, repo, "create", "feat/test-branch")
	if err != nil {
		t.Fatalf("ww create: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Created worktree") {
		t.Errorf("unexpected create output: %s", out)
	}

	wtPath := path.Join(path.Dir(repo), "myrepo@feat-test-branch")
	if !globalEnv.PathExists(wtPath) {
		t.Errorf("worktree directory not created at %s", wtPath)
	}
	if !globalEnv.PathExists(path.Join(wtPath, "go.mod")) {
		t.Error("worktree should contain go.mod from main branch")
	}

	out, err = runWW(t, repo, "list")
	if err != nil {
		t.Fatalf("ww list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "feat/test-branch") {
		t.Errorf("list output should contain branch name: %s", out)
	}
	if !strings.Contains(out, "(main worktree)") {
		t.Errorf("list output should mark the main worktree: %s", out)
	}
	if !strings.Contains(out, "STATUS") {
		t.Errorf("list output should include STATUS column: %s", out)
	}
}

func TestListJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	if _, err := runWW(t, repo, "create", "feat/json-test"); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, repo, "list", "--json")
	if err != nil {
		t.Fatalf("ww list --json: %v\n%s", err, out)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 NDJSON lines (main + worktree), got %d", len(lines))
	}
	for _, line := range lines {
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("invalid JSON line: %s\nerror: %v", line, err)
			continue
		}
		if _, ok := obj["path"]; !ok {
			t.Errorf("JSON object missing 'path' field: %s", line)
		}
		if _, ok := obj["repo"]; !ok {
			t.Errorf("JSON object missing 'repo' field: %s", line)
		}
		if _, ok := obj["status"]; !ok {
			t.Errorf("JSON object missing 'status' field: %s", line)
		}
	}
}

func TestListWorkspaceMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	ws := testutil.SetupNonGitWorkspace(t, globalEnv, testutil.WorkspaceOpts{NumRepos: 2})
	writeConfig(t, ws.RootDir, `default_base = "main"`)

	out, err := runWW(t, ws.RootDir, "list")
	if err != nil {
		t.Fatalf("ww list from workspace root: %v\n%s", err, out)
	}
	if !strings.Contains(out, "REPO") {
		t.Fatalf("workspace list should include REPO column: %s", out)
	}
	if !strings.Contains(out, "repo1") || !strings.Contains(out, "repo2") {
		t.Fatalf("workspace list should include both repos: %s", out)
	}
	if !strings.Contains(out, "STATUS") {
		t.Fatalf("workspace list should include STATUS column: %s", out)
	}
}

func TestListWorkspaceModeIgnoresHelperDirsAndChildSymlinks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	ws := testutil.SetupNonGitWorkspace(t, globalEnv, testutil.WorkspaceOpts{NumRepos: 2})
	writeConfig(t, ws.RootDir, `default_base = "main"`)

	helper := path.Join(ws.RootDir, ".claude")
	if err := globalEnv.MkdirAll(path.Join(helper, ".git", "gk")); err != nil {
		t.Fatal(err)
	}

	symlinkChild := path.Join(ws.RootDir, ".gemini")
	if err := os.Symlink(ws.RepoDirs[0], symlinkChild); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, ws.RootDir, "list")
	if err != nil {
		t.Fatalf("ww list from workspace root with helper dirs: %v\n%s", err, out)
	}
	if !strings.Contains(out, "repo1") || !strings.Contains(out, "repo2") {
		t.Fatalf("workspace list should include both real repos: %s", out)
	}
	if strings.Contains(out, ".claude") {
		t.Fatalf("workspace list should ignore helper dir with stray .git contents: %s", out)
	}
	if strings.Contains(out, ".gemini") {
		t.Fatalf("workspace list should ignore symlinked child entry: %s", out)
	}
}

func TestListStatusesAndCleanable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepoWithBareRemote(t)
	writeConfig(t, repo, `default_base = "main"`)

	if _, err := runWW(t, repo, "create", "feat/alpha"); err != nil {
		t.Fatalf("ww create feat/alpha: %v", err)
	}

	if _, err := runWW(t, repo, "create", "feat/beta"); err != nil {
		t.Fatalf("ww create feat/beta: %v", err)
	}
	staleWT := worktreePath(repo, "feat/beta")
	if err := globalEnv.WriteFile(path.Join(staleWT, "stale.txt"), "stale\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(staleWT, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(staleWT, "commit", "-m", "feat: stale"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "push", "-u", "origin", "feat/beta"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "push", "origin", ":feat/beta"); err != nil {
		t.Fatal(err)
	}

	if _, err := runWW(t, repo, "create", "feat/gamma"); err != nil {
		t.Fatalf("ww create feat/gamma: %v", err)
	}
	activeWT := worktreePath(repo, "feat/gamma")
	if err := globalEnv.WriteFile(path.Join(activeWT, "active.txt"), "active\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(activeWT, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(activeWT, "commit", "-m", "feat: active"); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, repo, "list")
	if err != nil {
		t.Fatalf("ww list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "feat/alpha") {
		t.Fatalf("list output should include merged branch: %s", out)
	}
	if !strings.Contains(out, "merged") {
		t.Fatalf("list output should include merged status: %s", out)
	}
	if !strings.Contains(out, "feat/beta") || !strings.Contains(out, "stale") {
		t.Fatalf("list output should include stale status: %s", out)
	}
	if !strings.Contains(out, "feat/gamma") || !strings.Contains(out, "active") {
		t.Fatalf("list output should include active status: %s", out)
	}

	out, err = runWW(t, repo, "list", "--cleanable")
	if err != nil {
		t.Fatalf("ww list --cleanable: %v\n%s", err, out)
	}
	if strings.Contains(out, "feat/gamma") {
		t.Fatalf("cleanable output should exclude active worktrees: %s", out)
	}
	if !strings.Contains(out, "feat/alpha") || !strings.Contains(out, "feat/beta") {
		t.Fatalf("cleanable output should include merged and stale worktrees: %s", out)
	}
}

func TestCleanDryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepoWithBareRemote(t)
	writeConfig(t, repo, `default_base = "main"`)

	if _, err := runWW(t, repo, "create", "feat/alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := runWW(t, repo, "create", "feat/beta"); err != nil {
		t.Fatal(err)
	}
	makeStaleWorktree(t, repo, "feat/beta", false)

	out, err := runWW(t, repo, "clean", "--dry-run")
	if err != nil {
		t.Fatalf("ww clean --dry-run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Would remove worktree at "+worktreePath(repo, "feat/alpha")) {
		t.Fatalf("dry-run output should mention merged worktree: %s", out)
	}
	if !strings.Contains(out, "Would remove worktree at "+worktreePath(repo, "feat/beta")) {
		t.Fatalf("dry-run output should mention stale worktree: %s", out)
	}
	if !globalEnv.PathExists(worktreePath(repo, "feat/alpha")) || !globalEnv.PathExists(worktreePath(repo, "feat/beta")) {
		t.Fatal("dry-run should not remove worktrees")
	}
}

func TestCleanRemovesCleanable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepoWithBareRemote(t)
	writeConfig(t, repo, `default_base = "main"`)

	if _, err := runWW(t, repo, "create", "feat/alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := runWW(t, repo, "create", "feat/beta"); err != nil {
		t.Fatal(err)
	}
	makeStaleWorktree(t, repo, "feat/beta", false)
	if _, err := runWW(t, repo, "create", "feat/gamma"); err != nil {
		t.Fatal(err)
	}
	activeWT := worktreePath(repo, "feat/gamma")
	if err := globalEnv.WriteFile(path.Join(activeWT, "active.txt"), "active\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(activeWT, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(activeWT, "commit", "-m", "feat: active"); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, repo, "clean")
	if err != nil {
		t.Fatalf("ww clean: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Deleted branch feat/alpha") {
		t.Fatalf("clean output should report deleted merged branch: %s", out)
	}
	if !strings.Contains(out, "warning: could not delete branch feat/beta") {
		t.Fatalf("clean output should warn when safe delete keeps a stale branch: %s", out)
	}
	if globalEnv.PathExists(worktreePath(repo, "feat/alpha")) || globalEnv.PathExists(worktreePath(repo, "feat/beta")) {
		t.Fatal("clean should remove merged and stale worktrees")
	}
	if !globalEnv.PathExists(activeWT) {
		t.Fatal("clean should leave active worktrees untouched")
	}
}

func TestCleanForceDirtyWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepoWithBareRemote(t)
	writeConfig(t, repo, `default_base = "main"`)

	if _, err := runWW(t, repo, "create", "feat/dirty"); err != nil {
		t.Fatal(err)
	}
	makeStaleWorktree(t, repo, "feat/dirty", true)

	out, err := runWW(t, repo, "clean", "--force")
	if err != nil {
		t.Fatalf("ww clean --force: %v\n%s", err, out)
	}
	if globalEnv.PathExists(worktreePath(repo, "feat/dirty")) {
		t.Fatal("force clean should remove dirty stale worktree")
	}
	if _, err := globalEnv.Git(repo, "rev-parse", "--verify", "refs/heads/feat/dirty"); err == nil {
		t.Fatal("force clean should delete the branch")
	}
}

func TestCleanJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepoWithBareRemote(t)
	writeConfig(t, repo, `default_base = "main"`)

	if _, err := runWW(t, repo, "create", "feat/alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := runWW(t, repo, "create", "feat/beta"); err != nil {
		t.Fatal(err)
	}
	makeStaleWorktree(t, repo, "feat/beta", false)

	out, err := runWW(t, repo, "clean", "--json")
	if err != nil {
		t.Fatalf("ww clean --json: %v\n%s", err, out)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 NDJSON lines, got %d: %s", len(lines), out)
	}
	for _, line := range lines {
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Fatalf("invalid JSON line %q: %v", line, err)
		}
		if obj["repo"] == "" || obj["branch"] == "" || obj["status"] == "" {
			t.Fatalf("clean json should include repo, branch, and status: %s", line)
		}
		if obj["removed"] != true {
			t.Fatalf("clean json should report worktree removal: %s", line)
		}
		if obj["status"] == "merged" && obj["branch_deleted"] != true {
			t.Fatalf("merged clean json should report branch deletion: %s", line)
		}
		if obj["status"] == "stale" && obj["branch_deleted"] != false {
			t.Fatalf("stale clean json should preserve branch_deleted=false with safe delete: %s", line)
		}
	}
}

func TestCleanWorkspaceModeFromRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	ws := testutil.SetupNonGitWorkspace(t, globalEnv, testutil.WorkspaceOpts{NumRepos: 2})
	writeConfig(t, ws.RootDir, `default_base = "main"`)

	if _, err := runWW(t, ws.RepoDirs[0], "create", "feat/repo1-clean"); err != nil {
		t.Fatal(err)
	}
	if _, err := runWW(t, ws.RepoDirs[1], "create", "feat/repo2-clean"); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, ws.RepoDirs[0], "clean")
	if err != nil {
		t.Fatalf("ww clean from repo in workspace: %v\n%s", err, out)
	}
	if globalEnv.PathExists(path.Join(ws.RootDir, ".worktrees", "repo1@feat-repo1-clean")) {
		t.Fatal("workspace clean should remove repo1 cleanable worktree")
	}
	if globalEnv.PathExists(path.Join(ws.RootDir, ".worktrees", "repo2@feat-repo2-clean")) {
		t.Fatal("workspace clean should remove repo2 cleanable worktree")
	}
}

func TestCleanWorkspaceModeFromRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	ws := testutil.SetupNonGitWorkspace(t, globalEnv, testutil.WorkspaceOpts{NumRepos: 2})
	writeConfig(t, ws.RootDir, `default_base = "main"`)

	if _, err := runWW(t, ws.RepoDirs[0], "create", "feat/root1-clean"); err != nil {
		t.Fatal(err)
	}
	if _, err := runWW(t, ws.RepoDirs[1], "create", "feat/root2-clean"); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, ws.RootDir, "clean")
	if err != nil {
		t.Fatalf("ww clean from workspace root: %v\n%s", err, out)
	}
	if globalEnv.PathExists(path.Join(ws.RootDir, ".worktrees", "repo1@feat-root1-clean")) {
		t.Fatal("workspace-root clean should remove repo1 cleanable worktree")
	}
	if globalEnv.PathExists(path.Join(ws.RootDir, ".worktrees", "repo2@feat-root2-clean")) {
		t.Fatal("workspace-root clean should remove repo2 cleanable worktree")
	}
}

func TestCleanEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	out, err := runWW(t, repo, "clean")
	if err != nil {
		t.Fatalf("ww clean empty case: %v\n%s", err, out)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("clean with no cleanable worktrees should produce no output: %q", out)
	}
}

func TestCleanContinuesAfterFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepoWithBareRemote(t)
	writeConfig(t, repo, `default_base = "main"`)

	if _, err := runWW(t, repo, "create", "feat/alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := runWW(t, repo, "create", "feat/beta"); err != nil {
		t.Fatal(err)
	}
	makeStaleWorktree(t, repo, "feat/beta", true)

	out, err := runWW(t, repo, "clean")
	if err == nil {
		t.Fatalf("expected ww clean to fail when a dirty cleanable worktree cannot be removed: %s", out)
	}
	if !strings.Contains(out, "Deleted branch feat/alpha") {
		t.Fatalf("clean should still remove successful candidates before failing: %s", out)
	}
	if !strings.Contains(out, "Failed to clean feat/beta") {
		t.Fatalf("clean should report the failed worktree: %s", out)
	}
	if globalEnv.PathExists(worktreePath(repo, "feat/alpha")) == true {
		t.Fatal("successful cleanable worktrees should still be removed")
	}
	if !globalEnv.PathExists(worktreePath(repo, "feat/beta")) {
		t.Fatal("failed dirty worktree should remain in place")
	}
}

func TestCleanContinuesAfterSubmoduleRemovalFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)
	addCommittedSubmodule(t, repo)
	skipIfGitAllowsSubmoduleWorktreeRemove(t, repo)

	if _, err := runWW(t, repo, "create", "feat/submodule-clean"); err != nil {
		t.Fatal(err)
	}
	submoduleWT := worktreePath(repo, "feat/submodule-clean")
	initSubmodules(t, submoduleWT)

	if _, err := runWW(t, repo, "create", "feat/clean-after-submodule"); err != nil {
		t.Fatal(err)
	}
	cleanWT := worktreePath(repo, "feat/clean-after-submodule")

	out, err := runWW(t, repo, "clean", "--force")
	if err == nil {
		t.Fatalf("expected ww clean --force to fail when one cleanable worktree contains submodules, got:\n%s", out)
	}
	if !strings.Contains(out, "Failed to clean feat/submodule-clean") {
		t.Fatalf("clean should report the failed submodule worktree:\n%s", out)
	}
	if !strings.Contains(out, "Git cannot remove worktrees containing submodules") {
		t.Fatalf("clean failure should include guided remediation:\n%s", out)
	}
	if !strings.Contains(out, "Deleted branch feat/clean-after-submodule") {
		t.Fatalf("clean should continue to later cleanable worktrees:\n%s", out)
	}
	if !globalEnv.PathExists(submoduleWT) {
		t.Fatal("failed submodule worktree should remain in place")
	}
	if globalEnv.PathExists(cleanWT) {
		t.Fatal("later cleanable worktree should be removed")
	}
}

func TestCreateDryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	out, err := runWW(t, repo, "create", "--dry-run", "feat/dry-test")
	if err != nil {
		t.Fatalf("ww create --dry-run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Would create") {
		t.Errorf("dry-run should show 'Would create': %s", out)
	}

	wtPath := path.Join(path.Dir(repo), "myrepo@feat-dry-test")
	if globalEnv.PathExists(wtPath) {
		t.Error("dry-run should not create worktree directory")
	}
}

func TestCreateQuiet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	stdout, stderr, err := runWWSplit(t, repo, "create", "-q", "feat/quiet-test")
	if err != nil {
		t.Fatalf("ww create -q: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	wantPath := worktreePath(repo, "feat/quiet-test")
	if strings.TrimSpace(stdout) != wantPath {
		t.Fatalf("quiet stdout = %q, want %q", stdout, wantPath+"\n")
	}
	if strings.Contains(stderr, "Created worktree") {
		t.Fatalf("quiet mode should not print human-readable success output to stderr: %s", stderr)
	}
}

func TestCreateQuietDryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	stdout, stderr, err := runWWSplit(t, repo, "create", "-q", "--dry-run", "feat/quiet-dry-run")
	if err != nil {
		t.Fatalf("ww create -q --dry-run: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	wantPath := worktreePath(repo, "feat/quiet-dry-run")
	if strings.TrimSpace(stdout) != wantPath {
		t.Fatalf("quiet dry-run stdout = %q, want %q", stdout, wantPath+"\n")
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("quiet dry-run stderr should be empty, got: %s", stderr)
	}
	if globalEnv.PathExists(wantPath) {
		t.Fatal("quiet dry-run should not create the worktree directory")
	}
}

func TestCreateQuietJSONTakesPrecedence(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	stdout, stderr, err := runWWSplit(t, repo, "create", "-q", "--json", "feat/quiet-json")
	if err != nil {
		t.Fatalf("ww create -q --json: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	var obj map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &obj); err != nil {
		t.Fatalf("quiet json output is not valid JSON: %v\n%s", err, stdout)
	}
	if obj["path"] != worktreePath(repo, "feat/quiet-json") {
		t.Fatalf("json path = %v, want %q", obj["path"], worktreePath(repo, "feat/quiet-json"))
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("quiet json stderr should be empty, got: %s", stderr)
	}
}

func TestCreateQuietSendsHookOutputToStderr(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, "default_base = \"main\"\npost_create_hook = \"echo hook-ran\"\n")

	stdout, stderr, err := runWWSplit(t, repo, "create", "-q", "feat/quiet-hook")
	if err != nil {
		t.Fatalf("ww create -q: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	wantPath := worktreePath(repo, "feat/quiet-hook")
	if strings.TrimSpace(stdout) != wantPath {
		t.Fatalf("quiet hook stdout = %q, want %q", stdout, wantPath+"\n")
	}
	if !strings.Contains(stderr, "hook-ran") {
		t.Fatalf("quiet hook should route hook output to stderr: %s", stderr)
	}
	if strings.Contains(stderr, "Running post_create_hook:") {
		t.Fatalf("quiet hook should suppress human-readable hook announcement: %s", stderr)
	}
}

func TestCdNoArg(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	if _, err := runWW(t, repo, "create", "feat/cd-default"); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runWWSplit(t, repo, "cd")
	if err != nil {
		t.Fatalf("ww cd: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if strings.TrimSpace(stdout) != worktreePath(repo, "feat/cd-default") {
		t.Fatalf("ww cd stdout = %q, want %q", stdout, worktreePath(repo, "feat/cd-default")+"\n")
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("ww cd stderr should be empty, got: %s", stderr)
	}
}

func TestCdNamed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	if _, err := runWW(t, repo, "create", "feat/cd-alpha"); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runWWSplit(t, repo, "cd", "refs/heads/feat/cd-alpha")
	if err != nil {
		t.Fatalf("ww cd refs/heads/...: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if strings.TrimSpace(stdout) != worktreePath(repo, "feat/cd-alpha") {
		t.Fatalf("ww cd named stdout = %q, want %q", stdout, worktreePath(repo, "feat/cd-alpha")+"\n")
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("ww cd named stderr should be empty, got: %s", stderr)
	}
}

func TestCdJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	if _, err := runWW(t, repo, "create", "feat/cd-json"); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runWWSplit(t, repo, "cd", "--json", "feat/cd-json")
	if err != nil {
		t.Fatalf("ww cd --json: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	var obj map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &obj); err != nil {
		t.Fatalf("cd json output is not valid JSON: %v\n%s", err, stdout)
	}
	if obj["path"] != worktreePath(repo, "feat/cd-json") {
		t.Fatalf("cd json path = %v, want %q", obj["path"], worktreePath(repo, "feat/cd-json"))
	}
	if obj["branch"] != "feat/cd-json" {
		t.Fatalf("cd json branch = %v, want %q", obj["branch"], "feat/cd-json")
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("cd json stderr should be empty, got: %s", stderr)
	}
}

func TestCdWithRepoFlagFromWorkspaceRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	ws := testutil.SetupNonGitWorkspace(t, globalEnv, testutil.WorkspaceOpts{NumRepos: 2})
	writeConfig(t, ws.RootDir, `default_base = "main"`)

	if _, err := runWW(t, ws.RootDir, "create", "feat/shared-branch", "--repo", "repo1"); err != nil {
		t.Fatal(err)
	}
	if _, err := runWW(t, ws.RootDir, "create", "feat/shared-branch", "--repo", "repo2"); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runWWSplit(t, ws.RootDir, "cd", "--repo", "repo2", "feat/shared-branch")
	if err != nil {
		t.Fatalf("ww cd --repo repo2: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	wantPath := workspaceWorktreePath(ws.RootDir, "repo2", "feat/shared-branch")
	if strings.TrimSpace(stdout) != wantPath {
		t.Fatalf("ww cd --repo stdout = %q, want %q", stdout, wantPath+"\n")
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("ww cd --repo stderr should be empty, got: %s", stderr)
	}
}

func TestCdFindsJustCreatedWorktreeFromGitBackedWorkspaceRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	outer, err := globalEnv.MkdirTemp("ww-git-backed-workspace")
	if err != nil {
		t.Fatal(err)
	}

	workspaceRoot := path.Join(outer, "workspace")
	initEmptyRepo(t, workspaceRoot)
	writeConfig(t, workspaceRoot, `default_base = "main"`)

	repo1 := path.Join(workspaceRoot, "repo1")
	repo2 := path.Join(workspaceRoot, "repo2")
	initEmptyRepo(t, repo1)
	initEmptyRepo(t, repo2)

	if _, err := runWW(t, workspaceRoot, "create", "feat/workspace-root-cd"); err != nil {
		t.Fatalf("ww create from git-backed workspace root: %v", err)
	}

	stdout, stderr, err := runWWSplit(t, workspaceRoot, "cd", "feat/workspace-root-cd")
	if err != nil {
		t.Fatalf("ww cd from git-backed workspace root: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	wantPath := workspaceWorktreePath(workspaceRoot, "workspace", "feat/workspace-root-cd")
	if strings.TrimSpace(stdout) != wantPath {
		t.Fatalf("ww cd from git-backed workspace root stdout = %q, want %q", stdout, wantPath+"\n")
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("ww cd from git-backed workspace root stderr should be empty, got: %s", stderr)
	}
}

func TestCdAbsorbsParallelCreateRaceForNamedLookup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	ws := testutil.SetupNonGitWorkspace(t, globalEnv, testutil.WorkspaceOpts{NumRepos: 2})
	writeConfig(t, ws.RootDir, `default_base = "main"`)

	const branch = "plan/parallel-create-cd-race"
	syncDir := t.TempDir()
	missMarker := filepath.Join(syncDir, "cd-miss.marker")
	retryRelease := filepath.Join(syncDir, "cd-retry.release")
	cdEnv := []string{
		"WW_TEST_CD_NAMED_LOOKUP_RETRY_COUNT=20",
		"WW_TEST_CD_NAMED_LOOKUP_RETRY_INTERVAL_MS=20",
		"WW_TEST_CD_NAMED_LOOKUP_MISS_MARKER=" + missMarker,
		"WW_TEST_CD_NAMED_LOOKUP_RETRY_RELEASE=" + retryRelease,
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	var stdout string
	var stderr string
	var err error
	var createOut string
	var createErr error
	go func() {
		defer wg.Done()
		<-start
		stdout, stderr, err = runWWSplitWithEnv(t, ws.RootDir, cdEnv, "cd", "--repo", "repo2", branch)
	}()

	go func() {
		defer wg.Done()
		<-start
		if waitErr := waitForPath(missMarker); waitErr != nil {
			createErr = waitErr
			return
		}
		createOut, createErr = runWWWithEnv(t, ws.RootDir, nil, "create", "--repo", "repo2", branch)
		if writeErr := os.WriteFile(retryRelease, []byte("release\n"), 0644); writeErr != nil {
			t.Errorf("write retry release marker: %v", writeErr)
		}
	}()

	close(start)
	wg.Wait()

	if createErr != nil {
		t.Fatalf("ww create in parallel goroutine: %v\n%s", createErr, createOut)
	}
	if err != nil {
		t.Fatalf("ww cd during parallel create: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	wantPath := workspaceWorktreePath(ws.RootDir, "repo2", branch)
	if strings.TrimSpace(stdout) != wantPath {
		t.Fatalf("ww cd stdout = %q, want trimmed path %q", stdout, wantPath)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("ww cd stderr should be empty, got: %s", stderr)
	}
}

func waitForPath(target string) error {
	for i := 0; i < 500; i++ {
		if _, err := os.Stat(target); err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %s", target)
}

func TestCdErrorsWithoutSecondaryWorktrees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	stdout, stderr, err := runWWSplit(t, repo, "cd")
	if err == nil {
		t.Fatalf("expected ww cd to fail without secondary worktrees, got stdout=%q stderr=%q", stdout, stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("ww cd error should not write stdout, got: %s", stdout)
	}
	if !strings.Contains(stderr, "no secondary worktrees found") {
		t.Fatalf("ww cd error should mention missing secondary worktrees: %s", stderr)
	}
}

func TestCreateWithRepoFlagFromWorkspaceRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	ws := testutil.SetupNonGitWorkspace(t, globalEnv, testutil.WorkspaceOpts{NumRepos: 2})
	writeConfig(t, ws.RootDir, `default_base = "main"`)

	out, err := runWW(t, ws.RootDir, "create", "feat/repo-flag", "--repo", "repo2")
	if err != nil {
		t.Fatalf("ww create --repo from workspace root: %v\n%s", err, out)
	}

	wtPath := workspaceWorktreePath(ws.RootDir, "repo2", "feat/repo-flag")
	if !globalEnv.PathExists(wtPath) {
		t.Fatalf("expected workspace worktree at %s", wtPath)
	}
	if globalEnv.PathExists(workspaceWorktreePath(ws.RootDir, "repo1", "feat/repo-flag")) {
		t.Fatal("create --repo should not create a worktree in another repo")
	}
}

func TestRemoveWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	if _, err := runWW(t, repo, "create", "feat/to-remove"); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, repo, "remove", "feat/to-remove")
	if err != nil {
		t.Fatalf("ww remove: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Removed worktree") {
		t.Errorf("unexpected remove output: %s", out)
	}

	wtPath := path.Join(path.Dir(repo), "myrepo@feat-to-remove")
	if globalEnv.PathExists(wtPath) {
		t.Error("worktree directory should be removed")
	}
}

func TestRemoveWithRepoFlagFromWorkspaceRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	ws := testutil.SetupNonGitWorkspace(t, globalEnv, testutil.WorkspaceOpts{NumRepos: 2})
	writeConfig(t, ws.RootDir, `default_base = "main"`)

	if _, err := runWW(t, ws.RootDir, "create", "feat/remove-flag", "--repo", "repo2"); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, ws.RootDir, "remove", "feat/remove-flag", "--repo", "repo2")
	if err != nil {
		t.Fatalf("ww remove --repo from workspace root: %v\n%s", err, out)
	}

	wtPath := workspaceWorktreePath(ws.RootDir, "repo2", "feat/remove-flag")
	if globalEnv.PathExists(wtPath) {
		t.Fatalf("expected workspace worktree to be removed at %s", wtPath)
	}
}

func TestRemoveWithRepoFlagUnknownRepoFromWorkspaceRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	ws := testutil.SetupNonGitWorkspace(t, globalEnv, testutil.WorkspaceOpts{NumRepos: 1})
	writeConfig(t, ws.RootDir, `default_base = "main"`)

	// Create a worktree in the known repo so that branch resolution itself works.
	if _, err := runWW(t, ws.RootDir, "create", "feat/remove-unknown", "--repo", "repo1"); err != nil {
		t.Fatal(err)
	}

	// Attempt to remove using a repo name that does not exist in the workspace.
	out, err := runWW(t, ws.RootDir, "remove", "feat/remove-unknown", "--repo", "does-not-exist")
	if err == nil {
		t.Fatalf("expected ww remove with unknown --repo to fail, got output:\n%s", out)
	}
	if !strings.Contains(out, `repo "does-not-exist" not found in workspace`) {
		t.Fatalf("expected unknown repo error, got output:\n%s", out)
	}
}

func TestRemoveWithRepoFlagOutsideWorkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	// Create a worktree in a standalone repo (not a ww workspace).
	if _, err := runWW(t, repo, "create", "feat/remove-outside"); err != nil {
		t.Fatal(err)
	}

	// Attempt to remove using --repo where it should not be valid (outside workspace).
	out, err := runWW(t, repo, "remove", "feat/remove-outside", "--repo", "some-repo")
	if err == nil {
		t.Fatalf("expected ww remove --repo outside workspace to fail, got output:\n%s", out)
	}
}
func TestInvalidBranchName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	out, err := runWW(t, repo, "create", "-starts-with-dash")
	if err == nil {
		t.Fatalf("expected error for invalid branch name, got: %s", out)
	}
}

func TestZeroConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)

	_, err := runWW(t, repo, "create", "feat/zero-config")
	if err == nil {
		t.Log("zero-config create succeeded (repo has remote)")
	}
}

func TestNonGitDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	dir, err := globalEnv.MkdirTemp("ww-nongit")
	if err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, dir, "list")
	if err == nil {
		t.Fatalf("expected error in non-git dir, got: %s", out)
	}
	if !strings.Contains(out, "not a git repository") {
		t.Errorf("error should mention 'not a git repository': %s", out)
	}
}

func TestCreateWithRepoFlagUnknownRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	ws := testutil.SetupNonGitWorkspace(t, globalEnv, testutil.WorkspaceOpts{NumRepos: 2})
	writeConfig(t, ws.RootDir, `default_base = "main"`)

	out, err := runWW(t, ws.RootDir, "create", "feat/unknown", "--repo", "missing")
	if err == nil {
		t.Fatalf("expected error for unknown repo, got: %s", out)
	}
	if !strings.Contains(out, `repo "missing" not found in workspace`) {
		t.Fatalf("expected unknown repo error, got: %s", out)
	}
}

func TestCreateWithRepoFlagOutsideWorkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	out, err := runWW(t, repo, "create", "feat/outside", "--repo", "myrepo")
	if err == nil {
		t.Fatalf("expected error for --repo outside workspace, got: %s", out)
	}
	if !strings.Contains(out, "--repo can only be used inside a detected workspace") {
		t.Fatalf("expected outside-workspace error, got: %s", out)
	}
}

func TestCopyFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)

	if err := globalEnv.WriteFile(path.Join(repo, ".env"), "SECRET=test123"); err != nil {
		t.Fatal(err)
	}

	writeConfig(t, repo, "default_base = \"main\"\ncopy_files = [\".env\"]\n")

	if _, err := runWW(t, repo, "create", "feat/copy-test"); err != nil {
		t.Fatal(err)
	}

	wtPath := path.Join(path.Dir(repo), "myrepo@feat-copy-test")
	data, err := globalEnv.ReadFile(path.Join(wtPath, ".env"))
	if err != nil {
		t.Fatalf("copied .env not found: %v", err)
	}
	expected := "SECRET=test123"
	normalized := strings.TrimRight(data, "\r\n")
	if normalized != expected {
		t.Errorf("copied .env content = %q (normalized %q), want %q", data, normalized, expected)
	}
}

func TestSymlinkFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)

	if err := globalEnv.MkdirAll(path.Join(repo, "node_modules", "pkg")); err != nil {
		t.Fatal(err)
	}

	writeConfig(t, repo, "default_base = \"main\"\nsymlink_files = [\"node_modules\"]\n")

	if _, err := runWW(t, repo, "create", "feat/symlink-test"); err != nil {
		t.Fatal(err)
	}

	wtPath := path.Join(path.Dir(repo), "myrepo@feat-symlink-test")
	if !globalEnv.IsSymlink(path.Join(wtPath, "node_modules")) {
		t.Error("expected symlink for node_modules, got regular file/dir")
	}
}

func TestPostCreateHook(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)

	writeConfig(t, repo, "default_base = \"main\"\npost_create_hook = \"echo hook-ran > hook-output.txt\"\n")

	if _, err := runWW(t, repo, "create", "feat/hook-test"); err != nil {
		t.Fatal(err)
	}

	wtPath := path.Join(path.Dir(repo), "myrepo@feat-hook-test")
	data, err := globalEnv.ReadFile(path.Join(wtPath, "hook-output.txt"))
	if err != nil {
		t.Fatalf("hook output not found: %v", err)
	}
	if !strings.Contains(data, "hook-ran") {
		t.Errorf("hook output = %q, want 'hook-ran'", data)
	}
}

func TestPostCreateHookAnnouncesCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)

	writeConfig(t, repo, "default_base = \"main\"\npost_create_hook = \"echo hook-ran\"\n")

	out, err := runWW(t, repo, "create", "feat/hook-announce-test")
	if err != nil {
		t.Fatalf("ww create: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Running post_create_hook: echo hook-ran") {
		t.Errorf("create output should announce post_create_hook: %s", out)
	}
	if !strings.Contains(out, "hook-ran") {
		t.Errorf("create output should include hook output: %s", out)
	}
	announcementIdx := strings.Index(out, "Running post_create_hook:")
	hookOutputIdx := strings.Index(out, "hook-ran")
	if announcementIdx >= 0 && hookOutputIdx >= 0 && announcementIdx >= hookOutputIdx {
		t.Errorf("hook announcement should appear before hook output; got:\n%s", out)
	}
}

func TestRemoveMainWorktreeRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	out, err := runWW(t, repo, "remove", "main")
	if err == nil {
		t.Fatalf("expected error when removing main worktree, got: %s", out)
	}
	if !strings.Contains(out, "cannot remove the main worktree") {
		t.Errorf("error should say 'cannot remove the main worktree': %s", out)
	}
}

func setupRepoWithBareRemote(t *testing.T) string {
	t.Helper()

	repo := setupRepo(t)
	root, err := globalEnv.MkdirTemp("ww-bare-remote")
	if err != nil {
		t.Fatal(err)
	}
	remote := path.Join(root, "origin.git")
	if _, err := globalEnv.Git(root, "init", "--bare", remote); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "remote", "set-url", "origin", remote); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "push", "-u", "origin", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "symbolic-ref", "--delete", "refs/remotes/origin/HEAD"); err != nil {
		t.Fatal(err)
	}
	return repo
}

func mustRemoteURL(t *testing.T, repo string) string {
	t.Helper()
	out, err := globalEnv.Git(repo, "remote", "get-url", "origin")
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(out)
}

func makeStaleWorktree(t *testing.T, repo, branch string, dirty bool) {
	t.Helper()

	wtPath := worktreePath(repo, branch)
	if err := globalEnv.WriteFile(path.Join(wtPath, "stale.txt"), "stale\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(wtPath, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(wtPath, "commit", "-m", "feat: stale"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "push", "-u", "origin", branch); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "push", "origin", ":"+branch); err != nil {
		t.Fatal(err)
	}
	if dirty {
		if err := globalEnv.WriteFile(path.Join(wtPath, "dirty.txt"), "dirty\n"); err != nil {
			t.Fatal(err)
		}
	}
}

func addCommittedSubmodule(t *testing.T, repo string) {
	t.Helper()

	root, err := globalEnv.MkdirTemp("ww-submodule-source")
	if err != nil {
		t.Fatal(err)
	}
	submoduleRepo := path.Join(root, "submodule")
	if err := globalEnv.MkdirAll(submoduleRepo); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(submoduleRepo, "init", "-b", "main"); err != nil {
		t.Fatal(err)
	}
	if err := globalEnv.WriteFile(path.Join(submoduleRepo, "README.md"), "# submodule\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(submoduleRepo, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(submoduleRepo, "commit", "-m", "initial submodule"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "-c", "protocol.file.allow=always", "submodule", "add", submoduleRepo, "vendor/submodule"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "commit", "-m", "add submodule"); err != nil {
		t.Fatal(err)
	}
}

func initSubmodules(t *testing.T, worktree string) {
	t.Helper()

	if _, err := globalEnv.Git(worktree, "-c", "protocol.file.allow=always", "submodule", "update", "--init", "--recursive"); err != nil {
		t.Fatal(err)
	}
}

func skipIfGitAllowsSubmoduleWorktreeRemove(t *testing.T, repo string) {
	t.Helper()

	branch := "feat/submodule-remove-probe"
	if _, err := runWW(t, repo, "create", branch); err != nil {
		t.Fatal(err)
	}
	wtPath := worktreePath(repo, branch)
	initSubmodules(t, wtPath)

	out, err := globalEnv.Git(repo, "worktree", "remove", "--force", wtPath)
	if err == nil {
		t.Skipf("installed Git removes submodule-containing worktrees without the target failure; probe output: %s", out)
	}
	if !strings.Contains(out, "working trees containing submodules cannot be moved or removed") {
		t.Fatalf("raw git submodule removal probe failed with an unexpected error:\n%s", out)
	}
	if removeErr := os.RemoveAll(wtPath); removeErr != nil {
		t.Fatal(removeErr)
	}
	if _, err := globalEnv.Git(repo, "worktree", "prune"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "branch", "-D", branch); err != nil {
		t.Fatal(err)
	}
}

func worktreePath(repo, branch string) string {
	return path.Join(path.Dir(repo), "myrepo@"+strings.ReplaceAll(branch, "/", "-"))
}

func workspaceWorktreePath(root, repo, branch string) string {
	return path.Join(root, ".worktrees", repo+"@"+strings.ReplaceAll(branch, "/", "-"))
}

func TestRemoveMainWorktreeDryRunRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	out, err := runWW(t, repo, "remove", "--dry-run", "main")
	if err == nil {
		t.Fatalf("expected error when dry-run removing main worktree, got: %s", out)
	}
	if !strings.Contains(out, "cannot remove the main worktree") {
		t.Errorf("error should say 'cannot remove the main worktree': %s", out)
	}
}

func TestRemoveNonexistentWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	out, err := runWW(t, repo, "remove", "feat/nonexistent")
	if err == nil {
		t.Fatalf("expected error for non-existent worktree, got: %s", out)
	}
	if !strings.Contains(out, "no worktree found") {
		t.Errorf("error should mention 'no worktree found': %s", out)
	}
}

func TestRemoveNonexistentWorktreeDryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	out, err := runWW(t, repo, "remove", "--dry-run", "feat/nonexistent")
	if err == nil {
		t.Fatalf("expected error for non-existent worktree dry-run, got: %s", out)
	}
	if !strings.Contains(out, "no worktree found") {
		t.Errorf("error should mention 'no worktree found': %s", out)
	}
}

func TestHelpFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)

	out, err := runWW(t, repo, "remove", "--help")
	if err != nil {
		t.Fatalf("--help should exit 0, got error: %v\n%s", err, out)
	}
	if strings.Contains(out, "pflag") {
		t.Errorf("--help output should not expose pflag internals: %s", out)
	}
	if !strings.Contains(out, "--keep-branch") {
		t.Errorf("--help should show available flags: %s", out)
	}
}

func TestRunFromWorktreeDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	out, err := runWW(t, repo, "create", "feat/first")
	if err != nil {
		t.Fatalf("ww create: %v\n%s", err, out)
	}

	wtPath := path.Join(path.Dir(repo), "myrepo@feat-first")
	writeConfig(t, path.Dir(repo), `default_base = "main"`)

	out, err = runWW(t, wtPath, "create", "feat/second")
	if err != nil {
		t.Fatalf("ww create from worktree: %v\n%s", err, out)
	}

	secondWtPath := path.Join(path.Dir(repo), "myrepo@feat-second")
	if !globalEnv.PathExists(secondWtPath) {
		t.Errorf("second worktree should be at %s (sibling of main repo)", secondWtPath)
	}

	out, err = runWW(t, wtPath, "list")
	if err != nil {
		t.Fatalf("ww list from worktree: %v\n%s", err, out)
	}
	if !strings.Contains(out, "(main worktree)") {
		t.Errorf("list from worktree should mark main worktree: %s", out)
	}
	if !strings.Contains(out, "feat/first") || !strings.Contains(out, "feat/second") {
		t.Errorf("list from worktree should show all worktrees: %s", out)
	}
}

func TestConfigFallbackFromWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	out, err := runWW(t, repo, "create", "feat/fallback-test")
	if err != nil {
		t.Fatalf("ww create: %v\n%s", err, out)
	}

	wtPath := path.Join(path.Dir(repo), "myrepo@feat-fallback-test")

	out, err = runWW(t, wtPath, "list")
	if err != nil {
		t.Fatalf("ww list from worktree (config fallback): %v\n%s", err, out)
	}
	if !strings.Contains(out, "feat/fallback-test") {
		t.Errorf("list should show the worktree branch: %s", out)
	}
}

func TestCreateExistingBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	out, err := runWW(t, repo, "create", "feat/existing")
	if err != nil {
		t.Fatalf("ww create existing branch: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Created worktree") {
		t.Errorf("unexpected output: %s", out)
	}

	wtPath := path.Join(path.Dir(repo), "myrepo@feat-existing")
	if !globalEnv.PathExists(path.Join(wtPath, "main.go")) {
		t.Error("worktree for existing branch should contain main.go")
	}
}

func TestCreateGuessRemoteRemoteOnlyBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepoWithBareRemote(t)
	writeConfig(t, repo, `default_base = "main"`)

	if _, err := globalEnv.Git(repo, "checkout", "-b", "feat/remote-only"); err != nil {
		t.Fatal(err)
	}
	if err := globalEnv.WriteFile(path.Join(repo, "remote-only.txt"), "remote only\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "commit", "-m", "feat: remote only"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "push", "-u", "origin", "feat/remote-only"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "checkout", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "branch", "-D", "feat/remote-only"); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, repo, "create", "--guess-remote", "feat/remote-only")
	if err != nil {
		t.Fatalf("ww create --guess-remote: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Created worktree") {
		t.Fatalf("unexpected output: %s", out)
	}

	wtPath := worktreePath(repo, "feat/remote-only")
	if !globalEnv.PathExists(path.Join(wtPath, "remote-only.txt")) {
		t.Fatalf("worktree should contain remote-only.txt: %s", wtPath)
	}
	upstream, err := globalEnv.Git(wtPath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		t.Fatalf("worktree upstream lookup: %v\n%s", err, upstream)
	}
	if strings.TrimSpace(upstream) != "origin/feat/remote-only" {
		t.Fatalf("upstream = %q, want %q", strings.TrimSpace(upstream), "origin/feat/remote-only")
	}
}

func TestCreateGuessRemoteFetchesOriginBeforeCheckout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepoWithBareRemote(t)
	writeConfig(t, repo, `default_base = "main"`)

	peer := setupRepo(t)
	if _, err := globalEnv.Git(peer, "remote", "set-url", "origin", mustRemoteURL(t, repo)); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(peer, "checkout", "-b", "feat/from-peer"); err != nil {
		t.Fatal(err)
	}
	if err := globalEnv.WriteFile(path.Join(peer, "from-peer.txt"), "from peer\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(peer, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(peer, "commit", "-m", "feat: from peer"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(peer, "push", "-u", "origin", "feat/from-peer"); err != nil {
		t.Fatal(err)
	}

	if _, err := globalEnv.Git(repo, "rev-parse", "--verify", "refs/remotes/origin/feat/from-peer"); err == nil {
		t.Fatal("expected origin/feat/from-peer to be absent before ww create fetches origin")
	}

	out, err := runWW(t, repo, "create", "--guess-remote", "feat/from-peer")
	if err != nil {
		t.Fatalf("ww create --guess-remote after peer push: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Created worktree") {
		t.Fatalf("unexpected output: %s", out)
	}

	wtPath := worktreePath(repo, "feat/from-peer")
	if !globalEnv.PathExists(path.Join(wtPath, "from-peer.txt")) {
		t.Fatalf("worktree should contain fetched branch content: %s", wtPath)
	}
}

func TestRemoveForceCleanWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	if _, err := runWW(t, repo, "create", "feat/force-clean"); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, repo, "remove", "--force", "feat/force-clean")
	if err != nil {
		t.Fatalf("ww remove --force (clean): %v\n%s", err, out)
	}
	if !strings.Contains(out, "Removed worktree") {
		t.Errorf("unexpected output: %s", out)
	}

	wtPath := path.Join(path.Dir(repo), "myrepo@feat-force-clean")
	if globalEnv.PathExists(wtPath) {
		t.Error("worktree directory should be removed")
	}
}

func TestRemoveForceDirtyWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	if _, err := runWW(t, repo, "create", "feat/force-dirty"); err != nil {
		t.Fatal(err)
	}

	wtPath := path.Join(path.Dir(repo), "myrepo@feat-force-dirty")
	if err := globalEnv.WriteFile(path.Join(wtPath, "dirty.txt"), "uncommitted"); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, repo, "remove", "feat/force-dirty")
	if err == nil {
		t.Fatalf("expected error removing dirty worktree without --force, got: %s", out)
	}

	out, err = runWW(t, repo, "remove", "--force", "feat/force-dirty")
	if err != nil {
		t.Fatalf("ww remove --force (dirty): %v\n%s", err, out)
	}
	if !strings.Contains(out, "Removed worktree") {
		t.Errorf("unexpected output: %s", out)
	}
	if globalEnv.PathExists(wtPath) {
		t.Error("dirty worktree directory should be removed with --force")
	}
	if _, err := globalEnv.Git(repo, "rev-parse", "--verify", "refs/heads/feat/force-dirty"); err == nil {
		t.Error("force remove should delete the branch with git branch -D")
	}
}

func TestRemoveForceSubmoduleWorktreeReportsGuidedRemediation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)
	addCommittedSubmodule(t, repo)
	skipIfGitAllowsSubmoduleWorktreeRemove(t, repo)

	if _, err := runWW(t, repo, "create", "feat/submodule-remove"); err != nil {
		t.Fatal(err)
	}
	wtPath := worktreePath(repo, "feat/submodule-remove")
	initSubmodules(t, wtPath)

	out, err := runWW(t, repo, "remove", "--force", "feat/submodule-remove")
	if err == nil {
		t.Fatalf("expected ww remove --force to fail for submodule worktree, got:\n%s", out)
	}
	for _, want := range []string{
		"Git cannot remove worktrees containing submodules",
		"Target worktree: " + wtPath,
		"Manual cleanup permanently deletes uncommitted work",
		"rm -rf",
		"git -C",
		"worktree prune",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("submodule removal diagnostic missing %q:\n%s", want, out)
		}
	}
	if !globalEnv.PathExists(wtPath) {
		t.Fatal("submodule worktree should remain after guided remediation failure")
	}
}

func TestCreateExistingPathRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	if _, err := runWW(t, repo, "create", "feat/dup-test"); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, repo, "create", "feat/dup-test")
	if err == nil {
		t.Fatalf("expected error for existing path, got: %s", out)
	}
	if !strings.Contains(out, "worktree already exists at") {
		t.Errorf("error should say 'worktree already exists at': %s", out)
	}
}

func TestWorkspaceCreateUsesCentralizedWorktreeDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	ws := testutil.SetupNonGitWorkspace(t, globalEnv, testutil.WorkspaceOpts{NumRepos: 2})
	writeConfig(t, ws.RootDir, `default_base = "main"`)

	out, err := runWW(t, ws.RepoDirs[0], "create", "feat/workspace-path")
	if err != nil {
		t.Fatalf("ww create in workspace: %v\n%s", err, out)
	}

	wtPath := path.Join(ws.RootDir, ".worktrees", "repo1@feat-workspace-path")
	if !globalEnv.PathExists(wtPath) {
		t.Fatalf("workspace worktree path not created at %s", wtPath)
	}
}

func TestNonGitWorkspaceRootRejectsWithoutRepoSelection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	ws := testutil.SetupNonGitWorkspace(t, globalEnv, testutil.WorkspaceOpts{NumRepos: 2})
	writeConfig(t, ws.RootDir, `default_base = "main"`)

	out, err := runWW(t, ws.RootDir, "list")
	if err != nil {
		t.Fatalf("expected list to succeed from non-git workspace root: %v\n%s", err, out)
	}
	if !strings.Contains(out, "REPO") || !strings.Contains(out, "repo1") || !strings.Contains(out, "repo2") {
		t.Fatalf("workspace-root list should include repo columns and both repos: %s", out)
	}
}

func TestListUsesNearestContainingWorkspaceRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	outer, err := globalEnv.MkdirTemp("ww-meta-workspace")
	if err != nil {
		t.Fatal(err)
	}
	initEmptyRepo(t, outer)

	otherRepo := path.Join(outer, "other")
	initEmptyRepo(t, otherRepo)

	workspaceRoot := path.Join(outer, "workspace")
	initEmptyRepo(t, workspaceRoot)
	writeConfig(t, workspaceRoot, `default_base = "main"`)

	repo1 := path.Join(workspaceRoot, "repo1")
	repo2 := path.Join(workspaceRoot, "repo2")
	initEmptyRepo(t, repo1)
	initEmptyRepo(t, repo2)

	out, err := runWW(t, repo1, "list")
	if err != nil {
		t.Fatalf("ww list from nested workspace repo: %v\n%s", err, out)
	}
	if !strings.Contains(out, "REPO") || !strings.Contains(out, "STATUS") {
		t.Fatalf("workspace list should include REPO and STATUS columns: %s", out)
	}
	if !strings.Contains(out, "workspace") || !strings.Contains(out, "repo1") || !strings.Contains(out, "repo2") {
		t.Fatalf("workspace list should resolve the nearest containing workspace root, got:\n%s", out)
	}
	if strings.Contains(out, "other") {
		t.Fatalf("workspace list should not include repos from the grandparent workspace, got:\n%s", out)
	}
}

func TestListUnknownStatusNoRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	// Create a repo with no remote and no default_base config.
	repo := testutil.SetupRepo(t, globalEnv, testutil.RepoOpts{Name: "no-remote-repo"})
	// No writeConfig — no default_base, no remote → base detection fails.

	// Create a worktree using an explicit branch (must exist already).
	if _, err := globalEnv.Git(repo, "branch", "feat/test-unknown"); err != nil {
		t.Fatal(err)
	}
	wtPath := path.Join(path.Dir(repo), "no-remote-repo@feat-test-unknown")
	if _, err := globalEnv.Git(repo, "worktree", "add", wtPath, "feat/test-unknown"); err != nil {
		t.Fatal(err)
	}

	// ww list should succeed (not error) with unknown status.
	out, err := runWW(t, repo, "list")
	if err != nil {
		t.Fatalf("ww list should succeed without remote: %v\n%s", err, out)
	}
	if !strings.Contains(out, "unknown(base-detect-failed)") {
		t.Errorf("expected unknown(base-detect-failed) in output, got:\n%s", out)
	}
	if !strings.Contains(out, "active") {
		t.Errorf("expected main worktree to show active, got:\n%s", out)
	}
}

func initEmptyRepo(t *testing.T, dir string) {
	t.Helper()
	if err := globalEnv.MkdirAll(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(dir, "init", "-b", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(dir, "commit", "--allow-empty", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
}

func TestListUnknownStatusJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := testutil.SetupRepo(t, globalEnv, testutil.RepoOpts{Name: "no-remote-json"})

	if _, err := globalEnv.Git(repo, "branch", "feat/json-unknown"); err != nil {
		t.Fatal(err)
	}
	wtPath := path.Join(path.Dir(repo), "no-remote-json@feat-json-unknown")
	if _, err := globalEnv.Git(repo, "worktree", "add", wtPath, "feat/json-unknown"); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, repo, "list", "--json")
	if err != nil {
		t.Fatalf("ww list --json should succeed without remote: %v\n%s", err, out)
	}

	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Fatalf("invalid JSON line: %s", line)
		}

		isMain, _ := obj["main"].(bool)
		status, _ := obj["status"].(string)
		statusDetail, _ := obj["status_detail"].(string)

		if isMain {
			if status != "active" {
				t.Errorf("main worktree status = %q, want active", status)
			}
			if statusDetail != "" {
				t.Errorf("main worktree status_detail = %q, want empty", statusDetail)
			}
		} else {
			if status != "unknown" {
				t.Errorf("non-main worktree status = %q, want unknown", status)
			}
			if statusDetail != "base-detect-failed" {
				t.Errorf("non-main worktree status_detail = %q, want base-detect-failed", statusDetail)
			}
		}
	}
}

func TestHeuristicBaseResolutionListAndClean(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepoWithBareRemote(t)

	mergedWT := worktreePath(repo, "feat/merged")
	if _, err := globalEnv.Git(repo, "checkout", "-b", "feat/merged"); err != nil {
		t.Fatal(err)
	}
	if err := globalEnv.WriteFile(path.Join(repo, "merged.txt"), "merged\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "commit", "-m", "feat: merged"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "checkout", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "merge", "--ff-only", "feat/merged"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "push", "origin", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "worktree", "add", mergedWT, "feat/merged"); err != nil {
		t.Fatal(err)
	}

	staleWT := worktreePath(repo, "feat/stale")
	if _, err := globalEnv.Git(repo, "checkout", "-b", "feat/stale"); err != nil {
		t.Fatal(err)
	}
	if err := globalEnv.WriteFile(path.Join(repo, "stale.txt"), "stale\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "commit", "-m", "feat: stale"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "push", "-u", "origin", "feat/stale"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "push", "origin", ":feat/stale"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "checkout", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "worktree", "add", staleWT, "feat/stale"); err != nil {
		t.Fatal(err)
	}

	activeWT := worktreePath(repo, "feat/active")
	if _, err := globalEnv.Git(repo, "checkout", "-b", "feat/active"); err != nil {
		t.Fatal(err)
	}
	if err := globalEnv.WriteFile(path.Join(repo, "active.txt"), "active\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "commit", "-m", "feat: active"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "checkout", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := globalEnv.Git(repo, "worktree", "add", activeWT, "feat/active"); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, repo, "list", "--json")
	if err != nil {
		t.Fatalf("ww list --json should succeed with heuristic base: %v\n%s", err, out)
	}

	seenStatuses := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Fatalf("invalid JSON line: %s", line)
		}
		branch, _ := obj["branch"].(string)
		status, _ := obj["status"].(string)
		statusDetail, _ := obj["status_detail"].(string)
		if statusDetail != "heuristic-base" {
			t.Fatalf("branch %q status_detail = %q, want heuristic-base", branch, statusDetail)
		}
		seenStatuses[branch] = status
	}

	if seenStatuses["main"] != "active" {
		t.Fatalf("main status = %q, want active", seenStatuses["main"])
	}
	if seenStatuses["feat/merged"] != "merged" {
		t.Fatalf("feat/merged status = %q, want merged", seenStatuses["feat/merged"])
	}
	if seenStatuses["feat/stale"] != "stale" {
		t.Fatalf("feat/stale status = %q, want stale", seenStatuses["feat/stale"])
	}
	if seenStatuses["feat/active"] != "active" {
		t.Fatalf("feat/active status = %q, want active", seenStatuses["feat/active"])
	}

	out, err = runWW(t, repo, "list", "--cleanable", "--json")
	if err != nil {
		t.Fatalf("ww list --cleanable --json should succeed with heuristic base: %v\n%s", err, out)
	}
	if !strings.Contains(out, `"branch":"feat/merged"`) || !strings.Contains(out, `"branch":"feat/stale"`) {
		t.Fatalf("cleanable output should include merged and stale heuristic worktrees: %s", out)
	}
	if strings.Contains(out, `"branch":"feat/active"`) {
		t.Fatalf("cleanable output should exclude active heuristic worktrees: %s", out)
	}

	out, err = runWW(t, repo, "clean")
	if err != nil {
		t.Fatalf("ww clean should succeed with heuristic base: %v\n%s", err, out)
	}
	if globalEnv.PathExists(mergedWT) || globalEnv.PathExists(staleWT) {
		t.Fatal("ww clean should remove heuristic merged/stale worktrees")
	}
	if !globalEnv.PathExists(activeWT) {
		t.Fatal("ww clean should preserve active heuristic worktrees")
	}
}

func TestListCleanableExcludesUnknown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := testutil.SetupRepo(t, globalEnv, testutil.RepoOpts{Name: "no-remote-cleanable"})

	if _, err := globalEnv.Git(repo, "branch", "feat/not-cleanable"); err != nil {
		t.Fatal(err)
	}
	wtPath := path.Join(path.Dir(repo), "no-remote-cleanable@feat-not-cleanable")
	if _, err := globalEnv.Git(repo, "worktree", "add", wtPath, "feat/not-cleanable"); err != nil {
		t.Fatal(err)
	}

	// --cleanable should not include unknown worktrees.
	out, err := runWW(t, repo, "list", "--cleanable", "--json")
	if err != nil {
		t.Fatalf("ww list --cleanable --json should succeed: %v\n%s", err, out)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected no cleanable worktrees in JSON output, got:\n%s", out)
	}
}

func TestCleanIgnoresUnknownWorktrees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := testutil.SetupRepo(t, globalEnv, testutil.RepoOpts{Name: "no-remote-clean"})

	if _, err := globalEnv.Git(repo, "branch", "feat/safe-from-clean"); err != nil {
		t.Fatal(err)
	}
	wtPath := path.Join(path.Dir(repo), "no-remote-clean@feat-safe-from-clean")
	if _, err := globalEnv.Git(repo, "worktree", "add", wtPath, "feat/safe-from-clean"); err != nil {
		t.Fatal(err)
	}

	// ww clean should succeed with no output (nothing to clean).
	out, err := runWW(t, repo, "clean")
	if err != nil {
		t.Fatalf("ww clean should succeed: %v\n%s", err, out)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected no output from clean, got:\n%s", out)
	}

	// Worktree should still exist.
	if !globalEnv.PathExists(wtPath) {
		t.Errorf("worktree should not have been cleaned: %s", wtPath)
	}
}
