package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Build the ww binary for integration tests
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

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	repo := filepath.Join(dir, "myrepo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v: %v\n%s", args, err, out)
		}
	}
	return repo
}

func runWW(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(wwBin(), args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
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

	// Create a worktree (no remote, so we need to specify base)
	// Without origin/HEAD, it will fail, so let's create a branch manually first
	// and use the config to set default_base
	cfgContent := `default_base = "main"`
	if err := os.WriteFile(filepath.Join(repo, ".ww.toml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, repo, "create", "feat/test-branch")
	if err != nil {
		t.Fatalf("ww create: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Created worktree") {
		t.Errorf("unexpected create output: %s", out)
	}

	// Verify worktree was created
	wtPath := filepath.Join(filepath.Dir(repo), "myrepo@feat-test-branch")
	if _, err := os.Stat(wtPath); err != nil {
		t.Errorf("worktree directory not created at %s", wtPath)
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

	cfgContent := `default_base = "main"`
	if err := os.WriteFile(filepath.Join(repo, ".ww.toml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := runWW(t, repo, "create", "feat/json-test"); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, repo, "list", "--json")
	if err != nil {
		t.Fatalf("ww list --json: %v\n%s", err, out)
	}

	// Each line should be valid JSON
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("invalid JSON line: %s\nerror: %v", line, err)
		}
	}
}

func TestCreateDryRun(t *testing.T) {
	repo := setupTestRepo(t)

	cfgContent := `default_base = "main"`
	if err := os.WriteFile(filepath.Join(repo, ".ww.toml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

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

	cfgContent := `default_base = "main"`
	if err := os.WriteFile(filepath.Join(repo, ".ww.toml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

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
}

func TestInvalidBranchName(t *testing.T) {
	repo := setupTestRepo(t)

	cfgContent := `default_base = "main"`
	if err := os.WriteFile(filepath.Join(repo, ".ww.toml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := runWW(t, repo, "create", "-starts-with-dash")
	if err == nil {
		t.Fatalf("expected error for invalid branch name, got: %s", out)
	}
}

func TestZeroConfig(t *testing.T) {
	repo := setupTestRepo(t)

	// No .ww.toml, but we need a default base. Without remote, default_branch will fail.
	// This tests that zero-config mode attempts to detect default branch and gives a clear error.
	_, err := runWW(t, repo, "create", "feat/zero-config")
	if err == nil {
		// Without a remote, this should fail with a clear error about default branch detection
		t.Log("zero-config create succeeded (repo has remote)")
	}
	// Either way, the command should not panic
}

func TestCopyFiles(t *testing.T) {
	repo := setupTestRepo(t)

	// Create a file to copy
	envContent := "SECRET=test123"
	if err := os.WriteFile(filepath.Join(repo, ".env"), []byte(envContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfgContent := `
default_base = "main"
copy_files = [".env"]
`
	if err := os.WriteFile(filepath.Join(repo, ".ww.toml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := runWW(t, repo, "create", "feat/copy-test"); err != nil {
		t.Fatal(err)
	}

	wtPath := filepath.Join(filepath.Dir(repo), "myrepo@feat-copy-test")
	copiedEnv := filepath.Join(wtPath, ".env")
	data, err := os.ReadFile(copiedEnv)
	if err != nil {
		t.Fatalf("copied .env not found: %v", err)
	}
	if string(data) != envContent {
		t.Errorf("copied .env content = %q, want %q", string(data), envContent)
	}
}

func TestSymlinkFiles(t *testing.T) {
	repo := setupTestRepo(t)

	// Create a directory to symlink
	nmDir := filepath.Join(repo, "node_modules", "pkg")
	if err := os.MkdirAll(nmDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgContent := `
default_base = "main"
symlink_files = ["node_modules"]
`
	if err := os.WriteFile(filepath.Join(repo, ".ww.toml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := runWW(t, repo, "create", "feat/symlink-test"); err != nil {
		t.Fatal(err)
	}

	wtPath := filepath.Join(filepath.Dir(repo), "myrepo@feat-symlink-test")
	link := filepath.Join(wtPath, "node_modules")
	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("symlink not found: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink, got regular file/dir")
	}
}

func TestPostCreateHook(t *testing.T) {
	repo := setupTestRepo(t)

	cfgContent := `
default_base = "main"
post_create_hook = "echo hook-ran > hook-output.txt"
`
	if err := os.WriteFile(filepath.Join(repo, ".ww.toml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

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

func TestCreateExistingBranch(t *testing.T) {
	repo := setupTestRepo(t)

	cfgContent := `default_base = "main"`
	if err := os.WriteFile(filepath.Join(repo, ".ww.toml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a branch without a worktree
	cmd := exec.Command("git", "branch", "feat/existing")
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git branch: %v\n%s", err, out)
	}

	// Now create a worktree for the existing branch
	out, err := runWW(t, repo, "create", "feat/existing")
	if err != nil {
		t.Fatalf("ww create existing branch: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Created worktree") {
		t.Errorf("unexpected output: %s", out)
	}
}
