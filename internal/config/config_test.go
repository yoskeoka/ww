package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "" {
		t.Errorf("WorktreeDir = %q, want empty", cfg.WorktreeDir)
	}
	if cfg.DefaultBase != "" {
		t.Errorf("DefaultBase = %q, want empty", cfg.DefaultBase)
	}
	if len(cfg.CopyFiles) != 0 {
		t.Errorf("CopyFiles = %v, want empty", cfg.CopyFiles)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	content := `
worktree_dir = ".worktrees"
default_base = "origin/main"
copy_files = [".env"]
symlink_files = ["node_modules"]
post_create_hook = "npm install"
`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != ".worktrees" {
		t.Errorf("WorktreeDir = %q, want .worktrees", cfg.WorktreeDir)
	}
	if cfg.DefaultBase != "origin/main" {
		t.Errorf("DefaultBase = %q, want origin/main", cfg.DefaultBase)
	}
	if len(cfg.CopyFiles) != 1 || cfg.CopyFiles[0] != ".env" {
		t.Errorf("CopyFiles = %v, want [.env]", cfg.CopyFiles)
	}
	if cfg.PostCreateHook != "npm install" {
		t.Errorf("PostCreateHook = %q, want 'npm install'", cfg.PostCreateHook)
	}
}

func TestLoadSearchUpward(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	content := `worktree_dir = "found"`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(sub)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "found" {
		t.Errorf("WorktreeDir = %q, want found", cfg.WorktreeDir)
	}
}

// initGitRepo initializes a bare-minimum git repo at dir with one commit.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
		{"commit", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
}

func TestLoadFromMainWorktreeFallback(t *testing.T) {
	// Create a git repo with .ww.toml
	mainRepo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(mainRepo, 0755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, mainRepo)

	content := `worktree_dir = "from-main"`
	if err := os.WriteFile(filepath.Join(mainRepo, FileName), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a sibling worktree
	wtDir := filepath.Join(filepath.Dir(mainRepo), "repo@feat-x")
	cmd := exec.Command("git", "worktree", "add", "-b", "feat-x", wtDir, "HEAD")
	cmd.Dir = mainRepo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add failed: %v\n%s", err, out)
	}

	// Load config from the worktree — should find .ww.toml in main repo
	cfg, err := Load(wtDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "from-main" {
		t.Errorf("WorktreeDir = %q, want from-main", cfg.WorktreeDir)
	}
}

func TestUpwardSearchTakesPriorityOverMainWorktree(t *testing.T) {
	// Create a git repo
	mainRepo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(mainRepo, 0755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, mainRepo)

	// Put .ww.toml in main repo
	if err := os.WriteFile(filepath.Join(mainRepo, FileName), []byte(`worktree_dir = "from-main"`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a worktree
	wtDir := filepath.Join(filepath.Dir(mainRepo), "repo@feat-y")
	cmd := exec.Command("git", "worktree", "add", "-b", "feat-y", wtDir, "HEAD")
	cmd.Dir = mainRepo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add failed: %v\n%s", err, out)
	}

	// Put a different .ww.toml in the parent dir (upward search should find this first)
	parentDir := filepath.Dir(mainRepo)
	if err := os.WriteFile(filepath.Join(parentDir, FileName), []byte(`worktree_dir = "from-parent"`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(wtDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "from-parent" {
		t.Errorf("WorktreeDir = %q, want from-parent (upward search should win)", cfg.WorktreeDir)
	}
}

func TestFallbackGracefullyReturnsDefaultsOutsideGitRepo(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "" {
		t.Errorf("WorktreeDir = %q, want empty (default)", cfg.WorktreeDir)
	}
}
