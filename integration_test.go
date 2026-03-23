package integration_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/yoskeoka/ww/internal/testutil"
)

// globalEnv is the shared container environment for all integration tests.
// It is nil when running in short mode (go test -short).
var globalEnv *testutil.ContainerEnv

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		var err error
		globalEnv, err = testutil.NewContainerEnv(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: setup container env: %v\n", err)
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

func TestVersionCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires Docker")
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

func TestCreateAndList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires Docker")
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
		t.Skip("skipping: requires Docker")
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
		t.Skip("skipping: requires Docker")
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

func TestListStatusesAndCleanable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires Docker")
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

func TestCreateDryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires Docker")
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

func TestRemoveWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires Docker")
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

func TestInvalidBranchName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires Docker")
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
		t.Skip("skipping: requires Docker")
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
		t.Skip("skipping: requires Docker")
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

func TestCopyFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires Docker")
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
		t.Skip("skipping: requires Docker")
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
		t.Skip("skipping: requires Docker")
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

func TestRemoveMainWorktreeRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires Docker")
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
	return repo
}

func worktreePath(repo, branch string) string {
	return path.Join(path.Dir(repo), "myrepo@"+strings.ReplaceAll(branch, "/", "-"))
}

func TestRemoveMainWorktreeDryRunRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires Docker")
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
		t.Skip("skipping: requires Docker")
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
		t.Skip("skipping: requires Docker")
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
		t.Skip("skipping: requires Docker")
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
		t.Skip("skipping: requires Docker")
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
		t.Skip("skipping: requires Docker")
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
		t.Skip("skipping: requires Docker")
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

func TestRemoveForceCleanWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires Docker")
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
		t.Skip("skipping: requires Docker")
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
}

func TestCreateExistingPathRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires Docker")
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
		t.Skip("skipping: requires Docker")
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
		t.Skip("skipping: requires Docker")
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
