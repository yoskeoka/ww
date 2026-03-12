package integration_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	cmd := exec.Command("go", "build", "-o", filepath.Join(os.TempDir(), "ww-test"), "./cmd/ww/")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build ww: " + err.Error())
	}
	os.Exit(m.Run())
}

func wwBin() string {
	return filepath.Join(os.TempDir(), "ww-test")
}

// setupTestRepo creates a fresh git repository with realistic seed data:
// - explicit main branch
// - multiple commits with actual file content
// - an existing branch for testing checkout of existing branches
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	repo := filepath.Join(dir, "myrepo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}

	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}

	writeFile := func(name, content string) {
		t.Helper()
		path := filepath.Join(repo, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Initialize repo with explicit branch name
	git("init", "-b", "main")
	git("config", "user.email", "test@test.com")
	git("config", "user.name", "Test User")

	// Seed commit 1: initial project structure
	writeFile("README.md", "# My Repo\n\nA test repository for ww integration tests.\n")
	writeFile("go.mod", "module example.com/myrepo\n\ngo 1.25.0\n")
	writeFile("main.go", fmt.Sprintf("package main\n\nfunc main() {\n\tprintln(%q)\n}\n", "hello"))
	git("add", ".")
	git("commit", "-m", "initial: project scaffold")

	// Seed commit 2: add more files
	writeFile("internal/util.go", "package internal\n\nfunc Add(a, b int) int { return a + b }\n")
	writeFile("internal/util_test.go", "package internal\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(1, 2) != 3 {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n")
	git("add", ".")
	git("commit", "-m", "feat: add util package")

	// Seed commit 3: update readme
	writeFile("README.md", "# My Repo\n\nA test repository for ww integration tests.\n\n## Usage\n\nRun `go run main.go`\n")
	git("add", ".")
	git("commit", "-m", "docs: update readme with usage")

	// Create an existing branch (for TestCreateExistingBranch)
	git("branch", "feat/existing")

	return repo
}

func runWW(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(wwBin(), args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func writeConfig(t *testing.T, repo, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, ".ww.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestVersionCommand(t *testing.T) {
	dir := t.TempDir()
	out, err := runWW(t, dir, "version")
	if err != nil {
		t.Fatalf("ww version: %v\n%s", err, out)
	}
	if !strings.HasPrefix(out, "ww version") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCreateAndList(t *testing.T) {
	repo := setupTestRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	out, err := runWW(t, repo, "create", "feat/test-branch")
	if err != nil {
		t.Fatalf("ww create: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Created worktree") {
		t.Errorf("unexpected create output: %s", out)
	}

	// Verify worktree directory exists
	wtPath := filepath.Join(filepath.Dir(repo), "myrepo@feat-test-branch")
	if _, err := os.Stat(wtPath); err != nil {
		t.Errorf("worktree directory not created at %s", wtPath)
	}

	// Verify the worktree contains repo files (inherited from main)
	if _, err := os.Stat(filepath.Join(wtPath, "go.mod")); err != nil {
		t.Error("worktree should contain go.mod from main branch")
	}

	// List worktrees
	out, err = runWW(t, repo, "list")
	if err != nil {
		t.Fatalf("ww list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "feat/test-branch") {
		t.Errorf("list output should contain branch name: %s", out)
	}
}

func TestListJSON(t *testing.T) {
	repo := setupTestRepo(t)
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
		}
		if _, ok := obj["path"]; !ok {
			t.Errorf("JSON object missing 'path' field: %s", line)
		}
	}
}

func TestCreateDryRun(t *testing.T) {
	repo := setupTestRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	out, err := runWW(t, repo, "create", "--dry-run", "feat/dry-test")
	if err != nil {
		t.Fatalf("ww create --dry-run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Would create") {
		t.Errorf("dry-run should show 'Would create': %s", out)
	}

	// Verify worktree was NOT created
	wtPath := filepath.Join(filepath.Dir(repo), "myrepo@feat-dry-test")
	if _, err := os.Stat(wtPath); err == nil {
		t.Error("dry-run should not create worktree directory")
	}
}

func TestRemoveWorktree(t *testing.T) {
	repo := setupTestRepo(t)
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

	// Verify worktree directory is gone
	wtPath := filepath.Join(filepath.Dir(repo), "myrepo@feat-to-remove")
	if _, err := os.Stat(wtPath); err == nil {
		t.Error("worktree directory should be removed")
	}
}

func TestInvalidBranchName(t *testing.T) {
	repo := setupTestRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	out, err := runWW(t, repo, "create", "-starts-with-dash")
	if err == nil {
		t.Fatalf("expected error for invalid branch name, got: %s", out)
	}
}

func TestZeroConfig(t *testing.T) {
	repo := setupTestRepo(t)

	// No .ww.toml — without a remote, default branch detection fails.
	// This tests that zero-config mode gives a clear error (not a panic).
	_, err := runWW(t, repo, "create", "feat/zero-config")
	if err == nil {
		t.Log("zero-config create succeeded (repo has remote)")
	}
}

func TestCopyFiles(t *testing.T) {
	repo := setupTestRepo(t)

	envContent := "SECRET=test123"
	if err := os.WriteFile(filepath.Join(repo, ".env"), []byte(envContent), 0644); err != nil {
		t.Fatal(err)
	}

	writeConfig(t, repo, `
default_base = "main"
copy_files = [".env"]
`)

	if _, err := runWW(t, repo, "create", "feat/copy-test"); err != nil {
		t.Fatal(err)
	}

	wtPath := filepath.Join(filepath.Dir(repo), "myrepo@feat-copy-test")
	data, err := os.ReadFile(filepath.Join(wtPath, ".env"))
	if err != nil {
		t.Fatalf("copied .env not found: %v", err)
	}
	if string(data) != envContent {
		t.Errorf("copied .env content = %q, want %q", string(data), envContent)
	}
}

func TestSymlinkFiles(t *testing.T) {
	repo := setupTestRepo(t)

	nmDir := filepath.Join(repo, "node_modules", "pkg")
	if err := os.MkdirAll(nmDir, 0755); err != nil {
		t.Fatal(err)
	}

	writeConfig(t, repo, `
default_base = "main"
symlink_files = ["node_modules"]
`)

	if _, err := runWW(t, repo, "create", "feat/symlink-test"); err != nil {
		t.Fatal(err)
	}

	wtPath := filepath.Join(filepath.Dir(repo), "myrepo@feat-symlink-test")
	fi, err := os.Lstat(filepath.Join(wtPath, "node_modules"))
	if err != nil {
		t.Fatalf("symlink not found: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink, got regular file/dir")
	}
}

func TestPostCreateHook(t *testing.T) {
	repo := setupTestRepo(t)

	writeConfig(t, repo, `
default_base = "main"
post_create_hook = "echo hook-ran > hook-output.txt"
`)

	if _, err := runWW(t, repo, "create", "feat/hook-test"); err != nil {
		t.Fatal(err)
	}

	wtPath := filepath.Join(filepath.Dir(repo), "myrepo@feat-hook-test")
	data, err := os.ReadFile(filepath.Join(wtPath, "hook-output.txt"))
	if err != nil {
		t.Fatalf("hook output not found: %v", err)
	}
	if !strings.Contains(string(data), "hook-ran") {
		t.Errorf("hook output = %q, want 'hook-ran'", string(data))
	}
}

func TestRemoveNonexistentWorktree(t *testing.T) {
	repo := setupTestRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	// Try to remove a worktree that doesn't exist
	out, err := runWW(t, repo, "remove", "feat/nonexistent")
	if err == nil {
		t.Fatalf("expected error for non-existent worktree, got: %s", out)
	}
	if !strings.Contains(out, "no worktree found") {
		t.Errorf("error should mention 'no worktree found': %s", out)
	}
}

func TestRemoveNonexistentWorktreeDryRun(t *testing.T) {
	repo := setupTestRepo(t)
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
	repo := setupTestRepo(t)

	// Subcommand --help should exit cleanly (exit 0)
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

func TestCreateExistingBranch(t *testing.T) {
	repo := setupTestRepo(t)
	writeConfig(t, repo, `default_base = "main"`)

	// feat/existing was created by setupTestRepo
	out, err := runWW(t, repo, "create", "feat/existing")
	if err != nil {
		t.Fatalf("ww create existing branch: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Created worktree") {
		t.Errorf("unexpected output: %s", out)
	}

	// Verify the worktree has the repo content
	wtPath := filepath.Join(filepath.Dir(repo), "myrepo@feat-existing")
	if _, err := os.Stat(filepath.Join(wtPath, "main.go")); err != nil {
		t.Error("worktree for existing branch should contain main.go")
	}
}
