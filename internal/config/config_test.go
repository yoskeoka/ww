package config

import (
	"os"
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
