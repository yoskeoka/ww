package integration_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/yoskeoka/ww/testutil"
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
		defer globalEnv.Terminate()
	}
	os.Exit(m.Run())
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
}

func TestListJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires Docker")
	}
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
		if !strings.Contains(line, `"path"`) {
			t.Errorf("JSON line missing 'path' field: %s", line)
		}
	}
}

func TestCreateDryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires Docker")
	}
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

func TestRemoveMainWorktreeDryRunRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires Docker")
	}
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
